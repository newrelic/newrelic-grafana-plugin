import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ConfigEditor } from '../ConfigEditor';
import { NewRelicDataSourceOptions, NewRelicSecureJsonData } from '../../types';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';

// Mock the validation utilities
jest.mock('../../utils/validation', () => ({
  validateApiKey: jest.fn(),
  validateAccountId: jest.fn(),
  validateConfiguration: jest.fn(),
}));

// Mock the logger
jest.mock('../../utils/logger', () => ({
  logger: {
    warn: jest.fn(),
    info: jest.fn(),
  },
}));

const mockValidation = require('../../utils/validation');

type MockProps = DataSourcePluginOptionsEditorProps<NewRelicDataSourceOptions, NewRelicSecureJsonData>;

describe('ConfigEditor', () => {
  const defaultProps: MockProps = {
    options: {
      id: 1,
      uid: 'test-uid',
      name: 'Test New Relic',
      type: 'newrelic-grafana-plugin',
      typeName: 'New Relic',
      access: 'proxy',
      url: '',
      user: '',
      database: '',
      basicAuth: false,
      basicAuthUser: '',
      withCredentials: false,
      isDefault: false,
      orgId: 1,
      typeLogoUrl: '',
      jsonData: {} as NewRelicDataSourceOptions,
      secureJsonFields: {},
      secureJsonData: {} as NewRelicSecureJsonData,
      readOnly: false,
    },
    onOptionsChange: jest.fn(),
  };

  beforeEach(() => {
    jest.clearAllMocks();
    // Set up default validation mocks
    mockValidation.validateApiKey.mockReturnValue({ isValid: true });
    mockValidation.validateAccountId.mockReturnValue({ isValid: true });
    mockValidation.validateConfiguration.mockReturnValue({ isValid: true });
  });

  describe('Rendering', () => {
    it('should render all configuration fields', () => {
      render(<ConfigEditor {...defaultProps} />);

      expect(screen.getByLabelText('New Relic API Key')).toBeInTheDocument();
      expect(screen.getByLabelText('New Relic Account ID')).toBeInTheDocument();
      expect(screen.getByLabelText('New Relic Region')).toBeInTheDocument();
    });

    it('should show help text for each field', () => {
      render(<ConfigEditor {...defaultProps} />);

      expect(screen.getByText(/You can find your API key in your New Relic account settings/)).toBeInTheDocument();
      expect(screen.getByText(/Your account ID can be found in the New Relic URL/)).toBeInTheDocument();
      expect(screen.getByText(/Choose US for accounts in the United States/)).toBeInTheDocument();
    });

    it('should show success alert when configuration is complete', () => {
      const propsWithData: MockProps = {
        ...defaultProps,
        options: {
          ...defaultProps.options,
          secureJsonData: {
            apiKey: 'test-api-key',
            accountID: '1234567',
          },
        },
      };

      render(<ConfigEditor {...propsWithData} />);

      expect(screen.getByText('Configuration Complete')).toBeInTheDocument();
      expect(screen.getByText('Your New Relic data source is properly configured and ready to use.')).toBeInTheDocument();
    });
  });

  describe('API Key Field', () => {
    it('should handle API key changes', async () => {
      const user = userEvent.setup();
      render(<ConfigEditor {...defaultProps} />);

      const apiKeyInput = screen.getByTestId('api-key-input');
      await user.type(apiKeyInput, 'test-api-key');

      expect(defaultProps.onOptionsChange).toHaveBeenCalledWith({
        ...defaultProps.options,
        secureJsonData: {
          apiKey: 'test-api-key',
        },
      });
    });

    it('should validate API key on change', async () => {
      const user = userEvent.setup();
      mockValidation.validateApiKey.mockReturnValue({
        isValid: false,
        message: 'Invalid API key format',
      });

      render(<ConfigEditor {...defaultProps} />);

      const apiKeyInput = screen.getByTestId('api-key-input');
      await user.type(apiKeyInput, 'invalid-key');

      await waitFor(() => {
        expect(mockValidation.validateApiKey).toHaveBeenCalledWith('invalid-key');
      });
    });

    it('should handle API key reset', async () => {
      const user = userEvent.setup();
      const propsWithApiKey = {
        ...defaultProps,
        options: {
          ...defaultProps.options,
          secureJsonFields: { apiKey: true },
          secureJsonData: { apiKey: 'existing-key' },
        },
      };

      render(<ConfigEditor {...propsWithApiKey} />);

      // Find and click the reset button (this might need adjustment based on SecretInput implementation)
      const resetButton = screen.getByRole('button', { name: /reset/i });
      await user.click(resetButton);

      expect(defaultProps.onOptionsChange).toHaveBeenCalledWith({
        ...propsWithApiKey.options,
        secureJsonFields: {
          apiKey: false,
        },
        secureJsonData: {
          apiKey: '',
        },
      });
    });
  });

  describe('Account ID Field', () => {
    it('should handle account ID changes', async () => {
      const user = userEvent.setup();
      render(<ConfigEditor {...defaultProps} />);

      const accountIdInput = screen.getByTestId('account-id-input');
      await user.type(accountIdInput, '1234567');

      expect(defaultProps.onOptionsChange).toHaveBeenCalledWith({
        ...defaultProps.options,
        secureJsonData: {
          accountID: '1234567',
        },
      });
    });

    it('should filter non-numeric characters from account ID', async () => {
      const user = userEvent.setup();
      render(<ConfigEditor {...defaultProps} />);

      const accountIdInput = screen.getByTestId('account-id-input');
      await user.type(accountIdInput, 'abc123def');

      expect(defaultProps.onOptionsChange).toHaveBeenCalledWith({
        ...defaultProps.options,
        secureJsonData: {
          accountID: '123',
        },
      });
    });

    it('should validate account ID on change', async () => {
      const user = userEvent.setup();
      mockValidation.validateAccountId.mockReturnValue({
        isValid: false,
        message: 'Invalid account ID',
      });

      render(<ConfigEditor {...defaultProps} />);

      const accountIdInput = screen.getByTestId('account-id-input');
      await user.type(accountIdInput, '123');

      await waitFor(() => {
        expect(mockValidation.validateAccountId).toHaveBeenCalledWith('123');
      });
    });
  });

  describe('Region Selection', () => {
    it('should handle region changes', async () => {
      const user = userEvent.setup();
      render(<ConfigEditor {...defaultProps} />);

      const regionSelect = screen.getByTestId('region-select');
      await user.click(regionSelect);

      // Select US region
      const usOption = screen.getByText('United States (US)');
      await user.click(usOption);

      expect(defaultProps.onOptionsChange).toHaveBeenCalledWith({
        ...defaultProps.options,
        jsonData: {
          region: 'US',
        },
      });
    });

    it('should show current region selection', () => {
      const propsWithRegion: MockProps = {
        ...defaultProps,
        options: {
          ...defaultProps.options,
          jsonData: { region: 'EU' as const },
        },
      };

      render(<ConfigEditor {...propsWithRegion} />);

      // Note: This test might need adjustment based on how Select component renders
      expect(screen.getByText('Europe (EU)')).toBeInTheDocument();
    });
  });

  describe('Validation and Error Handling', () => {
    it('should show validation errors', () => {
      mockValidation.validateConfiguration.mockReturnValue({
        isValid: false,
        message: 'Configuration is incomplete',
      });

      render(<ConfigEditor {...defaultProps} />);

      expect(screen.getByText('Configuration Error')).toBeInTheDocument();
      expect(screen.getByText('Configuration is incomplete')).toBeInTheDocument();
    });

    it('should show field-specific validation errors', async () => {
      const user = userEvent.setup();
      mockValidation.validateApiKey.mockReturnValue({
        isValid: false,
        message: 'API key is too short',
      });

      render(<ConfigEditor {...defaultProps} />);

      const apiKeyInput = screen.getByTestId('api-key-input');
      await user.type(apiKeyInput, 'short');

      await waitFor(() => {
        expect(screen.getByText('API key is too short')).toBeInTheDocument();
      });
    });

    it('should clear validation errors when field becomes valid', async () => {
      const user = userEvent.setup();
      
      // First, make validation fail
      mockValidation.validateApiKey.mockReturnValue({
        isValid: false,
        message: 'API key is too short',
      });

      render(<ConfigEditor {...defaultProps} />);

      const apiKeyInput = screen.getByTestId('api-key-input');
      await user.type(apiKeyInput, 'short');

      await waitFor(() => {
        expect(screen.getByText('API key is too short')).toBeInTheDocument();
      });

      // Then make validation pass
      mockValidation.validateApiKey.mockReturnValue({ isValid: true });
      
      await user.clear(apiKeyInput);
      await user.type(apiKeyInput, 'NRAK1234567890abcdef1234567890abcdef1234');

      await waitFor(() => {
        expect(screen.queryByText('API key is too short')).not.toBeInTheDocument();
      });
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA labels', () => {
      render(<ConfigEditor {...defaultProps} />);

      expect(screen.getByLabelText('New Relic API Key')).toBeInTheDocument();
      expect(screen.getByLabelText('New Relic Account ID')).toBeInTheDocument();
      expect(screen.getByLabelText('New Relic Region')).toBeInTheDocument();
    });

    it('should have proper ARIA descriptions', () => {
      render(<ConfigEditor {...defaultProps} />);

      const apiKeyInput = screen.getByLabelText('New Relic API Key');
      expect(apiKeyInput).toHaveAttribute('aria-describedby', 'api-key-help');

      const accountIdInput = screen.getByLabelText('New Relic Account ID');
      expect(accountIdInput).toHaveAttribute('aria-describedby', 'account-id-help');
    });

    it('should mark invalid fields with aria-invalid', async () => {
      const user = userEvent.setup();
      mockValidation.validateApiKey.mockReturnValue({
        isValid: false,
        message: 'Invalid API key',
      });

      render(<ConfigEditor {...defaultProps} />);

      const apiKeyInput = screen.getByTestId('api-key-input');
      await user.type(apiKeyInput, 'invalid');

      await waitFor(() => {
        expect(apiKeyInput).toHaveAttribute('aria-invalid', 'true');
      });
    });
  });

  describe('Integration', () => {
    it('should handle complete configuration flow', async () => {
      const user = userEvent.setup();
      render(<ConfigEditor {...defaultProps} />);

      // Enter API key
      const apiKeyInput = screen.getByTestId('api-key-input');
      await user.type(apiKeyInput, 'NRAK1234567890abcdef1234567890abcdef1234');

      // Enter account ID
      const accountIdInput = screen.getByTestId('account-id-input');
      await user.type(accountIdInput, '1234567');

      // Select region
      const regionSelect = screen.getByTestId('region-select');
      await user.click(regionSelect);
      const usOption = screen.getByText('United States (US)');
      await user.click(usOption);

      // Verify all changes were called
      expect(defaultProps.onOptionsChange).toHaveBeenCalledTimes(3);
    });
  });
}); 