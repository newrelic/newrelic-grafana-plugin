import React from 'react';
import { InlineFieldRow, Switch, Input, InlineField } from '@grafana/ui';
import { QueryBuilderProps } from '../../types/query/types';
import { useQueryBuilder } from '../../hooks/useQueryBuilder';
import { AggregationSelector } from './AggregationSelector';
import { FieldSelector } from './FieldSelector';
import { EventTypeSelector } from './EventTypeSelector';
import { TimeRangeSelector } from './TimeRangeSelector';
import { AGGREGATION_FUNCTIONS } from '../../types/query/constants';

export function NRQLQueryBuilder({ value, onChange, onRunQuery }: QueryBuilderProps) {
  const { queryComponents, updateComponents } = useQueryBuilder({
    initialQuery: value,
    onChange,
  });

  const selectedAggregation = AGGREGATION_FUNCTIONS.find((f) => f.value === queryComponents.aggregation);
  const showFieldSelector = selectedAggregation?.requiresField ?? false;

  return (
    <div>
      <InlineFieldRow>
        <AggregationSelector
          value={queryComponents.aggregation}
          onChange={(value) => updateComponents({ aggregation: value })}
          showFieldSelector={showFieldSelector}
        />
        {showFieldSelector && (
          <FieldSelector
            value={queryComponents.field}
            onChange={(value) => updateComponents({ field: value })}
          />
        )}
      </InlineFieldRow>

      <InlineFieldRow>
        <EventTypeSelector
          value={queryComponents.from}
          onChange={(value) => updateComponents({ from: value })}
        />
      </InlineFieldRow>

      <InlineFieldRow>
        <InlineField
          label="WHERE"
          labelWidth={14}
          tooltip="Add conditions to filter your query"
        >
          <Input
            value={queryComponents.where}
            onChange={(e) => updateComponents({ where: e.currentTarget.value })}
            placeholder="e.g., appName = 'My App'"
            width={40}
            aria-label="WHERE clause"
          />
        </InlineField>
      </InlineFieldRow>

      <InlineFieldRow>
        <InlineField
          label="FACET"
          labelWidth={14}
          tooltip="Group results by one or more attributes"
        >
          <Input
            value={queryComponents.facet.join(', ')}
            onChange={(e) => updateComponents({ facet: e.currentTarget.value.split(',').map(s => s.trim()) })}
            placeholder="e.g., appName, host"
            width={40}
            aria-label="FACET clause"
          />
        </InlineField>
      </InlineFieldRow>

      <InlineFieldRow>
        <TimeRangeSelector
          value={queryComponents.since}
          onChange={(value) => updateComponents({ since: value })}
          label="SINCE"
          tooltip="Select the time range for your query"
        />
        {queryComponents.until && (
          <TimeRangeSelector
            value={queryComponents.until}
            onChange={(value) => updateComponents({ until: value })}
            label="UNTIL"
            tooltip="Select the end time for your query"
          />
        )}
      </InlineFieldRow>

      <InlineFieldRow>
        <InlineField
          label="TIMESERIES"
          labelWidth={14}
          tooltip="Enable time series visualization"
        >
          <Switch
            value={queryComponents.timeseries}
            onChange={(e) => updateComponents({ timeseries: e.currentTarget.checked })}
            aria-label="Enable time series"
          />
        </InlineField>
      </InlineFieldRow>

      <InlineFieldRow>
        <InlineField
          label="LIMIT"
          labelWidth={14}
          tooltip="Limit the number of results"
        >
          <Input
            type="number"
            value={queryComponents.limit}
            onChange={(e) => updateComponents({ limit: parseInt(e.currentTarget.value, 10) })}
            min={1}
            max={1000}
            width={20}
            aria-label="Result limit"
          />
        </InlineField>
      </InlineFieldRow>
    </div>
  );
} 