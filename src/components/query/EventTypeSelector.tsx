import React from 'react';
import { Select, InlineField } from '@grafana/ui';
import { EVENT_TYPES } from '../../types/query/constants';

interface EventTypeSelectorProps {
  value: string;
  onChange: (value: string) => void;
}

export function EventTypeSelector({ value, onChange }: EventTypeSelectorProps) {
  const selectedType = EVENT_TYPES.find(t => t.value === value) || EVENT_TYPES[0];

  return (
    <InlineField
      label="Event Type"
      labelWidth={14}
      tooltip="Select the event type to query"
    >
      <Select
        options={EVENT_TYPES}
        value={selectedType}
        onChange={(option) => option?.value && onChange(option.value)}
        width={40}
        aria-label="Select event type"
      />
    </InlineField>
  );
} 