import React from 'react';
import { CodeEditor } from '@grafana/ui';
import { registerNrqlCompletionProvider } from '../utils/nrqlCompletions';

interface VariableQueryEditorProps {
  query: string;
  onChange: (query: string, definition: string) => void;
}

/**
 * Custom variable query editor with NRQL intellisense.
 * Replaces Grafana's default plain text input for "Query" type variables.
 */
export const VariableQueryEditor = ({ query, onChange }: VariableQueryEditorProps) => {
  return (
    <CodeEditor
      height="100px"
      language="sql"
      value={query}
      onBlur={(value) => onChange(value, value)}
      onBeforeEditorMount={(monaco) => registerNrqlCompletionProvider(monaco)}
      showMiniMap={false}
      monacoOptions={{ wordWrap: 'on' }}
    />
  );
};
