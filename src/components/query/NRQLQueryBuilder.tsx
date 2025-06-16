import React, { useState, useCallback } from 'react';
import { InlineFieldRow, Input, InlineField, Select } from '@grafana/ui';

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

export function NRQLQueryBuilder({ value, onChange, onRunQuery, useGrafanaTime = false }: NRQLQueryBuilderProps) {
  const [aggregation, setAggregation] = useState('count');
  const [field, setField] = useState('');
  const [eventType, setEventType] = useState('Transaction');
  const [whereClause, setWhereClause] = useState('');
  const [facetClause, setFacetClause] = useState('');
  const [sinceClause, setSinceClause] = useState('1 hour');
  const [limit, setLimit] = useState(100);

  // Aggregation options
  const aggregationOptions = [
    { label: 'count(*)', value: 'count' },
    { label: 'average', value: 'average' },
    { label: 'sum', value: 'sum' },
    { label: 'min', value: 'min' },
    { label: 'max', value: 'max' },
    { label: 'latest', value: 'latest' },
  ];

  // Event type options
  const eventTypeOptions = [
    { label: 'Transaction', value: 'Transaction' },
    { label: 'Span', value: 'Span' },
    { label: 'Metric', value: 'Metric' },
    { label: 'Log', value: 'Log' },
    { label: 'Error', value: 'Error' },
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
  ];

  // Build query from components
  const buildQuery = useCallback(() => {
    let selectClause = '';
    if (aggregation === 'count') {
      selectClause = 'count(*)';
    } else if (field) {
      selectClause = `${aggregation}(${field})`;
    } else {
      selectClause = `${aggregation}(duration)`;
    }

    let query = `SELECT ${selectClause} FROM ${eventType}`;
    
    if (whereClause.trim()) {
      query += ` WHERE ${whereClause}`;
    }
    
    if (facetClause.trim()) {
      query += ` FACET ${facetClause}`;
    }
    
    if (!useGrafanaTime && sinceClause.trim()) {
      query += ` SINCE ${sinceClause} ago`;
    } else if (useGrafanaTime) {
      if (whereClause.trim()) {
        query += ` AND timestamp >= $__from AND timestamp <= $__to`;
      } else {
        query += ` WHERE timestamp >= $__from AND timestamp <= $__to`;
      }
    }
    
    if (limit > 0) {
      query += ` LIMIT ${limit}`;
    }
    
    return query;
  }, [aggregation, field, eventType, whereClause, facetClause, sinceClause, limit, useGrafanaTime]);

  // Update query when components change
  const updateQuery = useCallback(() => {
    const newQuery = buildQuery();
    onChange(newQuery);
  }, [buildQuery, onChange]);

  // Handle component changes
  const handleAggregationChange = useCallback((selectedOption: any) => {
    setAggregation(selectedOption.value);
    setTimeout(updateQuery, 0);
  }, [updateQuery]);

  const handleEventTypeChange = useCallback((selectedOption: any) => {
    setEventType(selectedOption.value);
    setTimeout(updateQuery, 0);
  }, [updateQuery]);

  const handleFieldChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setField(e.target.value);
    setTimeout(updateQuery, 0);
  }, [updateQuery]);

  const handleWhereChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setWhereClause(e.target.value);
    setTimeout(updateQuery, 0);
  }, [updateQuery]);

  const handleFacetChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setFacetClause(e.target.value);
    setTimeout(updateQuery, 0);
  }, [updateQuery]);

  const handleSinceChange = useCallback((selectedOption: any) => {
    setSinceClause(selectedOption.value);
    setTimeout(updateQuery, 0);
  }, [updateQuery]);

  const handleLimitChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const newLimit = parseInt(e.target.value, 10);
    if (!isNaN(newLimit) && newLimit >= 0) {
      setLimit(newLimit);
      setTimeout(updateQuery, 0);
    }
  }, [updateQuery]);

  return (
    <div style={{ padding: '8px 0' }}>
      {/* Aggregation and Field */}
      <InlineFieldRow>
        <InlineField label="Aggregation" labelWidth={14}>
          <Select
            options={aggregationOptions}
            value={aggregationOptions.find(opt => opt.value === aggregation)}
            onChange={handleAggregationChange}
            width={20}
          />
        </InlineField>
        {aggregation !== 'count' && (
          <InlineField label="Field" labelWidth={10}>
            <Input
              value={field}
              onChange={handleFieldChange}
              placeholder="e.g., duration"
              width={20}
            />
          </InlineField>
        )}
      </InlineFieldRow>

      {/* Event Type */}
      <InlineFieldRow>
        <InlineField label="FROM" labelWidth={14}>
          <Select
            options={eventTypeOptions}
            value={eventTypeOptions.find(opt => opt.value === eventType)}
            onChange={handleEventTypeChange}
            width={30}
          />
        </InlineField>
      </InlineFieldRow>

      {/* WHERE clause */}
      <InlineFieldRow>
        <InlineField label="WHERE" labelWidth={14}>
          <Input
            value={whereClause}
            onChange={handleWhereChange}
            placeholder="e.g., appName = 'MyApp'"
            width={40}
          />
        </InlineField>
      </InlineFieldRow>

      {/* FACET clause */}
      <InlineFieldRow>
        <InlineField label="FACET" labelWidth={14}>
          <Input
            value={facetClause}
            onChange={handleFacetChange}
            placeholder="e.g., host, appName"
            width={40}
          />
        </InlineField>
      </InlineFieldRow>

      {/* Time range (only show if not using Grafana time picker) */}
      {!useGrafanaTime && (
        <InlineFieldRow>
          <InlineField label="SINCE" labelWidth={14}>
            <Select
              options={timeRangeOptions}
              value={timeRangeOptions.find(opt => opt.value === sinceClause)}
              onChange={handleSinceChange}
              width={20}
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
            value={limit}
            onChange={handleLimitChange}
            min={1}
            max={1000}
            width={15}
          />
        </InlineField>
      </InlineFieldRow>
    </div>
  );
} 