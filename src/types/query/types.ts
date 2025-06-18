// Query-related type definitions

// NRQL Query components
export interface QueryComponents {
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

// Aggregation function definition
export interface AggregationFunction {
  label: string;
  value: string;
  requiresField: boolean;
}

// Field definition
export interface FieldDefinition {
  label: string;
  value: string;
}

// Event type definition
export interface EventType {
  label: string;
  value: string;
}

// Time range option definition
export interface TimeRangeOption {
  label: string;
  value: string;
}

// Query builder props
export interface QueryBuilderProps {
  value: string;
  onChange: (query: string) => void;
  onRunQuery: () => void;
}

// Query validation result
export interface QueryValidationResult {
  isValid: boolean;
  message?: string;
} 