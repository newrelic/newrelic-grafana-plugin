import { DataSourceInstanceSettings, CoreApp, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

import { NewRelicQuery, NewRelicDataSourceOptions } from './types';

export class DataSource extends DataSourceWithBackend<NewRelicQuery, NewRelicDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<NewRelicDataSourceOptions>) {
    super(instanceSettings);
  }

  getDefaultQuery(_: CoreApp): Partial<NewRelicQuery> {
    return {
      queryText: 'SELECT count(*) FROM Transaction SINCE 1 hour ago'
    };
  }

  applyTemplateVariables(query: NewRelicQuery, scopedVars: ScopedVars) {
    // Migration logic: if old 'nrql' field exists but queryText doesn't, migrate it
    const migratedQuery = this.migrateQuery(query);
    
    return {
      ...migratedQuery,
      queryText: getTemplateSrv().replace(migratedQuery.queryText, scopedVars),
    };
  }

  filterQuery(query: NewRelicQuery): boolean {
    // Migrate query before filtering
    const migratedQuery = this.migrateQuery(query);
    // if no query has been provided, prevent the query from being executed
    return !!migratedQuery.queryText;
  }

  private migrateQuery(query: any): NewRelicQuery {
    // Handle migration from old 'nrql' field to new 'queryText' field
    if (query.nrql && !query.queryText) {
      console.log('Migrating old nrql field to queryText:', query.nrql);
      return {
        ...query,
        queryText: query.nrql,
        nrql: undefined, // Remove the old field
      };
    }
    
    // If both exist, prefer queryText and remove nrql
    if (query.nrql && query.queryText) {
      console.log('Removing duplicate nrql field, keeping queryText:', query.queryText);
      return {
        ...query,
        nrql: undefined, // Remove the old field
      };
    }
    
    return query;
  }
}
