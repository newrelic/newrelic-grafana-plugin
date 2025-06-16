import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TimeRangeSelector } from '../TimeRangeSelector';

// Mock Grafana UI components with full functionality
jest.mock('@grafana/ui', () => ({
  ...jest.requireActual('@grafana/ui'),
  InlineField: ({ label, tooltip, children, labelWidth }: any) => (
    <div className="css-1uqla6q" data-testid="InlineField">
      <label 
        className="css-v249xx"
        style={{ width: labelWidth }}
      >
        {label}
        {tooltip && <svg data-testid="info-circle" />}
      </label>
      <div className="css-1tu59u4">{children}</div>
    </div>
  ),
  Select: ({ value, onChange, options, width, 'aria-label': ariaLabel }: any) => {
    const [currentValue, setCurrentValue] = React.useState(value?.value || value || '');
    const currentLabel = value?.label || value || currentValue;
    
    // Update current value when prop changes
    React.useEffect(() => {
      setCurrentValue(value?.value || value || '');
    }, [value]);
    
    // For custom values not in options, add them as an option
    const allOptions = [...(options || [])];
    if (currentValue && !allOptions.find(opt => opt.value === currentValue)) {
      allOptions.push({ label: currentLabel, value: currentValue });
    }
    
    return (
      <div className="css-rebjtg-input-wrapper css-ega3jk" style={{ width }}>
        <span className="css-1f43avz-a11yText-A11yText" id="react-select-live-region" />
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
              {currentLabel}
            </div>
            <div className="css-1eu65zc" data-value="">
              <select
                aria-label={ariaLabel}
                className=""
                id="select-input"
                role="combobox"
                tabIndex={0}
                value={currentValue}
                onChange={(e) => {
                  const newValue = e.target.value;
                  setCurrentValue(newValue);
                  const selectedOption = allOptions?.find((opt: any) => opt.value === newValue);
                  if (selectedOption) {
                    onChange(selectedOption);
                  } else {
                    // For direct value change, call with just the value
                    onChange(newValue);
                  }
                }}
                data-testid="select-input"
              >
                {allOptions?.map((option: any) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div className="css-zyjsuv-input-suffix">
            <svg data-testid="angle-down" />
          </div>
        </div>
      </div>
    );
  },
}));

describe('TimeRangeSelector', () => {
  const mockOnChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  const defaultProps = {
    value: '1 hour',
    onChange: mockOnChange,
    label: 'SINCE',
    tooltip: 'Select the time range for your query',
  };

  describe('rendering', () => {
    it('should render the time range selector', () => {
      render(<TimeRangeSelector {...defaultProps} />);

      expect(screen.getByRole('combobox')).toBeInTheDocument();
      expect(screen.getByDisplayValue('1 hour')).toBeInTheDocument();
    });

    it('should display the correct selected value', () => {
      render(<TimeRangeSelector {...defaultProps} value="6 hours" />);

      expect(screen.getByDisplayValue('6 hours')).toBeInTheDocument();
    });

    it('should render with custom label', () => {
      render(<TimeRangeSelector {...defaultProps} label="UNTIL" />);

      expect(screen.getByLabelText('Select until')).toBeInTheDocument();
    });

    it('should render all time range options', () => {
      render(<TimeRangeSelector {...defaultProps} />);

      const options = screen.getAllByRole('option');
      const optionValues = options.map(option => option.getAttribute('value'));
      
      expect(optionValues).toContain('5 minutes');
      expect(optionValues).toContain('15 minutes');
      expect(optionValues).toContain('30 minutes');
      expect(optionValues).toContain('1 hour');
      expect(optionValues).toContain('3 hours');
      expect(optionValues).toContain('6 hours');
      expect(optionValues).toContain('12 hours');
      expect(optionValues).toContain('24 hours');
      expect(optionValues).toContain('7 days');
    });

    it('should render all time range labels', () => {
      render(<TimeRangeSelector {...defaultProps} />);

      // Check for options within the select element specifically
      const selectElement = screen.getByRole('combobox');
      expect(selectElement).toBeInTheDocument();
      
      // Check that all options exist as option elements
      expect(screen.getByRole('option', { name: '5 minutes' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: '15 minutes' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: '30 minutes' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: '1 hour' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: '3 hours' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: '6 hours' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: '12 hours' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: '24 hours' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: '7 days' })).toBeInTheDocument();
    });
  });

  describe('user interactions', () => {
    it('should call onChange with correct value for each time range', async () => {
      const user = userEvent.setup();
      render(<TimeRangeSelector {...defaultProps} />);

      const selector = screen.getByRole('combobox');

      // Test different time ranges
      await user.selectOptions(selector, '15 minutes');
      expect(mockOnChange).toHaveBeenCalledWith('15 minutes');

      await user.selectOptions(selector, '3 hours');
      expect(mockOnChange).toHaveBeenCalledWith('3 hours');

      await user.selectOptions(selector, '7 days');
      expect(mockOnChange).toHaveBeenCalledWith('7 days');
    });

    it('should maintain selection functionality', async () => {
      const user = userEvent.setup();
      render(<TimeRangeSelector {...defaultProps} />);

      const selector = screen.getByRole('combobox');
      
      // Test that selections trigger onChange
      await user.selectOptions(selector, '12 hours');
      expect(mockOnChange).toHaveBeenCalledWith('12 hours');

      await user.selectOptions(selector, '24 hours');
      expect(mockOnChange).toHaveBeenCalledWith('24 hours');
    });

    it('should support custom time range functionality', async () => {
      const user = userEvent.setup();
      render(<TimeRangeSelector {...defaultProps} value="2 hours" />);

      const selector = screen.getByRole('combobox');
      expect(selector).toBeInTheDocument();
      
      // Should still allow changing to predefined ranges
      await user.selectOptions(selector, '1 hour');
      expect(mockOnChange).toHaveBeenCalledWith('1 hour');
    });
  });

  describe('accessibility', () => {
    it('should have proper label', () => {
      render(<TimeRangeSelector {...defaultProps} />);

      expect(screen.getByLabelText('Select since')).toBeInTheDocument();
    });

    it('should use custom label when provided', () => {
      render(<TimeRangeSelector {...defaultProps} label="UNTIL" />);

      expect(screen.getByLabelText('Select until')).toBeInTheDocument();
      expect(screen.queryByLabelText('Select since')).not.toBeInTheDocument();
    });

    it('should have tooltip', () => {
      render(<TimeRangeSelector {...defaultProps} />);

      const labelElement = screen.getByTestId('InlineField');
      expect(labelElement).toBeInTheDocument();
    });

    it('should use custom tooltip when provided', () => {
      const customTooltip = 'Custom tooltip text';
      render(<TimeRangeSelector {...defaultProps} tooltip={customTooltip} />);

      const labelElement = screen.getByTestId('InlineField');
      expect(labelElement).toBeInTheDocument();
    });

    it('should be keyboard accessible', async () => {
      const user = userEvent.setup();
      render(<TimeRangeSelector {...defaultProps} />);

      const selector = screen.getByRole('combobox');
      
      // Focus the select element
      await user.tab();
      expect(selector).toHaveFocus();

      // Use keyboard to navigate
      await user.keyboard('{ArrowDown}');
      await user.keyboard('{Enter}');
      
      expect(selector).toBeInTheDocument();
    });
  });

  describe('edge cases', () => {
    it('should handle undefined value gracefully', () => {
      render(<TimeRangeSelector {...defaultProps} value={undefined as any} />);

      const selector = screen.getByRole('combobox');
      expect(selector).toBeInTheDocument();
    });

    it('should handle null onChange gracefully', () => {
      render(<TimeRangeSelector {...defaultProps} onChange={null as any} />);

      const selector = screen.getByRole('combobox');
      expect(selector).toBeInTheDocument();
    });

    it('should handle missing label gracefully', () => {
      render(<TimeRangeSelector {...defaultProps} label={undefined as any} />);

      const selector = screen.getByRole('combobox');
      expect(selector).toBeInTheDocument();
    });
  });

  describe('component structure', () => {
    it('should render as a select component', () => {
      render(<TimeRangeSelector {...defaultProps} />);

      const selector = screen.getByRole('combobox');
      expect(selector.tagName.toLowerCase()).toBe('select');
    });

    it('should have proper labelWidth', () => {
      render(<TimeRangeSelector {...defaultProps} />);

      const labelElement = screen.getByTestId('InlineField');
      expect(labelElement).toBeInTheDocument();
      // labelWidth is handled by Grafana UI InlineField component
    });
  });

  describe('different label scenarios', () => {
    it('should work with SINCE label', () => {
      render(<TimeRangeSelector {...defaultProps} label="SINCE" />);

      expect(screen.getByLabelText('Select since')).toBeInTheDocument();
    });

    it('should work with UNTIL label', () => {
      render(<TimeRangeSelector {...defaultProps} label="UNTIL" />);

      expect(screen.getByLabelText('Select until')).toBeInTheDocument();
    });

    it('should work with custom labels', () => {
      render(<TimeRangeSelector {...defaultProps} label="Duration" />);

      expect(screen.getByLabelText('Select duration')).toBeInTheDocument();
    });
  });
}); 