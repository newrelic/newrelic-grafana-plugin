import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { NRQLQueryBuilder } from '../NRQLQueryBuilder';

// Mock the useQueryBuilder hook
jest.mock('../../../hooks/useQueryBuilder');
const mockUseQueryBuilder = require('../../../hooks/useQueryBuilder').useQueryBuilder as jest.MockedFunction<any>;

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
    useGrafanaTime: false,
  };

  describe('rendering', () => {
    it('should render all query builder fields', () => {
      render(<NRQLQueryBuilder {...defaultProps} />);

      // Check that labels are present
      expect(screen.getByText('Aggregation')).toBeInTheDocument();
      expect(screen.getByText('FROM')).toBeInTheDocument();
      expect(screen.getByText('WHERE')).toBeInTheDocument();
      expect(screen.getByText('FACET')).toBeInTheDocument();
      expect(screen.getByText('SINCE')).toBeInTheDocument();
      expect(screen.getByText('LIMIT')).toBeInTheDocument();

      // Check that inputs are present
      expect(screen.getByPlaceholderText('e.g., appName = \'MyApp\' AND duration > 1')).toBeInTheDocument();
      expect(screen.getByPlaceholderText('e.g., host, appName')).toBeInTheDocument();
      expect(screen.getByDisplayValue('100')).toBeInTheDocument();
    });

    it('should render field input when aggregation requires field', () => {
      mockUseQueryBuilder.mockReturnValue({
        queryComponents: { ...defaultQueryComponents, aggregation: 'average', field: 'duration' },
        updateComponents: mockUpdateComponents,
        validationResult: { isValid: true },
      });

      render(<NRQLQueryBuilder {...defaultProps} />);

      expect(screen.getByText('Field')).toBeInTheDocument();
      expect(screen.getByPlaceholderText('e.g., duration, responseTime')).toBeInTheDocument();
    });

    it('should not render field input when aggregation is count', () => {
      render(<NRQLQueryBuilder {...defaultProps} />);

      expect(screen.queryByText('Field')).not.toBeInTheDocument();
    });

    it('should show Grafana time picker info when useGrafanaTime is true', () => {
      render(<NRQLQueryBuilder {...defaultProps} useGrafanaTime={true} />);

      expect(screen.getByText('Using Grafana Dashboard Time Picker')).toBeInTheDocument();
      expect(screen.queryByText('SINCE')).not.toBeInTheDocument();
    });

    it('should show query preview at the bottom', () => {
      render(<NRQLQueryBuilder {...defaultProps} />);

      expect(screen.getByText('Generated Query:')).toBeInTheDocument();
      expect(screen.getByText(defaultProps.value)).toBeInTheDocument();
    });
  });

  describe('validation', () => {
    it('should show validation errors for invalid aggregation', () => {
      mockUseQueryBuilder.mockReturnValue({
        queryComponents: { ...defaultQueryComponents, aggregation: 'xyz' },
        updateComponents: mockUpdateComponents,
        validationResult: { isValid: false },
      });

      render(<NRQLQueryBuilder {...defaultProps} />);

      expect(screen.getByText('Query Builder Validation Errors')).toBeInTheDocument();
    });

    it('should show validation error for missing field when required', () => {
      mockUseQueryBuilder.mockReturnValue({
        queryComponents: { ...defaultQueryComponents, aggregation: 'average', field: '' },
        updateComponents: mockUpdateComponents,
        validationResult: { isValid: false },
      });

      render(<NRQLQueryBuilder {...defaultProps} />);

      expect(screen.getByText('Query Builder Validation Errors')).toBeInTheDocument();
    });
  });

  describe('user interactions', () => {
    it('should call updateComponents when WHERE field changes', async () => {
      const user = userEvent.setup();
      render(<NRQLQueryBuilder {...defaultProps} />);

      const whereInput = screen.getByPlaceholderText('e.g., appName = \'MyApp\' AND duration > 1');
      await user.clear(whereInput);
      await user.type(whereInput, 'test');

      // Just verify that updateComponents was called with where parameter
      await waitFor(() => {
        expect(mockUpdateComponents).toHaveBeenCalledWith(expect.objectContaining({ where: expect.any(String) }));
      });
    });

    it('should call updateComponents when FACET field changes', async () => {
      const user = userEvent.setup();
      render(<NRQLQueryBuilder {...defaultProps} />);

      const facetInput = screen.getByPlaceholderText('e.g., host, appName');
      await user.type(facetInput, 'host');

      // Just verify that updateComponents was called with facet parameter
      await waitFor(() => {
        expect(mockUpdateComponents).toHaveBeenCalledWith(expect.objectContaining({ facet: expect.any(Array) }));
      });
    });

    it('should call updateComponents when LIMIT field changes', async () => {
      const user = userEvent.setup();
      render(<NRQLQueryBuilder {...defaultProps} />);

      const limitInput = screen.getByDisplayValue('100');
      await user.clear(limitInput);
      await user.type(limitInput, '50');

      // Just verify that updateComponents was called with limit parameter
      await waitFor(() => {
        expect(mockUpdateComponents).toHaveBeenCalledWith(expect.objectContaining({ limit: expect.any(Number) }));
      });
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
        until: '',
        timeseries: false,
        limit: 50,
      };

      mockUseQueryBuilder.mockReturnValue({
        queryComponents: customComponents,
        updateComponents: mockUpdateComponents,
        validationResult: { isValid: true },
      });

      render(<NRQLQueryBuilder {...defaultProps} />);

      expect(screen.getByDisplayValue('duration')).toBeInTheDocument();
      expect(screen.getByDisplayValue('appName = "MyApp"')).toBeInTheDocument();
      expect(screen.getByDisplayValue('host, appName')).toBeInTheDocument();
      expect(screen.getByDisplayValue('50')).toBeInTheDocument();
    });
  });

  describe('hook integration', () => {
    it('should call useQueryBuilder with correct props', () => {
      render(<NRQLQueryBuilder {...defaultProps} />);

      expect(mockUseQueryBuilder).toHaveBeenCalledWith({
        initialQuery: defaultProps.value,
        onChange: defaultProps.onChange,
        useGrafanaTime: false,
      });
    });

    it('should pass correct initial query to hook', () => {
      const customQuery = 'SELECT average(duration) FROM Span SINCE 2 hours ago';
      render(<NRQLQueryBuilder {...defaultProps} value={customQuery} />);

      expect(mockUseQueryBuilder).toHaveBeenCalledWith({
        initialQuery: customQuery,
        onChange: defaultProps.onChange,
        useGrafanaTime: false,
      });
    });

    it('should pass useGrafanaTime to hook', () => {
      render(<NRQLQueryBuilder {...defaultProps} useGrafanaTime={true} />);

      expect(mockUseQueryBuilder).toHaveBeenCalledWith({
        initialQuery: defaultProps.value,
        onChange: defaultProps.onChange,
        useGrafanaTime: true,
      });
    });
  });
}); 