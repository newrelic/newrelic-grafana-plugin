import React, { useState, useCallback, useEffect } from 'react';
import { InlineFieldRow, Input, InlineField, Select, Alert } from '@grafana/ui';
import { useQueryBuilder } from '../../hooks/useQueryBuilder';

interface NRQLQueryBuilderProps {
  /** The current query value */
  value: string;
  /** Callback when query changes */
  onChange: (query: string) => void;
  /** Callback to run the query */
  onRunQuery: () => void;
  /** Whether to use Grafana's time picker integration */
  useGrafanaTime?: boolean;
}

// Valid aggregation functions for validation
const VALID_AGGREGATIONS = [
  'count', 'sum', 'average', 'min', 'max', 'latest', 'earliest', 
  'percentile', 'uniqueCount', 'stddev', 'rate', 'median'
];

export function NRQLQueryBuilder({ value, onChange, onRunQuery, useGrafanaTime = false }: NRQLQueryBuilderProps) {
  const { queryComponents, updateComponents } = useQueryBuilder({
    initialQuery: value,
    onChange,
    useGrafanaTime
  });

  const [validationErrors, setValidationErrors] = useState<string[]>([]);

  // Aggregation options - only show valid ones
  const aggregationOptions = [
    { label: 'count(*)', value: 'count' },
    { label: 'average', value: 'average' },
    { label: 'sum', value: 'sum' },
    { label: 'min', value: 'min' },
    { label: 'max', value: 'max' },
    { label: 'latest', value: 'latest' },
    { label: 'earliest', value: 'earliest' },
    { label: 'percentile', value: 'percentile' },
    { label: 'uniqueCount', value: 'uniqueCount' },
  ];

  // Event type options
  const eventTypeOptions = [
    { label: 'Transaction', value: 'Transaction' },
    { label: 'Span', value: 'Span' },
    { label: 'Metric', value: 'Metric' },
    { label: 'Log', value: 'Log' },
    { label: 'Error', value: 'Error' },
    { label: 'PageView', value: 'PageView' },
    { label: 'Mobile', value: 'Mobile' },
    { label: 'Browser', value: 'Browser' },
  ];

  // Time range options
  const timeRangeOptions = [
    { label: '5 minutes', value: '5 minutes' },
    { label: '15 minutes', value: '15 minutes' },
    { label: '30 minutes', value: '30 minutes' },
    { label: '1 hour', value: '1 hour' },
    { label: '3 hours', value: '3 hours' },
    { label: '6 hours', value: '6 hours' },
    { label: '12 hours', value: '12 hours' },
    { label: '1 day', value: '1 day' },
    { label: '7 days', value: '7 days' },
  ];

  // Validate components whenever they change
  useEffect(() => {
    const errors: string[] = [];

    // Validate aggregation
    if (queryComponents.aggregation && !VALID_AGGREGATIONS.includes(queryComponents.aggregation)) {
      errors.push(`Invalid aggregation function: "${queryComponents.aggregation}". Use one of: ${VALID_AGGREGATIONS.join(', ')}`);
    }

    // Validate field for certain aggregations
    if (['sum', 'average', 'min', 'max', 'percentile'].includes(queryComponents.aggregation) && 
        !queryComponents.field?.trim()) {
      errors.push(`Aggregation "${queryComponents.aggregation}" requires a field name (e.g., duration, responseTime)`);
    }

    // Validate event type
    if (!queryComponents.from?.trim()) {
      errors.push('Event type (FROM clause) is required');
    }

    setValidationErrors(errors);
  }, [queryComponents]);

  // Handle component changes with validation
  const handleAggregationChange = useCallback((selectedOption: any) => {
    const newAggregation = selectedOption.value;
    
    // Reset field if switching to count
    const updates: any = { aggregation: newAggregation };
    if (newAggregation === 'count') {
      updates.field = '';
    } else if (!queryComponents.field && newAggregation !== 'count') {
      // Set default field for non-count aggregations
      updates.field = 'duration';
    }
    
    updateComponents(updates);
  }, [updateComponents, queryComponents.field]);

  const handleEventTypeChange = useCallback((selectedOption: any) => {
    updateComponents({ from: selectedOption.value });
  }, [updateComponents]);

  const handleFieldChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    updateComponents({ field: e.target.value });
  }, [updateComponents]);

  const handleWhereChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    updateComponents({ where: e.target.value });
  }, [updateComponents]);

  const handleFacetChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const facetValue = e.target.value;
    const facetArray = facetValue ? facetValue.split(',').map(s => s.trim()).filter(Boolean) : [];
    updateComponents({ facet: facetArray });
  }, [updateComponents]);

  const handleSinceChange = useCallback((selectedOption: any) => {
    updateComponents({ since: selectedOption.value });
  }, [updateComponents]);

  const handleLimitChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const newLimit = parseInt(e.target.value, 10);
    if (!isNaN(newLimit) && newLimit >= 0) {
      updateComponents({ limit: newLimit });
    }
  }, [updateComponents]);

  // Find selected options for selects
  const selectedAggregation = aggregationOptions.find(opt => opt.value === queryComponents.aggregation);
  const selectedEventType = eventTypeOptions.find(opt => opt.value === queryComponents.from);
  const selectedTimeRange = timeRangeOptions.find(opt => opt.value === queryComponents.since);

  return (
    <div style={{ padding: '8px 0' }}>
      {/* Validation Errors */}
      {validationErrors.length > 0 && (
        <Alert title="Query Builder Validation Errors" severity="error" style={{ marginBottom: '12px' }}>
          <ul style={{ margin: 0, paddingLeft: '20px' }}>
            {validationErrors.map((error, index) => (
              <li key={index}>{error}</li>
            ))}
          </ul>
        </Alert>
      )}

      {/* Aggregation and Field */}
      <InlineFieldRow>
        <InlineField label="Aggregation" labelWidth={14}>
          <Select
            options={aggregationOptions}
            value={selectedAggregation}
            onChange={handleAggregationChange}
            width={20}
            placeholder="Select aggregation"
          />
        </InlineField>
        {queryComponents.aggregation !== 'count' && (
          <InlineField label="Field" labelWidth={10}>
            <Input
              value={queryComponents.field || ''}
              onChange={handleFieldChange}
              placeholder="e.g., duration, responseTime"
              width={25}
              invalid={['sum', 'average', 'min', 'max', 'percentile'].includes(queryComponents.aggregation) && !queryComponents.field?.trim()}
            />
          </InlineField>
        )}
      </InlineFieldRow>

      {/* Event Type */}
      <InlineFieldRow>
        <InlineField label="FROM" labelWidth={14}>
          <Select
            options={eventTypeOptions}
            value={selectedEventType}
            onChange={handleEventTypeChange}
            width={30}
            placeholder="Select event type"
            invalid={!queryComponents.from?.trim()}
          />
        </InlineField>
      </InlineFieldRow>

      {/* WHERE clause */}
      <InlineFieldRow>
        <InlineField label="WHERE" labelWidth={14}>
          <Input
            value={queryComponents.where || ''}
            onChange={handleWhereChange}
            placeholder="e.g., appName = 'MyApp' AND duration > 1"
            width={50}
          />
        </InlineField>
      </InlineFieldRow>

      {/* FACET clause */}
      <InlineFieldRow>
        <InlineField label="FACET" labelWidth={14}>
          <Input
            value={queryComponents.facet?.join(', ') || ''}
            onChange={handleFacetChange}
            placeholder="e.g., host, appName"
            width={40}
          />
        </InlineField>
      </InlineFieldRow>

      {/* Time range (only show if not using Grafana time picker and time filtering is enabled) */}
      {!useGrafanaTime && (
        <InlineFieldRow>
          <InlineField label="SINCE" labelWidth={14}>
            <Select
              options={timeRangeOptions}
              value={selectedTimeRange}
              onChange={handleSinceChange}
              width={20}
              placeholder="Select time range"
            />
          </InlineField>
        </InlineFieldRow>
      )}

      {/* Show time picker info when enabled */}
      {useGrafanaTime && (
        <InlineFieldRow>
          <InlineField label="Time Range" labelWidth={14}>
            <div style={{ 
              padding: '4px 8px', 
              backgroundColor: '#e3f2fd', 
              border: '1px solid #2196f3', 
              borderRadius: '3px',
              fontSize: '11px',
              color: '#0d47a1'
            }}>
              Using Grafana Dashboard Time Picker
            </div>
          </InlineField>
        </InlineFieldRow>
      )}

      {/* LIMIT */}
      <InlineFieldRow>
        <InlineField label="LIMIT" labelWidth={14}>
          <Input
            type="number"
            value={queryComponents.limit || ''}
            onChange={handleLimitChange}
            min={1}
            max={1000}
            width={15}
            placeholder="100"
          />
        </InlineField>
      </InlineFieldRow>

      {/* Query Preview */}
      <div style={{ 
        marginTop: '12px', 
        padding: '8px', 
        backgroundColor: '#f5f5f5', 
        borderRadius: '4px',
        fontSize: '12px',
        fontFamily: 'monospace'
      }}>
        <strong>Generated Query:</strong><br/>
        <span style={{ color: validationErrors.length > 0 ? '#d32f2f' : '#2e7d32' }}>
          {value || 'SELECT count(*) FROM Transaction SINCE 1 hour ago'}
        </span>
      </div>
    </div>
  );
} 