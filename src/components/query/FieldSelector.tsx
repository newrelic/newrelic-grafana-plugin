import React from 'react';
import { Select, InlineField } from '@grafana/ui';
import { COMMON_FIELDS } from '../../types/query/constants';

interface FieldSelectorProps {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

export function FieldSelector({ value, onChange, disabled }: FieldSelectorProps) {
  const selectedField = COMMON_FIELDS.find(f => f.value === value) || COMMON_FIELDS[0];

  return (
    <InlineField
      label="Field"
      labelWidth={14}
      tooltip="Select the field to aggregate"
      disabled={disabled}
    >
      <Select
        options={COMMON_FIELDS}
        value={selectedField}
        onChange={(option) => option?.value && onChange(option.value)}
        width={40}
        isDisabled={disabled}
        aria-label="Select field"
      />
    </InlineField>
  );
} 