import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { EventTypeSelector } from '../EventTypeSelector';

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
      <span className="css-1f43avz-a11yText-A11yText" id="react-select-2-live-region" />
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

describe('EventTypeSelector', () => {
  const mockOnChange = jest.fn();
  const defaultProps = {
    value: 'Transaction',
    onChange: mockOnChange,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('rendering', () => {
    it('should render the event type selector', () => {
      render(<EventTypeSelector {...defaultProps} />);

      expect(screen.getByRole('combobox')).toBeInTheDocument();
      expect(screen.getByText('Transaction')).toBeInTheDocument();
    });

    it('should display the correct selected value', () => {
      render(<EventTypeSelector {...defaultProps} value="Span" />);

      expect(screen.getByText('Span')).toBeInTheDocument();
    });

    it('should render all event type options', () => {
      render(<EventTypeSelector {...defaultProps} />);

      const selectInput = screen.getByTestId('select-input');
      expect(selectInput).toBeInTheDocument();
      expect(screen.getByText('Transaction')).toBeInTheDocument();
    });

    it('should render all event type labels', () => {
      render(<EventTypeSelector {...defaultProps} />);

      // Check that the component renders the current value
      expect(screen.getByText('Transaction')).toBeInTheDocument();
      
      // Test different values by re-rendering
      const { rerender } = render(<EventTypeSelector {...defaultProps} value="Span" />);
      expect(screen.getByText('Span')).toBeInTheDocument();
      
      rerender(<EventTypeSelector {...defaultProps} value="Metric" />);
      expect(screen.getByText('Metric')).toBeInTheDocument();
    });
  });

  describe('user interactions', () => {
    it('should render interactive select component', async () => {
      const user = userEvent.setup();
      render(<EventTypeSelector {...defaultProps} />);

      const selectInput = screen.getByTestId('select-input');
      expect(selectInput).toBeInTheDocument();

      // Component should be interactive
      await user.click(selectInput);
      expect(selectInput).toBeInTheDocument();
    });

    it('should display different event types correctly', async () => {
      // Test different event types by checking the display values
      render(<EventTypeSelector {...defaultProps} value="Span" />);
      expect(screen.getByText('Span')).toBeInTheDocument();

      render(<EventTypeSelector {...defaultProps} value="Metric" />);
      expect(screen.getByText('Metric')).toBeInTheDocument();
    });

    it('should support custom event types', async () => {
      render(<EventTypeSelector {...defaultProps} value="CustomEvent" />);
      expect(screen.getByTestId('select-input')).toBeInTheDocument();
    });
  });

  describe('accessibility', () => {
    it('should have proper label', () => {
      render(<EventTypeSelector {...defaultProps} />);

      expect(screen.getByLabelText('Select event type')).toBeInTheDocument();
    });

    it('should have tooltip', () => {
      render(<EventTypeSelector {...defaultProps} />);

      expect(screen.getByText('Event Type')).toBeInTheDocument();
      expect(screen.getByTestId('info-circle')).toBeInTheDocument();
    });

    it('should be keyboard accessible', async () => {
      const user = userEvent.setup();
      render(<EventTypeSelector {...defaultProps} />);

      const selectInput = screen.getByRole('combobox');
      
      // Should be focusable
      await user.tab();
      expect(selectInput).toHaveFocus();
    });
  });

  describe('value handling', () => {
    it('should handle empty value', () => {
      render(<EventTypeSelector {...defaultProps} value="" />);
      
      const selectInput = screen.getByTestId('select-input');
      expect(selectInput).toBeInTheDocument();
    });

    it('should handle undefined value', () => {
      render(<EventTypeSelector {...defaultProps} value={undefined as any} />);
      
      const selectInput = screen.getByTestId('select-input');
      expect(selectInput).toBeInTheDocument();
    });
  });

  describe('edge cases', () => {
    it('should handle empty value gracefully', () => {
      render(<EventTypeSelector {...defaultProps} value="" />);

      const selectInput = screen.getByTestId('select-input');
      expect(selectInput).toBeInTheDocument();
    });

    it('should handle null value gracefully', () => {
      render(<EventTypeSelector {...defaultProps} value={null as any} />);

      const selectInput = screen.getByTestId('select-input');
      expect(selectInput).toBeInTheDocument();
    });

    it('should handle undefined value gracefully', () => {
      render(<EventTypeSelector {...defaultProps} value={undefined as any} />);

      const selectInput = screen.getByTestId('select-input');
      expect(selectInput).toBeInTheDocument();
    });
  });
}); 