import React, { useState, useEffect, useCallback } from 'react';
import {
  Select,
  Input,
  Button,
  InlineField,
  InlineFieldRow,
} from '@grafana/ui';

interface NRQLQueryBuilderProps {
  value: string;
  onChange: (query: string) => void;
  onRunQuery: () => void;
}

// NRQL Query components
interface QueryComponents {
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

// Available aggregation functions
const aggregationFunctions = [
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
const commonFields = [
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
const eventTypes = [
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
  { label: '24 hours', value: '24 hours' },
  { label: '7 days', value: '7 days' },
];

export function NRQLQueryBuilder({ value, onChange, onRunQuery }: NRQLQueryBuilderProps) {
  // Parse initial query if exists
  const parseQuery = useCallback((query: string): QueryComponents => {
    const defaultComponents: QueryComponents = {
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

    if (!query || typeof query !== 'string') {
      return defaultComponents;
    }

    try {
      const components = { ...defaultComponents };
      
      // Parse SELECT clause
      const selectMatch = query.match(/SELECT\s+(.+?)\s+FROM/i);
      if (selectMatch && selectMatch[1]) {
        const selectClause = selectMatch[1].trim();
        
        // Check for SELECT *
        if (selectClause === '*') {
          components.aggregation = 'raw';
          components.field = '';
        }
        // Check for count(*)
        else if (selectClause === 'count(*)') {
          components.aggregation = 'count';
          components.field = '';
        } else {
          // Parse other aggregation functions
          const funcMatch = selectClause.match(/(\w+)\(([^)]+)\)/);
          if (funcMatch) {
            components.aggregation = funcMatch[1];
            components.field = funcMatch[2].trim();
          } else {
            // If no function found, treat as field selection
            components.aggregation = 'count';
            components.field = '';
          }
        }
      }

      const fromMatch = query.match(/FROM\s+(\w+)/i);
      if (fromMatch && fromMatch[1]) {
        components.from = fromMatch[1];
      }

      const whereMatch = query.match(/WHERE\s+(.+?)(?:\s+FACET|\s+SINCE|\s+UNTIL|\s+TIMESERIES|\s+LIMIT|$)/i);
      if (whereMatch && whereMatch[1]) {
        components.where = whereMatch[1].trim();
      }

      const facetMatch = query.match(/FACET\s+(.+?)(?:\s+SINCE|\s+UNTIL|\s+TIMESERIES|\s+LIMIT|$)/i);
      if (facetMatch && facetMatch[1]) {
        const facetItems = facetMatch[1].split(',').map(s => s.trim()).filter(s => s);
        components.facet = facetItems;
      }

      const sinceMatch = query.match(/SINCE\s+(.+?)\s+ago/i);
      if (sinceMatch && sinceMatch[1]) {
        components.since = sinceMatch[1].trim();
      }

      const untilMatch = query.match(/UNTIL\s+(.+?)\s+ago/i);
      if (untilMatch && untilMatch[1]) {
        components.until = untilMatch[1].trim();
      }

      components.timeseries = /TIMESERIES/i.test(query);

      const limitMatch = query.match(/LIMIT\s+(\d+)/i);
      if (limitMatch && limitMatch[1]) {
        const limitValue = parseInt(limitMatch[1], 10);
        if (!isNaN(limitValue)) {
          components.limit = limitValue;
        }
      }

      return components;
    } catch (error) {
      console.error('Error parsing NRQL query:', error);
      return defaultComponents;
    }
  }, []);

  // Build NRQL query from components
  const buildQuery = useCallback((components: QueryComponents): string => {
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
    
    if (components.where && components.where.trim()) {
      query += ` WHERE ${components.where}`;
    }
    
    if (components.facet && components.facet.length > 0) {
      query += ` FACET ${components.facet.join(', ')}`;
    }
    
    if (components.since && components.since.trim()) {
      query += ` SINCE ${components.since} ago`;
    }
    
    if (components.until && components.until.trim()) {
      query += ` UNTIL ${components.until} ago`;
    }
    
    if (components.timeseries) {
      query += ' TIMESERIES AUTO';
    }
    
    if (components.limit && components.limit > 0) {
      query += ` LIMIT ${components.limit}`;
    }
    
    return query;
  }, []);

  const [queryComponents, setQueryComponents] = useState<QueryComponents>(() => parseQuery(value));
  const [isUpdatingFromQuery, setIsUpdatingFromQuery] = useState(false);

  // Update query when components change (but not when we're updating from external query)
  useEffect(() => {
    if (!isUpdatingFromQuery && queryComponents) {
      const newQuery = buildQuery(queryComponents);
      if (newQuery !== value) {
        onChange(newQuery);
      }
    }
  }, [queryComponents, onChange, value, buildQuery, isUpdatingFromQuery]);

  // Update components when external query changes (from text mode)
  useEffect(() => {
    if (value !== buildQuery(queryComponents)) {
      setIsUpdatingFromQuery(true);
      const parsedComponents = parseQuery(value);
      setQueryComponents(parsedComponents);
      // Reset the flag after a short delay to allow state updates to complete
      setTimeout(() => setIsUpdatingFromQuery(false), 0);
    }
  }, [value, queryComponents, parseQuery, buildQuery]);

  const updateComponents = useCallback((update: Partial<QueryComponents>) => {
    setQueryComponents(prev => {
      if (!prev) {
        return prev;
      }
      return { ...prev, ...update };
    });
  }, []);

  // Safe value getters with fallbacks
  const safeAggregation = queryComponents?.aggregation || 'count';
  const safeField = queryComponents?.field || '';
  const safeFrom = queryComponents?.from || 'Transaction';
  const safeWhere = queryComponents?.where || '';
  const safeFacet = queryComponents?.facet || [];
  const safeSince = queryComponents?.since || '1 hour';
  const safeLimit = queryComponents?.limit || 100;
  const safeTimeseries = queryComponents?.timeseries || false;

  const selectedAggregation = aggregationFunctions.find(f => f.value === safeAggregation);
  const requiresField = selectedAggregation?.requiresField || false;

  return (
    <div className="gf-form-group">
      <InlineFieldRow>
        <InlineField label="Aggregation" grow>
          <Select
            options={aggregationFunctions}
            value={{ label: selectedAggregation?.label || safeAggregation, value: safeAggregation }}
            onChange={item => item && updateComponents({ aggregation: item.value })}
            placeholder="Choose aggregation function"
          />
        </InlineField>
      </InlineFieldRow>

      {requiresField && (
        <InlineFieldRow>
          <InlineField label="Field" grow>
            <div>
              <Input
                value={safeField}
                onChange={e => updateComponents({ field: e.currentTarget.value || '' })}
                placeholder="Enter field name (e.g., duration, responseTime)"
                list="common-fields"
              />
              <datalist id="common-fields">
                {commonFields.map(field => (
                  <option key={field.value} value={field.value} />
                ))}
              </datalist>
            </div>
          </InlineField>
        </InlineFieldRow>
      )}

      <InlineFieldRow>
        <InlineField label="From" grow>
          <Select
            options={eventTypes}
            value={{ label: safeFrom, value: safeFrom }}
            onChange={item => item && updateComponents({ from: item.value })}
            placeholder="Choose event type"
          />
        </InlineField>
      </InlineFieldRow>

      <InlineFieldRow>
        <InlineField label="Where" grow>
          <Input
            value={safeWhere}
            onChange={e => updateComponents({ where: e.currentTarget.value || '' })}
            placeholder="Add conditions (e.g., duration > 1, appName = 'MyApp')"
          />
        </InlineField>
      </InlineFieldRow>

      <InlineFieldRow>
        <InlineField label="Since" grow>
          <Select
            options={timeRangeOptions}
            value={{ label: safeSince, value: safeSince }}
            onChange={item => item && updateComponents({ since: item.value })}
            placeholder="Choose time range"
          />
        </InlineField>
      </InlineFieldRow>

      <InlineFieldRow>
        <InlineField label="Limit" grow>
          <Input
            type="number"
            value={safeLimit}
            onChange={e => {
              const value = parseInt(e.currentTarget.value, 10);
              if (!isNaN(value) && value > 0) {
                updateComponents({ limit: value });
              }
            }}
            min={1}
            max={1000}
          />
        </InlineField>
      </InlineFieldRow>

      <InlineFieldRow>
        <InlineField label="Timeseries" grow>
          <Button
            variant={safeTimeseries ? 'primary' : 'secondary'}
            onClick={() => updateComponents({ timeseries: !safeTimeseries })}
          >
            {safeTimeseries ? 'Enabled' : 'Disabled'}
          </Button>
        </InlineField>
      </InlineFieldRow>
    </div>
  );
} 