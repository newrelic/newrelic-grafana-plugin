// Mock @grafana/ui — keep real components, override CodeEditor with a testable textarea
jest.mock('@grafana/ui', () => {
  const actual = jest.requireActual('@grafana/ui');
  return {
    ...actual,
    CodeEditor: (props: any) => (
      <textarea
        data-testid="nrql-textarea"
        value={props.value}
        onChange={e => props.onChange && props.onChange(e.target.value)}
      />
    ),
  };
});

jest.mock('../../utils/nrqlCompletions', () => ({
  registerNrqlCompletionProvider: jest.fn(),
}));

import React from 'react';
import { render, screen, act, fireEvent } from '@testing-library/react';
import { QueryEditor } from '../QueryEditor';
import { NewRelicQuery } from '../../types';
import { dateTime } from '@grafana/data';

jest.mock('../query/NRQLQueryBuilder', () => ({
  NRQLQueryBuilder: ({ value, onChange, onRunQuery, useGrafanaTime }: any) => (
    <div data-testid="nrql-query-builder">
      <button onClick={() => onChange('SELECT * FROM Transaction')}>Change Query</button>
      <button onClick={onRunQuery}>Run Query</button>
      <span data-testid="builder-time-mode">{useGrafanaTime ? 'grafana-time' : 'manual-time'}</span>
      <span data-testid="builder-query-value">{value}</span>
    </div>
  ),
}));

jest.mock('../../utils/validation', () => ({
  ...jest.requireActual('../../utils/validation'),
  validateNrqlQuery: jest.fn((queryText: string) => ({
    isValid: !queryText.includes('INVALID') && queryText.trim().length > 0,
    message: queryText.includes('INVALID') ? 'Invalid NRQL syntax' :
      queryText.trim().length === 0 ? 'Query cannot be empty' : '',
  })),
}));

jest.mock('../../utils/logger', () => ({
  logger: {
    warn: jest.fn(),
    error: jest.fn(),
    debug: jest.fn(),
  },
}));

jest.mock('../../utils/timeUtils', () => ({
  buildNRQLWithTimeIntegration: jest.fn((queryText: string, enabled: boolean) => {
    if (enabled) {
      return queryText.includes('SINCE') ? queryText : `${queryText} SINCE $__from UNTIL $__to`;
    }
    return queryText;
  }),
  hasGrafanaTimeVariables: jest.fn((queryText: string) => {
    return queryText.includes('$__from') || queryText.includes('$__to');
  }),
  GRAFANA_TIME_VARIABLES: ['$__from', '$__to', '$__timeFrom', '$__timeTo']
}));

const defaultQuery: NewRelicQuery = {
  refId: 'A',
  queryText: 'SELECT * FROM Transaction',
  useGrafanaTime: true,
};

// Simplified mock datasource
const createMockDatasource = () => {
  return {
    name: 'test-datasource',
    type: 'newrelic',
    uid: 'test-uid',
    id: 1,
    query: jest.fn(),
    testDatasource: jest.fn(),
    getDefaultQuery: jest.fn(() => defaultQuery),
    applyTemplateVariables: jest.fn((query: any) => query),
    meta: {
      id: 'newrelic',
      name: 'New Relic',
    },
    getRef: jest.fn(() => ({ type: 'newrelic', uid: 'test-uid' })),
  } as any;
};

// Create proper time range mock
const createMockTimeRange = () => ({
  from: dateTime('2024-01-01T00:00:00Z'),
  to: dateTime('2024-01-01T01:00:00Z'),
  raw: { from: 'now-1h', to: 'now' }
});

const setup = async (props = {}) => {
  const onChange = jest.fn();
  const onRunQuery = jest.fn();
  const datasource = createMockDatasource();

  const defaultProps = {
    query: defaultQuery,
    onChange,
    onRunQuery,
    datasource,
    range: createMockTimeRange(),
    ...props
  };

  await act(async () => {
    render(<QueryEditor {...defaultProps} />);
  });
  return { onChange, onRunQuery, datasource };
};

/**
 * Renders QueryEditor behind a parent that only commits a new `query` prop
 * asynchronously (like Grafana's real dashboard state), instead of the
 * synchronous re-render `setup()` above uses. Lets tests reproduce races
 * where a second control's handler reads `query` before an earlier edit's
 * onChange has actually landed.
 */
const setupWithAsyncParent = async (initialQuery: NewRelicQuery) => {
  const onRunQuery = jest.fn();
  const datasource = createMockDatasource();
  const range = createMockTimeRange();
  let latestQuery = initialQuery;

  function Harness() {
    const [query, setQuery] = React.useState(initialQuery);
    (Harness as any).setQuery = setQuery;
    return (
      <QueryEditor
        query={query}
        onChange={(q: NewRelicQuery) => {
          latestQuery = q;
          // Defer the commit — mirrors Grafana's real (asynchronous) state update.
          setTimeout(() => setQuery(q), 0);
        }}
        onRunQuery={onRunQuery}
        datasource={datasource}
        range={range}
      />
    );
  }

  await act(async () => {
    render(<Harness />);
  });

  return {
    onRunQuery,
    getLatestQuery: () => latestQuery,
    flush: () => act(() => new Promise((resolve) => setTimeout(resolve, 10))),
  };
};

describe('QueryEditor', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Component Rendering', () => {
    it('renders NRQL Editor and Query Builder toggle buttons', async () => {
      await setup();
      expect(screen.getByText('NRQL Editor')).toBeInTheDocument();
      expect(screen.getByText('Query Builder')).toBeInTheDocument();
    });

    it('renders with textarea by default (NRQL Editor mode)', async () => {
      await setup();
    });
  });

  describe('Advanced timeout override', () => {
    it('starts collapsed and expands on click', async () => {
      await setup();

      expect(screen.queryByTestId('query-timeout-override-input')).not.toBeInTheDocument();

      fireEvent.click(screen.getByTestId('query-editor-advanced-toggle'));

      expect(screen.getByTestId('query-timeout-override-input')).toBeInTheDocument();
    });

    it('renders the Override Timeout label with an empty field', async () => {
      await setup();
      fireEvent.click(screen.getByTestId('query-editor-advanced-toggle'));

      expect(screen.getByText('Override Timeout')).toBeInTheDocument();
      const input = screen.getByTestId('query-timeout-override-input');
      expect(input).not.toHaveAttribute('placeholder');
      expect(input).toHaveValue(null);
    });

    it('does not introduce a timeout when other controls change and the field is unset', async () => {
      const { onChange } = await setup();

      fireEvent.change(screen.getByTestId('nrql-textarea'), { target: { value: 'SELECT count(*) FROM Log' } });
      fireEvent.click(screen.getByTestId('grafana-time-toggle'));

      expect(onChange).toHaveBeenCalled();
      for (const [q] of onChange.mock.calls) {
        expect(q.timeoutSeconds).toBeUndefined();
      }
    });

    it.each([5, 60, 120])('shows no error for in-range value %i', async (value) => {
      await setup({ query: { ...defaultQuery, timeoutSeconds: value } });

      fireEvent.blur(screen.getByTestId('query-timeout-override-input'));

      expect(screen.queryByText(/Up to 120s allowed/)).not.toBeInTheDocument();
    });

    it('starts expanded when a timeout override is already set', async () => {
      await setup({ query: { ...defaultQuery, timeoutSeconds: 30 } });

      expect(screen.getByTestId('query-timeout-override-input')).toBeInTheDocument();
    });

    it('fires onChange with the typed timeout value', async () => {
      const { onChange } = await setup();
      fireEvent.click(screen.getByTestId('query-editor-advanced-toggle'));

      fireEvent.change(screen.getByTestId('query-timeout-override-input'), { target: { value: '30' } });

      expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ timeoutSeconds: 30 }));
    });

    it('clears the override when the input is emptied', async () => {
      const { onChange } = await setup({ query: { ...defaultQuery, timeoutSeconds: 30 } });

      fireEvent.change(screen.getByTestId('query-timeout-override-input'), { target: { value: '' } });

      expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ timeoutSeconds: undefined }));
    });

    it('keeps an out-of-range value as entered and marks it invalid', async () => {
      const { onChange } = await setup({ query: { ...defaultQuery, timeoutSeconds: 200 } });

      fireEvent.blur(screen.getByTestId('query-timeout-override-input'));

      expect(onChange).not.toHaveBeenCalled();
      expect(screen.getByTestId('query-timeout-override-input')).toHaveValue(200);
      expect(screen.getByText(/Up to 120s allowed, or leave empty for default\./)).toBeInTheDocument();
    });

    it('marks a value below the allowed range invalid', async () => {
      await setup({ query: { ...defaultQuery, timeoutSeconds: 3 } });

      expect(screen.getByTestId('query-timeout-override-input')).toHaveValue(3);
      expect(screen.getByText(/Up to 120s allowed, or leave empty for default\./)).toBeInTheDocument();
    });

    it('does not rewrite a typed out-of-range value when focus leaves the field', async () => {
      const { getLatestQuery, flush } = await setupWithAsyncParent({ ...defaultQuery, timeoutSeconds: undefined });
      fireEvent.click(screen.getByTestId('query-editor-advanced-toggle'));
      const input = screen.getByTestId('query-timeout-override-input');

      fireEvent.change(input, { target: { value: '130' } });
      await flush();
      expect(screen.queryByText(/Up to 120s allowed, or leave empty for default\./)).not.toBeInTheDocument();

      fireEvent.blur(input);
      await flush();

      expect(getLatestQuery().timeoutSeconds).toBe(130);
      expect(input).toHaveValue(130);
      expect(screen.getByText(/Up to 120s allowed, or leave empty for default\./)).toBeInTheDocument();
    });

    it('clears the error once the value is edited again', async () => {
      await setup({ query: { ...defaultQuery, timeoutSeconds: 200 } });
      fireEvent.blur(screen.getByTestId('query-timeout-override-input'));
      expect(screen.getByText(/Up to 120s allowed, or leave empty for default\./)).toBeInTheDocument();

      fireEvent.change(screen.getByTestId('query-timeout-override-input'), { target: { value: '60' } });

      expect(screen.queryByText(/Up to 120s allowed, or leave empty for default\./)).not.toBeInTheDocument();
    });

    it('survives an Auto time toggle fired before the typed value has round-tripped back as a prop', async () => {
      const { getLatestQuery, flush } = await setupWithAsyncParent({ ...defaultQuery, timeoutSeconds: undefined });
      fireEvent.click(screen.getByTestId('query-editor-advanced-toggle'));

      // Type 45 into the timeout field — onChange fires, but (per the async
      // harness) the parent hasn't committed a new `query` prop yet.
      fireEvent.change(screen.getByTestId('query-timeout-override-input'), { target: { value: '45' } });

      // Immediately toggle Auto time, before that commit lands — this is the
      // exact race: handleTimeIntegrationToggle reads whatever `query` prop
      // is currently rendered, which is still the pre-45 one.
      fireEvent.click(screen.getByTestId('grafana-time-toggle'));

      await flush();

      expect(getLatestQuery().timeoutSeconds).toBe(45);
    });
  });
});
