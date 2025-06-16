import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface NewRelicQuery extends DataQuery {
  queryText: string;
  accountID?: number;
}

export interface NewRelicDataSourceOptions extends DataSourceJsonData {
  apiKey?: string;
  accountId?: number;
  region?: string;
}

export interface DataPoint {
  Time: number;
  Value: number;
}

export interface DataSourceResponse {
  datapoints: DataPoint[];
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface NewRelicSecureJsonData {
  apiKey?: string;
}
