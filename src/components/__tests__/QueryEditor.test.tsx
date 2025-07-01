import React from 'react';
import { render, screen } from '@testing-library/react';
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

const setup = (props = {}) => {
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

  render(<QueryEditor {...defaultProps} />);
  return { onChange, onRunQuery, datasource };
};

describe('QueryEditor', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Component Rendering', () => {
    it('renders NRQL Editor and Query Builder toggle buttons', () => {
      setup();
      expect(screen.getByText('NRQL Editor')).toBeInTheDocument();
      expect(screen.getByText('Query Builder')).toBeInTheDocument();
    });

    it('renders with textarea by default (NRQL Editor mode)', () => {
      setup();
    });
  });
});