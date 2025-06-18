import { DataQuery, DataSourceJsonData } from '@grafana/data';

/**
 * Represents a New Relic NRQL query configuration
 */
export interface NewRelicQuery extends DataQuery {
  /** The NRQL query string to execute */
  queryText: string;
  /** Optional account ID to override the default configured account */
  accountID?: number;
  /** Whether to use Grafana's time picker for automatic time range integration */
  useGrafanaTime?: boolean;
}

/**
 * Configuration options for the New Relic data source
 */
export interface NewRelicDataSourceOptions extends DataSourceJsonData {
  /** New Relic API key (stored securely) */
  apiKey?: string;
  /** New Relic account ID */
  accountId?: number;
  /** New Relic region (US or EU) */
  region?: 'US' | 'EU';
  /** Custom API endpoint URL (optional) */
  apiUrl?: string;
}

/**
 * Represents a data point returned from New Relic
 */
export interface DataPoint {
  /** Timestamp in milliseconds */
  Time: number;
  /** Numeric value */
  Value: number;
}

/**
 * Response structure from New Relic API
 */
export interface DataSourceResponse {
  /** Array of data points */
  datapoints: DataPoint[];
}

/**
 * Secure configuration data that is only sent to the backend
 * Never exposed to the frontend for security reasons
 */
export interface NewRelicSecureJsonData {
  /** New Relic API key */
  apiKey?: string;
  /** New Relic account ID */
  accountID?: string;
}

/**
 * Available New Relic regions
 */
export const NEW_RELIC_REGIONS = {
  US: 'US',
  EU: 'EU',
} as const;

/**
 * Default API endpoints for different regions
 */
export const NEW_RELIC_API_ENDPOINTS = {
  US: 'https://api.newrelic.com/graphql',
  EU: 'https://api.eu.newrelic.com/graphql',
} as const;

/**
 * Common NRQL event types
 */
export const NRQL_EVENT_TYPES = [
  'Transaction',
  'Span',
  'Metric',
  'Log',
  'Error',
  'PageView',
  'PageAction',
  'BrowserInteraction',
  'Mobile',
  'MobileSession',
  'MobileCrash',
  'MobileHandledException',
] as const;

/**
 * Common NRQL aggregation functions
 */
export const NRQL_AGGREGATION_FUNCTIONS = [
  'count',
  'sum',
  'average',
  'min',
  'max',
  'percentile',
  'uniqueCount',
  'latest',
  'earliest',
  'stddev',
  'rate',
] as const;

/**
 * Validation result interface
 */
export interface ValidationResult {
  isValid: boolean;
  message?: string;
}

/**
 * Query builder component state
 */
export interface QueryBuilderState {
  aggregation: string;
  field: string;
  from: string;
  where: string;
  facet: string[];
  since: string;
  until: string;
  timeseries: boolean;
  limit: number;
}
