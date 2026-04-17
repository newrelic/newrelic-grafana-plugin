import {
  DataSourceInstanceSettings,
  CoreApp,
  ScopedVars,
  MetricFindValue,
  FieldType,
  SupplementaryQueryType,
  SupplementaryQueryOptions,
} from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { lastValueFrom } from 'rxjs';

import { NewRelicQuery, NewRelicDataSourceOptions } from './types';
import { validateNrqlQuery } from './utils/validation';
import { logger } from './utils/logger';

/**
 * New Relic data source implementation
 * Handles query execution and template variable substitution
 */
export class DataSource extends DataSourceWithBackend<NewRelicQuery, NewRelicDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<NewRelicDataSourceOptions>) {
    super(instanceSettings);
    logger.info('New Relic data source initialized', {
      id: instanceSettings.id,
      name: instanceSettings.name,
      region: instanceSettings.jsonData?.region,
    });
  }

  /**
   * Returns the default query configuration for new queries
   * @param app - The Grafana application context
   * @returns Default query configuration
   */
  getDefaultQuery(app: CoreApp): Partial<NewRelicQuery> {
    const defaultQuery = 'SELECT count(*) FROM Transaction SINCE 1 hour ago';
    
    logger.debug('Creating default query', { app, query: defaultQuery });
    
    return {
      queryText: defaultQuery,
      refId: 'A',
    };
  }

  /**
   * Executes a NRQL query and returns results for Grafana dashboard variables.
   * Called by Grafana when a "Query" type variable runs its query.
   */
  async metricFindQuery(query: string, options?: any): Promise<MetricFindValue[]> {
    if (!query || query.trim().length === 0) {
      return [];
    }

    const request = {
      targets: [
        {
          refId: 'variable',
          queryText: query,
          useGrafanaTime: false,
        } as NewRelicQuery,
      ],
      range: options?.range,
      requestId: 'variable-query',
      interval: '1m',
      intervalMs: 60000,
      scopedVars: options?.scopedVars || {},
      timezone: 'UTC',
      app: 'dashboard',
      startTime: Date.now(),
    } as any;

    try {
      const response = await lastValueFrom(super.query(request));

      if (!response?.data?.length) {
        return [];
      }

      const values: MetricFindValue[] = [];

      for (const frame of response.data) {
        if (!frame.fields?.length) {
          continue;
        }

        // Skip time/number fields — find string/data fields for variable values
        for (const field of frame.fields) {
          if (field.type === FieldType.time) {
            continue;
          }

          const fieldValues = field.values?.toArray?.() ?? Array.from(field.values ?? []);

          for (const val of fieldValues) {
            if (val == null) {
              continue;
            }
            const strVal = String(val);
            // Expand comma-separated uniques() results
            if (fieldValues.length === 1 && typeof val === 'string' && val.includes(', ')) {
              const parts = val.split(', ').map((v: string) => v.trim()).filter(Boolean);
              for (const part of parts) {
                values.push({ text: part, value: part });
              }
            } else {
              values.push({ text: strVal, value: strVal });
            }
          }

          // Use the first non-time field only
          break;
        }
      }

      return values;
    } catch (error) {
      logger.error('Variable query failed', error as Error);
      return [];
    }
  }

  /**
   * Applies template variables to the query
   * @param query - The query to process
   * @param scopedVars - Template variables to substitute
   * @returns Query with template variables substituted
   */
  applyTemplateVariables(query: NewRelicQuery, scopedVars: ScopedVars): NewRelicQuery {
    try {
      // Apply template variable substitution
      const processedQueryText = getTemplateSrv().replace(query.queryText, scopedVars);
      
      // Validate the processed query
      const validation = validateNrqlQuery(processedQueryText);
      if (!validation.isValid) {
        logger.warn('Query validation failed after template substitution', {
          refId: query.refId,
          error: validation.message,
        });
      }

      const result = {
        ...query,
        queryText: processedQueryText,
      };

      logger.debug('Template variables applied', {
        refId: query.refId,
        hasVariables: Object.keys(scopedVars).length > 0,
      });

      return result;
    } catch (error) {
      logger.error('Error applying template variables', error as Error, {
        refId: query.refId,
      });
      
      // Return the original query if template processing fails
      return query;
    }
  }

  /**
   * Filters queries to determine which should be executed
   * @param query - The query to filter
   * @returns True if the query should be executed, false otherwise
   */
  filterQuery(query: NewRelicQuery): boolean {
    try {
      // Check if query text exists and is not empty
      if (!query.queryText || query.queryText.trim().length === 0) {
        logger.debug('Query filtered out: empty query text', { refId: query.refId });
        return false;
      }

      // Validate the query
      const validation = validateNrqlQuery(query.queryText);
      if (!validation.isValid) {
        logger.warn('Query filtered out: validation failed', {
          refId: query.refId,
          error: validation.message,
        });
        return false;
      }

      logger.debug('Query passed filtering', { refId: query.refId });
      return true;
    } catch (error) {
      logger.error('Error filtering query', error as Error, { refId: query.refId });
      return false;
    }
  }

  // ── Logs support ──────────────────────────────────────────────────

  /** Regex matching "FROM Log" as a standalone event type (not "FROM LogMessage") */
  private static LOG_QUERY_REGEX = /\bFROM\s+Log\b/i;

  /**
   * Checks if the NRQL query targets the Log event type.
   */
  private isLogQuery(query: string): boolean {
    return DataSource.LOG_QUERY_REGEX.test(query);
  }

  /**
   * Returns the supplementary query types this datasource supports.
   * Enables the log volume histogram in Grafana Explore.
   */
  getSupportedSupplementaryQueryTypes(): SupplementaryQueryType[] {
    return [SupplementaryQueryType.LogsVolume];
  }

  /**
   * Generates a supplementary query for log volume histograms.
   * Preserves WHERE filters from the original log query.
   */
  getSupplementaryQuery(
    options: SupplementaryQueryOptions,
    originalQuery: NewRelicQuery
  ): NewRelicQuery | undefined {
    if (options.type !== SupplementaryQueryType.LogsVolume) {
      return undefined;
    }

    if (!originalQuery.queryText || !this.isLogQuery(originalQuery.queryText)) {
      return undefined;
    }

    // Extract WHERE clause from the original query, preserving user's filters
    const whereMatch = originalQuery.queryText.match(
      /\bWHERE\b(.+?)(?=\s*\b(?:SINCE|UNTIL|LIMIT|TIMESERIES|FACET|ORDER\s+BY|COMPARE\s+WITH)\b|$)/i
    );
    const whereClause = whereMatch ? ` WHERE${whereMatch[1]}` : '';

    return {
      ...originalQuery,
      refId: `${originalQuery.refId}-logs-volume`,
      queryText: `SELECT count(*) FROM Log${whereClause} TIMESERIES`,
    };
  }

  // ── Connection test ──────────────────────────────────────────────

  /**
   * Tests the data source connection
   * @returns Promise resolving to connection test result
   */
  async testDatasource() {
    try {
      logger.info('Testing data source connection');
      
      // Call the backend health check endpoint
      const response = await this.getResource('health');
      
      if (response?.status === 'OK') {
        logger.info('Data source connection test successful: ' + (response.message || 'Connected'));
        return {
          status: 'success',
          message: response.message || '✅ Successfully connected to New Relic!',
        };
      } else {
        logger.error('Data source connection test failed: ' + (response?.message || 'Unknown error'));
        return {
          status: 'error',
          message: response?.message || 'Connection test failed. Please check your configuration.',
        };
      }
    } catch (error) {
      logger.error('Data source test failed', error as Error);
      
      return {
        status: 'error',
        message: error instanceof Error ? error.message : 'Failed to test data source connection. Please check your configuration.',
      };
    }
  }
}
