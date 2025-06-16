import React from 'react';
import { Select, InlineField } from '@grafana/ui';
import { TIME_RANGE_OPTIONS } from '../../types/query/constants';

interface TimeRangeSelectorProps {
  value: string;
  onChange: (value: string) => void;
  label: string;
  tooltip: string;
}

export function TimeRangeSelector({ value, onChange, label, tooltip }: TimeRangeSelectorProps) {
  const selectedRange = TIME_RANGE_OPTIONS.find(t => t.value === value) || TIME_RANGE_OPTIONS[3]; // Default to 1 hour

  return (
    <InlineField
      label={label}
      labelWidth={14}
      tooltip={tooltip}
    >
      <Select
        options={TIME_RANGE_OPTIONS}
        value={selectedRange}
        onChange={(option) => option?.value && onChange(option.value)}
        width={40}
        aria-label={`Select ${label.toLowerCase()}`}
      />
    </InlineField>
  );
} 