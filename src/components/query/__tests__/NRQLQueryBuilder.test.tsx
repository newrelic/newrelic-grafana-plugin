import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { NRQLQueryBuilder } from '../NRQLQueryBuilder';

// Mock Grafana UI components
jest.mock('@grafana/ui', () => ({
  ...jest.requireActual('@grafana/ui'),
  InlineFieldRow: ({ children }: any) => <div className="css-15ix71y-InlineFieldRow">{children}</div>,
  InlineField: ({ label, tooltip, children }: any) => (
    <div className="css-1uqla6q">
      <label className="css-v249xx">
        {label}
        {tooltip && <svg data-testid="info-circle" />}
      </label>
      <div className="css-1tu59u4">{children}</div>
    </div>
  ),
  Input: ({ value, onChange, placeholder, type, min, max, width, 'aria-label': ariaLabel, ...props }: any) => (
    <div className="css-1c7kjdq-input-wrapper" data-testid="input-wrapper">
      <div className="css-10lnb82-input-inputWrapper">
        <input
          type={type || 'text'}
          value={value}
          onChange={(e) => {
            // Create synthetic event with currentTarget
            const syntheticEvent = {
              currentTarget: { value: e.target.value },
              target: { value: e.target.value }
            };
            onChange?.(syntheticEvent);
          }}
          placeholder={placeholder}
          min={min}
          max={max}
          width={width}
          aria-label={ariaLabel}
          className="css-xmqqi8-input-input"
          {...props}
        />
      </div>
    </div>
  ),
  Switch: ({ value, onChange, ...props }: any) => (
    <div className="css-1n85obj">
      <input
        type="checkbox"
        role="switch"
        checked={value}
        onChange={(e) => {
          const syntheticEvent = {
            currentTarget: { checked: e.target.checked },
            target: { checked: e.target.checked }
          };
          onChange?.(syntheticEvent);
        }}
        id={`switch-${Math.random().toString().slice(2)}`}
        aria-label="Enable time series"
        {...props}
      />
      <label htmlFor={`switch-${Math.random().toString().slice(2)}`}>
        <svg data-testid="check" />
      </label>
    </div>
  ),
}));

// Mock the hook
jest.mock('../../../hooks/useQueryBuilder');
const mockUseQueryBuilder = require('../../../hooks/useQueryBuilder').useQueryBuilder as jest.MockedFunction<any>;

// Mock child components with correct prop interfaces
jest.mock('../AggregationSelector', () => ({
  AggregationSelector: ({ value, onChange }: any) => (
    <select
      data-testid="aggregation-selector"
      value={value}
      onChange={(e) => onChange(e.target.value)}
    >
      <option value="count">count</option>
      <option value="average">average</option>
      <option value="sum">sum</option>
    </select>
  ),
}));

jest.mock('../FieldSelector', () => ({
  FieldSelector: ({ value, onChange }: any) => (
    <input
      data-testid="field-selector"
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  ),
}));

jest.mock('../EventTypeSelector', () => ({
  EventTypeSelector: ({ value, onChange }: any) => (
    <select
      data-testid="event-type-selector"
      value={value}
      onChange={(e) => onChange(e.target.value)}
    >
      <option value="Transaction">Transaction</option>
      <option value="Span">Span</option>
      <option value="Metric">Metric</option>
    </select>
  ),
}));

// Create a counter to give unique test IDs
let timeRangeSelectorCounter = 0;

jest.mock('../TimeRangeSelector', () => ({
  TimeRangeSelector: ({ value, onChange, label }: any) => {
    const testId = `time-range-selector-${label?.toLowerCase() || ++timeRangeSelectorCounter}`;
    return (
      <select
        data-testid={testId}
        value={value}
        onChange={(e) => onChange(e.target.value)}
      >
        <option value="1 hour">1 hour</option>
        <option value="6 hours">6 hours</option>
        <option value="24 hours">24 hours</option>
      </select>
    );
  },
}));

describe('NRQLQueryBuilder', () => {
  const mockOnChange = jest.fn();
  const mockOnRunQuery = jest.fn();
  const mockUpdateComponents = jest.fn();

  const defaultQueryComponents = {
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

  beforeEach(() => {
    jest.clearAllMocks();
    mockUseQueryBuilder.mockReturnValue({
      queryComponents: defaultQueryComponents,
      updateComponents: mockUpdateComponents,
      validationResult: { isValid: true },
    });
  });

  const defaultProps = {
    value: 'SELECT count(*) FROM Transaction SINCE 1 hour ago',
    onChange: mockOnChange,
    onRunQuery: mockOnRunQuery,
  };

  describe('rendering', () => {
    it('should render all query builder components', () => {
      render(<NRQLQueryBuilder {...defaultProps} />);

      expect(screen.getByTestId('aggregation-selector')).toBeInTheDocument();
      expect(screen.getByTestId('event-type-selector')).toBeInTheDocument();
      expect(screen.getByTestId('time-range-selector-since')).toBeInTheDocument();
      expect(screen.getByPlaceholderText('e.g., appName = \'My App\'')).toBeInTheDocument();
      expect(screen.getByPlaceholderText('e.g., appName, host')).toBeInTheDocument();
      expect(screen.getByRole('switch')).toBeInTheDocument();
      expect(screen.getByRole('spinbutton')).toBeInTheDocument();
    });

    it('should render field selector when aggregation requires field', () => {
      mockUseQueryBuilder.mockReturnValue({
        queryComponents: { ...defaultQueryComponents, aggregation: 'average' },
        updateComponents: mockUpdateComponents,
        validationResult: { isValid: true },
      });

      render(<NRQLQueryBuilder {...defaultProps} />);

      expect(screen.getByTestId('field-selector')).toBeInTheDocument();
    });

    it('should not render field selector when aggregation does not require field', () => {
      render(<NRQLQueryBuilder {...defaultProps} />);

      expect(screen.queryByTestId('field-selector')).not.toBeInTheDocument();
    });

    it('should render UNTIL selector when until value exists', () => {
      mockUseQueryBuilder.mockReturnValue({
        queryComponents: { ...defaultQueryComponents, until: '30 minutes' },
        updateComponents: mockUpdateComponents,
        validationResult: { isValid: true },
      });

      render(<NRQLQueryBuilder {...defaultProps} />);

      expect(screen.getByTestId('time-range-selector-since')).toBeInTheDocument();
      expect(screen.getByTestId('time-range-selector-until')).toBeInTheDocument();
    });
  });

  describe('user interactions', () => {
    it('should call updateComponents when aggregation changes', async () => {
      const user = userEvent.setup();
      render(<NRQLQueryBuilder {...defaultProps} />);

      const aggregationSelector = screen.getByTestId('aggregation-selector');
      await user.selectOptions(aggregationSelector, 'average');

      expect(mockUpdateComponents).toHaveBeenCalledWith({ aggregation: 'average' });
    });

    it('should call updateComponents when event type changes', async () => {
      const user = userEvent.setup();
      render(<NRQLQueryBuilder {...defaultProps} />);

      const eventTypeSelector = screen.getByTestId('event-type-selector');
      await user.selectOptions(eventTypeSelector, 'Span');

      expect(mockUpdateComponents).toHaveBeenCalledWith({ from: 'Span' });
    });

    it('should call updateComponents when SINCE changes', async () => {
      const user = userEvent.setup();
      render(<NRQLQueryBuilder {...defaultProps} />);

      const sinceSelector = screen.getByTestId('time-range-selector-since');
      await user.selectOptions(sinceSelector, '6 hours');

      expect(mockUpdateComponents).toHaveBeenCalledWith({ since: '6 hours' });
    });

    it('should call updateComponents when TIMESERIES switch is toggled', async () => {
      const user = userEvent.setup();
      render(<NRQLQueryBuilder {...defaultProps} />);

      const timeseriesSwitch = screen.getByRole('switch');
      await user.click(timeseriesSwitch);

      expect(mockUpdateComponents).toHaveBeenCalledWith({ timeseries: true });
    });
  });

  describe('component values', () => {
    it('should display current query component values', () => {
      const customComponents = {
        aggregation: 'average',
        field: 'duration',
        from: 'Span',
        where: 'appName = "MyApp"',
        facet: ['host', 'appName'],
        since: '6 hours',
        until: '1 hour',
        timeseries: true,
        limit: 50,
      };

      mockUseQueryBuilder.mockReturnValue({
        queryComponents: customComponents,
        updateComponents: mockUpdateComponents,
        validationResult: { isValid: true },
      });

      render(<NRQLQueryBuilder {...defaultProps} />);

      expect(screen.getByTestId('aggregation-selector')).toHaveValue('average');
      expect(screen.getByTestId('field-selector')).toHaveValue('duration');
      expect(screen.getByTestId('event-type-selector')).toHaveValue('Span');
      expect(screen.getByDisplayValue('appName = "MyApp"')).toBeInTheDocument();
      expect(screen.getByDisplayValue('host, appName')).toBeInTheDocument();
      expect(screen.getByTestId('time-range-selector-since')).toHaveValue('6 hours');
      expect(screen.getByTestId('time-range-selector-until')).toHaveValue('1 hour');
      expect(screen.getByRole('switch')).toBeChecked();
      expect(screen.getByRole('spinbutton')).toHaveValue(50);
    });
  });

  describe('accessibility', () => {
    it('should have proper aria labels', () => {
      render(<NRQLQueryBuilder {...defaultProps} />);

      expect(screen.getByLabelText('WHERE clause')).toBeInTheDocument();
      expect(screen.getByLabelText('FACET clause')).toBeInTheDocument();
      expect(screen.getByLabelText('Enable time series')).toBeInTheDocument();
      expect(screen.getByLabelText('Result limit')).toBeInTheDocument();
    });

    it('should have proper input constraints for limit', () => {
      render(<NRQLQueryBuilder {...defaultProps} />);

      const limitInput = screen.getByRole('spinbutton');
      expect(limitInput).toHaveAttribute('min', '1');
      expect(limitInput).toHaveAttribute('max', '1000');
      expect(limitInput).toHaveAttribute('type', 'number');
    });
  });

  describe('hook integration', () => {
    it('should call useQueryBuilder with correct props', () => {
      render(<NRQLQueryBuilder {...defaultProps} />);

      expect(mockUseQueryBuilder).toHaveBeenCalledWith({
        initialQuery: defaultProps.value,
        onChange: defaultProps.onChange,
      });
    });

    it('should pass correct initial query to hook', () => {
      const customQuery = 'SELECT average(duration) FROM Span SINCE 2 hours ago';
      render(<NRQLQueryBuilder {...defaultProps} value={customQuery} />);

      expect(mockUseQueryBuilder).toHaveBeenCalledWith({
        initialQuery: customQuery,
        onChange: defaultProps.onChange,
      });
    });
  });
}); 