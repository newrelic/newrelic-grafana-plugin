import { DataSourceInstanceSettings, SupplementaryQueryType, SupplementaryQueryOptions } from '@grafana/data';
import { DataSource } from '../datasource';
import { NewRelicDataSourceOptions, NewRelicQuery } from '../types';

jest.mock('@grafana/runtime', () => ({
  DataSourceWithBackend: class {},
  getTemplateSrv: () => ({ replace: (s: string) => s }),
}));

describe('DataSource.filterQuery', () => {
  const ds = new DataSource({ jsonData: {} } as DataSourceInstanceSettings<NewRelicDataSourceOptions>);
  const query = (timeoutSeconds?: number): NewRelicQuery =>
    ({ refId: 'A', queryText: 'SELECT count(*) FROM Transaction', timeoutSeconds }) as NewRelicQuery;

  it.each([undefined, 5, 120])('runs the query with timeout %p', (value) => {
    expect(ds.filterQuery(query(value))).toBe(true);
  });

  it.each([3, 121, 140, 1.5])('skips the query with invalid timeout %p', (value) => {
    expect(ds.filterQuery(query(value))).toBe(false);
  });
});

describe('DataSource.getSupplementaryQuery', () => {
  const ds = new DataSource({ jsonData: {} } as DataSourceInstanceSettings<NewRelicDataSourceOptions>);

  it('does not carry table output into the log volume query', () => {
    const originalQuery: NewRelicQuery = {
      refId: 'A',
      queryText: "SELECT message FROM Log WHERE hostname = 'web-1'",
      format: 'table',
    };
    const result = ds.getSupplementaryQuery(
      { type: SupplementaryQueryType.LogsVolume } as SupplementaryQueryOptions,
      originalQuery
    );
    expect(result).toEqual(expect.objectContaining({
      refId: 'A-logs-volume',
      queryText: "SELECT count(*) FROM Log WHERE hostname = 'web-1' TIMESERIES",
    }));
    expect(result?.format).toBeUndefined();
    expect(originalQuery.format).toBe('table');
  });
});
