import React, { ChangeEvent, useState, useCallback, useMemo } from 'react';
import { InlineField, InlineFieldRow, SecretInput, Select, Alert } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps, SelectableValue } from '@grafana/data';
import { NewRelicDataSourceOptions, NewRelicSecureJsonData, NEW_RELIC_REGIONS } from '../types';
import { validateApiKeyDetailed, validateAccountIdDetailed, validateConfiguration } from '../utils/validation';
import { logger } from '../utils/logger';

interface Props extends DataSourcePluginOptionsEditorProps<NewRelicDataSourceOptions, NewRelicSecureJsonData> {}

/**
 * Configuration editor component for the New Relic data source
 * Handles API key, account ID, and region configuration
 */
export function ConfigEditor({ onOptionsChange, options }: Props) {
  const { secureJsonFields, secureJsonData, jsonData } = options;
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});

  // Region options for the select dropdown
  const regionOptions: Array<SelectableValue<string>> = [
    { label: 'United States (US)', value: NEW_RELIC_REGIONS.US },
    { label: 'Europe (EU)', value: NEW_RELIC_REGIONS.EU },
  ];

  /**
   * Validates and updates the API key
   */
  const handleApiKeyChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    const apiKey = event.target.value;
    const validation = validateApiKeyDetailed(apiKey);
    
    // Update validation errors
    setValidationErrors(prev => ({
      ...prev,
      apiKey: validation.isValid ? '' : validation.message || 'Invalid API key',
    }));

    // Update the options
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...secureJsonData,
        apiKey,
      },
    });

    if (!validation.isValid) {
      logger.warn('API key validation failed', { error: validation.message });
    }
  }, [options, secureJsonData, onOptionsChange]);

  /**
   * Resets the API key field
   */
  const handleApiKeyReset = useCallback(() => {
    setValidationErrors(prev => ({ ...prev, apiKey: '' }));
    
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...secureJsonFields,
        apiKey: false,
      },
      secureJsonData: {
        ...secureJsonData,
        apiKey: '',
      },
    });

    logger.info('API key reset');
  }, [options, secureJsonFields, secureJsonData, onOptionsChange]);

  /**
   * Validates and updates the account ID
   */
  const handleAccountIdChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    const rawValue = event.target.value;
    // Only allow numeric input
    const numericValue = rawValue.replace(/[^0-9]/g, '');
    const validation = validateAccountIdDetailed(numericValue);

    // Update validation errors
    setValidationErrors(prev => ({
      ...prev,
      accountID: validation.isValid ? '' : validation.message || 'Invalid account ID',
    }));

    // Update the options
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...secureJsonData,
        accountID: numericValue,
      },
    });

    if (!validation.isValid) {
      logger.warn('Account ID validation failed', { error: validation.message });
    }
  }, [options, secureJsonData, onOptionsChange]);

  /**
   * Resets the account ID field
   */
  const handleAccountIdReset = useCallback(() => {
    setValidationErrors(prev => ({ ...prev, accountID: '' }));
    
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...secureJsonFields,
        accountID: false,
      },
      secureJsonData: {
        ...secureJsonData,
        accountID: '',
      },
    });

    logger.info('Account ID reset');
  }, [options, secureJsonFields, secureJsonData, onOptionsChange]);

  /**
   * Updates the selected region
   */
  const handleRegionChange = useCallback((selectedOption: SelectableValue<string>) => {
    const region = selectedOption?.value as 'US' | 'EU' | undefined;
    
    onOptionsChange({
      ...options,
      jsonData: {
        ...jsonData,
        region,
      },
    });

    logger.info('Region changed', { region });
  }, [options, jsonData, onOptionsChange]);

  // Validate overall configuration
  const configValidation = useMemo(() => {
    // If API key and account ID are already configured (secureJsonFields), don't show validation errors
    const isApiKeyConfigured = !!secureJsonFields?.apiKey;
    const isAccountIdConfigured = !!secureJsonFields?.accountID;
    
    // Only validate if we have actual values to validate (when user is entering new values)
    const currentApiKey = secureJsonData?.apiKey || '';
    const currentAccountId = secureJsonData?.accountID || '';
    
    // If fields are configured but we don't have current values, assume they're valid
    if (isApiKeyConfigured && !currentApiKey) {
      // API key is configured, check other fields
      if (isAccountIdConfigured && !currentAccountId) {
        // Both are configured, configuration is valid
        return { isValid: true };
      } else if (!isAccountIdConfigured && currentAccountId) {
        // API key configured, validate new account ID
        return validateAccountIdDetailed(currentAccountId);
      } else if (!isAccountIdConfigured && !currentAccountId) {
        // API key configured, account ID needed
        return {
          isValid: false,
          message: 'Account ID is required',
        };
      }
    } else if (isAccountIdConfigured && !currentAccountId) {
      // Account ID is configured, validate API key
      if (currentApiKey) {
        return validateApiKeyDetailed(currentApiKey);
      } else {
        return {
          isValid: false,
          message: 'API key is required',
        };
      }
    }
    
    // Neither field is configured or we have new values to validate
    return validateConfiguration({
      apiKey: currentApiKey,
      accountId: currentAccountId,
      region: jsonData?.region,
    });
  }, [secureJsonFields, secureJsonData, jsonData]);

  return (
    <div>
      {/* Configuration validation alert */}
      {!configValidation.isValid && (
        <Alert title="Configuration Error" severity="error">
          {configValidation.message}
        </Alert>
      )}

      {/* API Key Field */}
      <InlineFieldRow>
        <InlineField
          label="API Key"
          labelWidth={14}
          tooltip="Your New Relic API key. This is stored securely and never sent to the frontend."
          required
          invalid={!!validationErrors.apiKey && !secureJsonFields?.apiKey}
          error={validationErrors.apiKey && !secureJsonFields?.apiKey ? validationErrors.apiKey : ''}
        >
          <SecretInput
            id="config-editor-api-key"
            data-testid="api-key-input"
            isConfigured={!!secureJsonFields?.apiKey}
            value={secureJsonData?.apiKey || ''}
            placeholder="Enter your New Relic API key"
            width={40}
            onReset={handleApiKeyReset}
            onChange={handleApiKeyChange}
            aria-label="New Relic API Key"
            aria-describedby="api-key-help"
            aria-invalid={!!validationErrors.apiKey && !secureJsonFields?.apiKey}
          />
        </InlineField>
      </InlineFieldRow>

      {/* API Key Help Text */}
      <div id="api-key-help" style={{ fontSize: '12px', color: '#6c757d', marginBottom: '16px' }}>
        You can find your API key in your New Relic account settings under "API keys".
      </div>

      {/* Account ID Field */}
      <InlineFieldRow>
        <InlineField
          label="Account ID"
          labelWidth={14}
          tooltip="Your New Relic account ID. This is stored securely and never sent to the frontend."
          required
          invalid={!!validationErrors.accountID && !secureJsonFields?.accountID}
          error={validationErrors.accountID && !secureJsonFields?.accountID ? validationErrors.accountID : ''}
        >
          <SecretInput
            id="config-editor-account-id"
            data-testid="account-id-input"
            isConfigured={!!secureJsonFields?.accountID}
            value={secureJsonData?.accountID || ''}
            placeholder="Enter your New Relic account ID"
            width={40}
            onReset={handleAccountIdReset}
            onChange={handleAccountIdChange}
            type="text"
            aria-label="New Relic Account ID"
            aria-describedby="account-id-help"
            aria-invalid={!!validationErrors.accountID && !secureJsonFields?.accountID}
          />
        </InlineField>
      </InlineFieldRow>

      {/* Account ID Help Text */}
      <div id="account-id-help" style={{ fontSize: '12px', color: '#6c757d', marginBottom: '16px' }}>
        Your account ID can be found in the New Relic URL when you're logged in (e.g., one.newrelic.com/accounts/YOUR_ACCOUNT_ID).
      </div>

      {/* Region Selection */}
      <InlineFieldRow>
        <InlineField
          label="Region"
          labelWidth={14}
          tooltip="Select the New Relic region for your account (US or EU)"
        >
          <Select
            id="config-editor-region"
            data-testid="region-select"
            options={regionOptions}
            value={regionOptions.find(option => option.value === jsonData?.region)}
            onChange={handleRegionChange}
            placeholder="Select region"
            width={40}
            aria-label="New Relic Region"
          />
        </InlineField>
      </InlineFieldRow>

      {/* Region Help Text */}
      <div style={{ fontSize: '12px', color: '#6c757d', marginBottom: '16px' }}>
        Choose US for accounts in the United States, or EU for accounts in Europe.
      </div>

      {/* Configuration Status */}
      {configValidation.isValid && (secureJsonFields?.apiKey || secureJsonData?.apiKey) && (secureJsonFields?.accountID || secureJsonData?.accountID) && (
        <Alert title="Configuration Complete" severity="success">
          Your New Relic data source is properly configured and ready to use.
        </Alert>
      )}
    </div>
  );
}
