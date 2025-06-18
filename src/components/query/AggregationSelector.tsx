import React from 'react';
import { Select, InlineField } from '@grafana/ui';
import { AGGREGATION_FUNCTIONS } from '../../types/query/constants';

interface AggregationSelectorProps {
  value: string;
  onChange: (value: string) => void;
  showFieldSelector: boolean;
}

export function AggregationSelector({ value, onChange, showFieldSelector }: AggregationSelectorProps) {
  const selectedFunction = AGGREGATION_FUNCTIONS.find(f => f.value === value) || AGGREGATION_FUNCTIONS[0];

  return (
    <InlineField
      label="Aggregation"
      labelWidth={14}
      tooltip="Select the aggregation function for your query"
    >
      <Select
        options={AGGREGATION_FUNCTIONS}
        value={selectedFunction}
        onChange={(option) => option?.value && onChange(option.value)}
        width={40}
        aria-label="Select aggregation function"
      />
    </InlineField>
  );
} 