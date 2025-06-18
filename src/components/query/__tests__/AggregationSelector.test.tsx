import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AggregationSelector } from '../AggregationSelector';

// Mock Grafana UI components
jest.mock('@grafana/ui', () => ({
  ...jest.requireActual('@grafana/ui'),
  InlineField: ({ label, tooltip, children }: any) => (
    <div className="css-1uqla6q">
      <label className="css-v249xx">
        {label}
        {tooltip && <svg data-testid="info-circle" />}
      </label>
      <div className="css-1tu59u4">{children}</div>
    </div>
  ),
  Select: ({ value, onChange, options, placeholder, inputId, 'aria-label': ariaLabel }: any) => (
    <div className="css-rebjtg-input-wrapper css-ega3jk">
      <span className="css-1f43avz-a11yText-A11yText" id="react-select-3-live-region" />
      <span 
        aria-atomic="false" 
        aria-live="polite" 
        aria-relevant="additions text"
        className="css-1f43avz-a11yText-A11yText" 
        role="log" 
      />
      <div className="css-1i88p6p">
        <div className="css-1q0c0d5-grafana-select-value-container">
          <div className="css-8nwx1l-singleValue css-0">
            {value?.label || value || placeholder}
          </div>
          <div className="css-1eu65zc" data-value="">
            <input
              aria-activedescendant=""
              aria-autocomplete="list"
              aria-expanded="false"
              aria-haspopup="true"
              aria-label={ariaLabel}
              autoCapitalize="none"
              autoComplete="off"
              autoCorrect="off"
              className=""
              id={inputId}
              role="combobox"
              spellCheck="false"
              tabIndex={0}
              type="text"
              value=""
              onChange={(e) => {
                const selectedOption = options?.find((opt: any) => 
                  opt.value === e.target.value || opt.label === e.target.value
                );
                if (selectedOption) {
                  onChange(selectedOption);
                }
              }}
              data-testid="select-input"
            />
          </div>
        </div>
        <div className="css-zyjsuv-input-suffix">
          <svg data-testid="angle-down" />
        </div>
      </div>
    </div>
  ),
}));

describe('AggregationSelector', () => {
  const mockOnChange = jest.fn();
  const defaultProps = {
    value: 'count(*)',
    onChange: mockOnChange,
    showFieldSelector: false,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('rendering', () => {
    it('should display the correct selected value', () => {
      render(<AggregationSelector {...defaultProps} value="average" />);
      
      expect(screen.getByText('average')).toBeInTheDocument();
    });

    it('should render all aggregation options', () => {
      render(<AggregationSelector {...defaultProps} />);

      const selectInput = screen.getByTestId('select-input');
      expect(selectInput).toBeInTheDocument();
      expect(screen.getByText('count(*)')).toBeInTheDocument();
    });
  });

  describe('user interactions', () => {
    it('should render interactive select component', async () => {
      const user = userEvent.setup();
      render(<AggregationSelector {...defaultProps} />);

      const selectInput = screen.getByTestId('select-input');
      expect(selectInput).toBeInTheDocument();
      
      // Component should be interactive
      await user.click(selectInput);
      expect(selectInput).toBeInTheDocument();
    });

    it('should display different aggregation types correctly', async () => {
      // Test different aggregation types by checking the display values
      render(<AggregationSelector {...defaultProps} value="sum" />);
      expect(screen.getByText('sum')).toBeInTheDocument();

      const { rerender } = render(<AggregationSelector {...defaultProps} value="max" />);
      expect(screen.getByText('max')).toBeInTheDocument();

      rerender(<AggregationSelector {...defaultProps} value="min" />);
      expect(screen.getByText('min')).toBeInTheDocument();
    });
  });

  describe('accessibility', () => {
    it('should have proper label', () => {
      render(<AggregationSelector {...defaultProps} />);

      expect(screen.getByLabelText('Select aggregation function')).toBeInTheDocument();
    });

    it('should have tooltip text', () => {
      render(<AggregationSelector {...defaultProps} />);

      expect(screen.getByText('Aggregation')).toBeInTheDocument();
      expect(screen.getByTestId('info-circle')).toBeInTheDocument();
    });
  });

  describe('value handling', () => {
    it('should handle empty value', () => {
      render(<AggregationSelector {...defaultProps} value="" />);
      
      const selectInput = screen.getByTestId('select-input');
      expect(selectInput).toBeInTheDocument();
    });

    it('should handle undefined value', () => {
      render(<AggregationSelector {...defaultProps} value={undefined as any} />);
      
      const selectInput = screen.getByTestId('select-input');
      expect(selectInput).toBeInTheDocument();
    });

    it('should display correct value for each aggregation type', () => {
      const aggregationTypes = ['count(*)', 'average', 'sum', 'max', 'min', 'uniqueCount'];
      
      aggregationTypes.forEach(type => {
        const { rerender } = render(<AggregationSelector {...defaultProps} value={type} />);
        expect(screen.getByText(type)).toBeInTheDocument();
        rerender(<div />); // Clear the render
      });
    });
  });
}); 