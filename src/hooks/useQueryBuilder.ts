import { useState, useEffect, useCallback, useRef } from 'react';
import { QueryComponents, QueryValidationResult } from '../types/query/types';
import { DEFAULT_QUERY_COMPONENTS } from '../types/query/constants';
import { logger } from '../utils/logger';

interface UseQueryBuilderProps {
  initialQuery: string;
  onChange: (query: string) => void;
  useGrafanaTime?: boolean;
}

interface UseQueryBuilderResult {
  queryComponents: QueryComponents;
  updateComponents: (update: Partial<QueryComponents>) => void;
  validationResult: QueryValidationResult;
}

// Parse NRQL query into components (standalone function)
function parseNRQLQuery(query: string): QueryComponents {
  if (!query || typeof query !== 'string') {
    return DEFAULT_QUERY_COMPONENTS;
  }

  try {
    const components: QueryComponents = { 
      ...DEFAULT_QUERY_COMPONENTS,
      limit: 0  // Start with no limit - only set if explicitly found in query
    };
    
    // Parse SELECT clause with improved regex
    const selectMatch = query.match(/SELECT\s+(.+?)\s+FROM/i);
    if (selectMatch && selectMatch[1]) {
      const selectClause = selectMatch[1].trim();
      
      if (selectClause === '*') {
        components.aggregation = 'raw';
        components.field = '';
      } else if (selectClause === 'count(*)') {
        components.aggregation = 'count';
        components.field = '';
      } else {
        // Enhanced regex to handle various function patterns
        const funcMatch = selectClause.match(/(\w+)\s*\(\s*([^)]*)\s*\)/);
        if (funcMatch) {
          const funcName = funcMatch[1].toLowerCase();
          const funcParam = funcMatch[2].trim();
          
          // Handle different aggregation functions
          if (funcName === 'count') {
            components.aggregation = 'count';
            components.field = funcParam === '*' ? '' : funcParam;
          } else if (['sum', 'average', 'avg', 'min', 'max', 'latest', 'earliest', 'uniquecount', 'stddev', 'rate', 'median'].includes(funcName)) {
            components.aggregation = funcName === 'avg' ? 'average' : funcName === 'uniquecount' ? 'uniqueCount' : funcName;
            components.field = funcParam === '*' ? '' : funcParam;
          } else if (funcName === 'percentile') {
            components.aggregation = 'percentile';
            // Extract field from percentile(field, percentage)
            const percentileMatch = funcParam.match(/^([^,]+)/);
            components.field = percentileMatch ? percentileMatch[1].trim() : funcParam;
          } else {
            // Unknown function - keep as is but mark for validation
            components.aggregation = funcName;
            components.field = funcParam === '*' ? '' : funcParam;
          }
        } else {
          // If no function pattern, treat as raw field selection
          components.aggregation = 'raw';
          components.field = selectClause;
        }
      }
    }

    // Parse FROM clause
    const fromMatch = query.match(/FROM\s+(\w+)/i);
    if (fromMatch && fromMatch[1]) {
      components.from = fromMatch[1];
    }

    // Parse WHERE clause - improved to handle complex conditions and remove Grafana variables
    const whereMatch = query.match(/WHERE\s+(.+?)(?:\s+FACET|\s+SINCE|\s+UNTIL|\s+TIMESERIES|\s+LIMIT|$)/i);
    if (whereMatch && whereMatch[1]) {
      let whereClause = whereMatch[1].trim();
      
      // Remove all variations of Grafana time variables from WHERE clause
      const grafanaTimePatterns = [
        /\s*AND\s*timestamp\s*>=\s*\$__from\s*AND\s*timestamp\s*<=\s*\$__to/gi,
        /timestamp\s*>=\s*\$__from\s*AND\s*timestamp\s*<=\s*\$__to\s*AND\s*/gi,
        /timestamp\s*>=\s*\$__from\s*AND\s*timestamp\s*<=\s*\$__to/gi,
        /\s*timestamp\s*>=\s*\$__from\s*AND\s*timestamp\s*<=\s*\$__to/gi,
      ];
      
      grafanaTimePatterns.forEach(pattern => {
        whereClause = whereClause.replace(pattern, '');
      });
      
      // Clean up any remaining AND/OR at the beginning or end
      whereClause = whereClause.replace(/^\s*(AND|OR)\s*/i, '').replace(/\s*(AND|OR)\s*$/i, '');
      whereClause = whereClause.trim();
      
      components.where = whereClause;
    }

    // Parse FACET clause
    const facetMatch = query.match(/FACET\s+(.+?)(?:\s+SINCE|\s+UNTIL|\s+TIMESERIES|\s+LIMIT|$)/i);
    if (facetMatch && facetMatch[1]) {
      components.facet = facetMatch[1].split(',').map(s => s.trim()).filter(Boolean);
    }

    // Parse SINCE clause
    const sinceMatch = query.match(/SINCE\s+(.+?)\s+ago/i);
    if (sinceMatch && sinceMatch[1]) {
      components.since = sinceMatch[1].trim();
    }

    // Parse UNTIL clause
    const untilMatch = query.match(/UNTIL\s+(.+?)\s+ago/i);
    if (untilMatch && untilMatch[1]) {
      components.until = untilMatch[1].trim();
    }

    // Parse TIMESERIES
    components.timeseries = /TIMESERIES/i.test(query);

    // Parse LIMIT
    const limitMatch = query.match(/LIMIT\s+(\d+)/i);
    if (limitMatch && limitMatch[1]) {
      const limitValue = parseInt(limitMatch[1], 10);
      if (!isNaN(limitValue)) {
        components.limit = limitValue;
      }
    }
    // If no LIMIT clause found, leave it as 0 (no limit)

    return components;
  } catch (error) {
    logger.error('Error parsing NRQL query:', error as Error);
    return DEFAULT_QUERY_COMPONENTS;
  }
}

// Build NRQL query from components (standalone function)
function buildNRQLQuery(components: QueryComponents, useGrafanaTime = false): string {
  if (!components || !components.aggregation || !components.from) {
    return 'SELECT count(*) FROM Transaction SINCE 1 hour ago';
  }

  let selectClause = '';
  if (components.aggregation === 'count') {
    selectClause = 'count(*)';
  } else if (components.aggregation === 'raw') {
    selectClause = '*';
  } else {
    const field = components.field || 'duration';
    if (components.aggregation === 'percentile') {
      selectClause = `percentile(${field}, 95)`;
    } else {
      selectClause = `${components.aggregation}(${field})`;
    }
  }

  let query = `SELECT ${selectClause} FROM ${components.from}`;
  
  // Handle WHERE clause carefully to avoid duplicates
  const userWhere = components.where && components.where.trim();
  const needsGrafanaTime = useGrafanaTime;
  
  if (userWhere || needsGrafanaTime) {
    const conditions = [];
    
    // Add user WHERE conditions (excluding any existing Grafana time conditions)
    if (userWhere) {
      conditions.push(userWhere);
    }
    
    // Add Grafana time conditions only if needed and not already present
    if (needsGrafanaTime) {
      conditions.push('timestamp >= $__from AND timestamp <= $__to');
    }
    
    if (conditions.length > 0) {
      query += ` WHERE ${conditions.join(' AND ')}`;
    }
  }
  
  if (components.facet && components.facet.length > 0) {
    query += ` FACET ${components.facet.join(', ')}`;
  }
  
  // Only add SINCE/UNTIL if not using Grafana time
  if (!needsGrafanaTime) {
    if (components.since && components.since.trim()) {
      query += ` SINCE ${components.since} ago`;
    }
    
    if (components.until && components.until.trim()) {
      query += ` UNTIL ${components.until} ago`;
    }
  }
  
  if (components.timeseries) {
    query += ' TIMESERIES AUTO';
  }
  
  if (components.limit && components.limit > 0) {
    query += ` LIMIT ${components.limit}`;
  }
  
  return query;
}

export function useQueryBuilder({ initialQuery, onChange, useGrafanaTime = false }: UseQueryBuilderProps): UseQueryBuilderResult {
  // Start with defaults that don't include limit unless parsing a real query
  const [queryComponents, setQueryComponents] = useState<QueryComponents>(() => {
    if (initialQuery) {
      return parseNRQLQuery(initialQuery);
    }
    // For empty initial query, start with defaults but no limit
    return {
      ...DEFAULT_QUERY_COMPONENTS,
      limit: 0, // No limit initially
    };
  });
  const [validationResult, setValidationResult] = useState<QueryValidationResult>({ isValid: true });
  
  const lastQueryRef = useRef<string>(initialQuery || '');
  const isUpdatingRef = useRef<boolean>(false);

  // Update components when external query changes
  useEffect(() => {
    const currentQuery = initialQuery || '';
    if (currentQuery !== lastQueryRef.current && !isUpdatingRef.current) {
      isUpdatingRef.current = true;
      lastQueryRef.current = currentQuery;
      
      const parsedComponents = currentQuery ? parseNRQLQuery(currentQuery) : {
        ...DEFAULT_QUERY_COMPONENTS,
        limit: 0,
      };
      
      setQueryComponents(parsedComponents);
      
      // Reset the flag asynchronously
      setTimeout(() => {
        isUpdatingRef.current = false;
      }, 0);
    }
  }, [initialQuery]);

  // Also watch for useGrafanaTime changes to rebuild the query properly
  useEffect(() => {
    if (!isUpdatingRef.current && queryComponents.aggregation) {
      const newQuery = buildNRQLQuery(queryComponents, useGrafanaTime);
      if (newQuery !== lastQueryRef.current) {
        lastQueryRef.current = newQuery;
        onChange(newQuery);
      }
    }
  }, [useGrafanaTime, onChange, queryComponents]);

  // Update components
  const updateComponents = useCallback((update: Partial<QueryComponents>) => {
    setQueryComponents(prev => {
      const newComponents = { ...prev, ...update };
      // Trigger query rebuild immediately with Grafana time awareness
      const newQuery = buildNRQLQuery(newComponents, useGrafanaTime);
      if (newQuery !== lastQueryRef.current) {
        lastQueryRef.current = newQuery;
        onChange(newQuery);
      }
      return newComponents;
    });
  }, [onChange, useGrafanaTime]);

  return {
    queryComponents,
    updateComponents,
    validationResult,
  };
} 