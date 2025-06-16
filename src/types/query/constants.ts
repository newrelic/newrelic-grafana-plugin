import { AggregationFunction, FieldDefinition, EventType, TimeRangeOption } from './types';

// Available aggregation functions
export const AGGREGATION_FUNCTIONS: AggregationFunction[] = [
  { label: 'count(*)', value: 'count', requiresField: false },
  { label: 'SELECT * (all fields)', value: 'raw', requiresField: false },
  { label: 'average', value: 'average', requiresField: true },
  { label: 'sum', value: 'sum', requiresField: true },
  { label: 'min', value: 'min', requiresField: true },
  { label: 'max', value: 'max', requiresField: true },
  { label: 'percentile', value: 'percentile', requiresField: true },
  { label: 'latest', value: 'latest', requiresField: true },
  { label: 'uniqueCount', value: 'uniqueCount', requiresField: true },
];

// Common field names
export const COMMON_FIELDS: FieldDefinition[] = [
  { label: 'duration', value: 'duration' },
  { label: 'responseTime', value: 'responseTime' },
  { label: 'appName', value: 'appName' },
  { label: 'host', value: 'host' },
  { label: 'name', value: 'name' },
  { label: 'entityGuid', value: 'entityGuid' },
  { label: 'userId', value: 'userId' },
  { label: 'sessionId', value: 'sessionId' },
];

// Available event types
export const EVENT_TYPES: EventType[] = [
  { label: 'Transaction', value: 'Transaction' },
  { label: 'Span', value: 'Span' },
  { label: 'Metric', value: 'Metric' },
  { label: 'Log', value: 'Log' },
  { label: 'Error', value: 'Error' },
];

// Time range options
export const TIME_RANGE_OPTIONS: TimeRangeOption[] = [
  { label: '5 minutes', value: '5 minutes' },
  { label: '15 minutes', value: '15 minutes' },
  { label: '30 minutes', value: '30 minutes' },
  { label: '1 hour', value: '1 hour' },
  { label: '3 hours', value: '3 hours' },
  { label: '6 hours', value: '6 hours' },
  { label: '12 hours', value: '12 hours' },
  { label: '24 hours', value: '24 hours' },
  { label: '7 days', value: '7 days' },
];

// Default query components
export const DEFAULT_QUERY_COMPONENTS = {
  aggregation: 'count',
  field: '',
  from: 'Transaction',
  where: '',
  facet: [],
  since: '1 hour',
  until: '',
  timeseries: false,
  limit: 100,
}; 