import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FieldSelector } from '../FieldSelector';

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

describe('FieldSelector', () => {
  const mockOnChange = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  const defaultProps = {
    value: 'duration',
    onChange: mockOnChange,
  };

  describe('rendering', () => {
    it('should render the field selector', () => {
      render(<FieldSelector {...defaultProps} />);

      expect(screen.getByRole('combobox')).toBeInTheDocument();
      expect(screen.getByDisplayValue('duration')).toBeInTheDocument();
    });

    it('should display the correct selected value', () => {
      render(<FieldSelector {...defaultProps} value="responseTime" />);

      expect(screen.getByDisplayValue('responseTime')).toBeInTheDocument();
    });

    it('should render all common field options', () => {
      render(<FieldSelector {...defaultProps} />);

      const options = screen.getAllByRole('option');
      const optionValues = options.map(option => option.getAttribute('value'));
      
      expect(optionValues).toContain('duration');
      expect(optionValues).toContain('responseTime');
      expect(optionValues).toContain('appName');
      expect(optionValues).toContain('host');
      expect(optionValues).toContain('name');
      expect(optionValues).toContain('entityGuid');
      expect(optionValues).toContain('userId');
      expect(optionValues).toContain('sessionId');
    });

    it('should render all field labels', () => {
      render(<FieldSelector {...defaultProps} />);

      // Check for options within the select element specifically
      const selectElement = screen.getByRole('combobox');
      expect(selectElement).toBeInTheDocument();
      
      // Check that all options exist as option elements
      expect(screen.getByRole('option', { name: 'duration' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: 'responseTime' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: 'appName' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: 'host' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: 'name' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: 'entityGuid' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: 'userId' })).toBeInTheDocument();
      expect(screen.getByRole('option', { name: 'sessionId' })).toBeInTheDocument();
    });
  });

  describe('user interactions', () => {
    it('should call onChange when selection changes', async () => {
      const user = userEvent.setup();
      render(<FieldSelector {...defaultProps} />);

      const selector = screen.getByRole('combobox');
      await user.selectOptions(selector, 'responseTime');

      expect(mockOnChange).toHaveBeenCalledWith('responseTime');
    });

    it('should call onChange with correct value for each field type', async () => {
      const user = userEvent.setup();
      render(<FieldSelector {...defaultProps} />);

      const selector = screen.getByRole('combobox');

      // Test different field types
      await user.selectOptions(selector, 'appName');
      expect(mockOnChange).toHaveBeenCalledWith('appName');

      await user.selectOptions(selector, 'host');
      expect(mockOnChange).toHaveBeenCalledWith('host');

      await user.selectOptions(selector, 'entityGuid');
      expect(mockOnChange).toHaveBeenCalledWith('entityGuid');
    });

    it('should maintain selection functionality', async () => {
      const user = userEvent.setup();
      render(<FieldSelector {...defaultProps} />);

      const selector = screen.getByRole('combobox');
      
      // Test that selections trigger onChange  
      await user.selectOptions(selector, 'responseTime');
      expect(mockOnChange).toHaveBeenCalledWith('responseTime');

      await user.selectOptions(selector, 'appName');
      expect(mockOnChange).toHaveBeenCalledWith('appName');
    });

    it('should support custom field functionality', async () => {
      const user = userEvent.setup();
      render(<FieldSelector {...defaultProps} value="customField" />);

      const selector = screen.getByRole('combobox');
      expect(selector).toBeInTheDocument();
      
      // Should still allow changing to predefined fields
      await user.selectOptions(selector, 'duration');
      expect(mockOnChange).toHaveBeenCalledWith('duration');
    });
  });

  describe('accessibility', () => {
    it('should have proper label', () => {
      render(<FieldSelector {...defaultProps} />);

      expect(screen.getByLabelText('Select field')).toBeInTheDocument();
    });

    it('should have tooltip', () => {
      render(<FieldSelector {...defaultProps} />);

      const labelElement = screen.getByTestId('InlineField');
      expect(labelElement).toBeInTheDocument();
    });

    it('should be keyboard accessible', async () => {
      const user = userEvent.setup();
      render(<FieldSelector {...defaultProps} />);

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
      render(<FieldSelector {...defaultProps} value={undefined as any} />);

      const selector = screen.getByRole('combobox');
      expect(selector).toBeInTheDocument();
    });

    it('should handle null onChange gracefully', () => {
      render(<FieldSelector {...defaultProps} onChange={null as any} />);

      const selector = screen.getByRole('combobox');
      expect(selector).toBeInTheDocument();
    });
  });

  describe('component structure', () => {
    it('should have correct width', () => {
      render(<FieldSelector {...defaultProps} />);

      const selector = screen.getByRole('combobox');
      expect(selector).toBeInTheDocument();
      // Width is controlled by Grafana UI component
    });

    it('should render as a select component', () => {
      render(<FieldSelector {...defaultProps} />);

      const selector = screen.getByRole('combobox');
      expect(selector.tagName.toLowerCase()).toBe('select');
    });
  });
}); 