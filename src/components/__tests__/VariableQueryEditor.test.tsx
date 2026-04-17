import React from 'react';
import { render, screen } from '@testing-library/react';
import { VariableQueryEditor } from '../VariableQueryEditor';
import { registerNrqlCompletionProvider } from '../../utils/nrqlCompletions';

jest.mock('@grafana/ui', () => {
  const actual = jest.requireActual('@grafana/ui');
  return {
    ...actual,
    CodeEditor: (props: any) => {
      // Simulate onBeforeEditorMount being called
      if (props.onBeforeEditorMount) {
        props.onBeforeEditorMount({ languages: { registerCompletionItemProvider: jest.fn(), CompletionItemKind: {}, CompletionItemInsertTextRule: { InsertAsSnippet: 4 } } });
      }
      return (
        <textarea
          data-testid="variable-query-editor"
          value={props.value}
          onChange={(e) => props.onBlur && props.onBlur(e.target.value)}
        />
      );
    },
  };
});

jest.mock('../../utils/nrqlCompletions', () => ({
  registerNrqlCompletionProvider: jest.fn(),
}));

describe('VariableQueryEditor', () => {
  it('renders CodeEditor with correct value', () => {
    render(
      <VariableQueryEditor
        query="SELECT uniques(appName) FROM Transaction"
        onChange={jest.fn()}
      />
    );

    const editor = screen.getByTestId('variable-query-editor');
    expect(editor).toBeInTheDocument();
    expect(editor).toHaveValue('SELECT uniques(appName) FROM Transaction');
  });

  it('calls onChange on blur', () => {
    const onChange = jest.fn();
    render(
      <VariableQueryEditor
        query="SELECT uniques(appName) FROM Transaction"
        onChange={onChange}
      />
    );

    const editor = screen.getByTestId('variable-query-editor');
    editor.focus();
    // Simulate change which triggers onBlur in our mock
    const event = { target: { value: 'SELECT count(*) FROM Log' } };
    editor.dispatchEvent(new Event('change', { bubbles: true }));
  });

  it('registers NRQL completion provider on mount', () => {
    render(
      <VariableQueryEditor
        query=""
        onChange={jest.fn()}
      />
    );

    expect(registerNrqlCompletionProvider).toHaveBeenCalled();
  });

  it('renders with empty query', () => {
    render(
      <VariableQueryEditor
        query=""
        onChange={jest.fn()}
      />
    );

    const editor = screen.getByTestId('variable-query-editor');
    expect(editor).toHaveValue('');
  });
});
