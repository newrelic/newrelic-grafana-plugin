import React, { ChangeEvent, useState, useCallback } from 'react';
import { InlineField, InlineFieldRow, SecretInput, Select } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps, SelectableValue } from '@grafana/data';
import { NewRelicDataSourceOptions, NewRelicSecureJsonData, NEW_RELIC_REGIONS } from '../types';
import { validateApiKeyDetailed, validateAccountIdDetailed } from '../utils/validation';
import { logger } from '../utils/logger';

interface Props extends DataSourcePluginOptionsEditorProps<NewRelicDataSourceOptions, NewRelicSecureJsonData> {}

/**
 * Configuration editor component for the New Relic data source
 * Handles API key, account ID, and region configuration
 */
export function ConfigEditor({ onOptionsChange, options }: Props) {
  const { secureJsonFields, secureJsonData, jsonData } = options;
  const [validationErrors, setValidationErrors] = useState<Record<string, string>>({});
  const [hasInteracted, setHasInteracted] = useState<Record<string, boolean>>({});
  const [hasSaveAttempted, setHasSaveAttempted] = useState(false);

  // Region options for the select dropdown
  const regionOptions: Array<SelectableValue<string>> = [
    { label: 'United States (US)', value: NEW_RELIC_REGIONS.US },
    { label: 'Europe (EU)', value: NEW_RELIC_REGIONS.EU },
  ];

  /**
   * Validates and updates the API key (without showing errors while typing)
   */
  const handleApiKeyChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    const apiKey = event.target.value;
    
    // Update the options immediately but don't validate yet
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...secureJsonData,
        apiKey,
      },
    });
  }, [options, secureJsonData, onOptionsChange]);

  /**
   * Handles API key field blur for validation
   */
  const handleApiKeyBlur = useCallback(() => {
    setHasInteracted(prev => ({ ...prev, apiKey: true }));
    const apiKey = secureJsonData?.apiKey || '';
    const validation = validateApiKeyDetailed(apiKey);
    
    setValidationErrors(prev => ({
      ...prev,
      apiKey: validation.isValid ? '' : validation.message || 'Invalid API key',
    }));

    if (!validation.isValid) {
      logger.warn('API key validation failed', { error: validation.message });
    }
  }, [secureJsonData]);

  /**
   * Resets the API key field
   */
  const handleApiKeyReset = useCallback(() => {
    setValidationErrors(prev => ({ ...prev, apiKey: '' }));
    setHasInteracted(prev => ({ ...prev, apiKey: false }));
    
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
   * Validates and updates the account ID (without showing errors while typing)
   */
  const handleAccountIdChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    const accountId = event.target.value;
    
    // Update the options immediately but don't validate yet
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...secureJsonData,
        accountID: accountId,
      },
    });
  }, [options, secureJsonData, onOptionsChange]);

  /**
   * Handles account ID field blur for validation
   */
  const handleAccountIdBlur = useCallback(() => {
    setHasInteracted(prev => ({ ...prev, accountID: true }));
    const accountId = secureJsonData?.accountID || '';
    const validation = validateAccountIdDetailed(accountId.toString());
    
    setValidationErrors(prev => ({
      ...prev,
      accountID: validation.isValid ? '' : validation.message || 'Invalid account ID',
    }));

    if (!validation.isValid) {
      logger.warn('Account ID validation failed', { error: validation.message });
    }
  }, [secureJsonData]);

  /**
   * Resets the account ID field
   */
  const handleAccountIdReset = useCallback(() => {
    setValidationErrors(prev => ({ ...prev, accountID: '' }));
    setHasInteracted(prev => ({ ...prev, accountID: false }));
    
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

  /**
   * Validates all fields when save/test is attempted
   */
  const validateAllFields = useCallback(() => {
    setHasSaveAttempted(true);
    setHasInteracted({ apiKey: true, accountID: true });
    
    const apiKey = secureJsonData?.apiKey || '';
    const accountID = secureJsonData?.accountID || '';
    
    const apiKeyValidation = validateApiKeyDetailed(apiKey);
    const accountIdValidation = validateAccountIdDetailed(accountID.toString());
    
    setValidationErrors({
      apiKey: apiKeyValidation.isValid ? '' : apiKeyValidation.message || 'Invalid API key',
      accountID: accountIdValidation.isValid ? '' : accountIdValidation.message || 'Invalid account ID',
    });
    
    return apiKeyValidation.isValid && accountIdValidation.isValid;
  }, [secureJsonData]);

  // Attach validation to the form submission
  React.useEffect(() => {
    const form = document.querySelector('form');
    if (form) {
      const handleSubmit = (e: Event) => {
        const isValid = validateAllFields();
        if (!isValid) {
          e.preventDefault();
          logger.warn('Form submission prevented due to validation errors');
        }
      };

      form.addEventListener('submit', handleSubmit);
      return () => {
        form.removeEventListener('submit', handleSubmit);
      };
    }
  }, [validateAllFields]);

  // Also handle Save & test button clicks
  React.useEffect(() => {
    const saveButton = document.querySelector('button[type="submit"], button[form]');
    if (saveButton) {
      const handleClick = (e: Event) => {
        const isValid = validateAllFields();
        if (!isValid) {
          e.preventDefault();
          logger.warn('Save & test prevented due to validation errors');
        }
      };

      saveButton.addEventListener('click', handleClick);
      return () => {
        saveButton.removeEventListener('click', handleClick);
      };
    }
  }, [validateAllFields]);

  return (
    <div>
      {/* API Key Field */}
      <InlineFieldRow>
        <InlineField
          label="API Key"
          labelWidth={16}
          tooltip="Your New Relic API key. This is stored securely and never sent to the frontend."
          required
          invalid={!!validationErrors.apiKey && (hasInteracted.apiKey || hasSaveAttempted) && !secureJsonFields?.apiKey}
          error={validationErrors.apiKey && (hasInteracted.apiKey || hasSaveAttempted) && !secureJsonFields?.apiKey ? validationErrors.apiKey : ''}
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
            onBlur={handleApiKeyBlur}
            aria-label="New Relic API Key"
            aria-describedby="api-key-help"
            aria-invalid={!!validationErrors.apiKey && (hasInteracted.apiKey || hasSaveAttempted) && !secureJsonFields?.apiKey}
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
          labelWidth={16}
          tooltip="Your New Relic account ID. This is stored securely and never sent to the frontend."
          required
          invalid={!!validationErrors.accountID && (hasInteracted.accountID || hasSaveAttempted) && !secureJsonFields?.accountID}
          error={validationErrors.accountID && (hasInteracted.accountID || hasSaveAttempted) && !secureJsonFields?.accountID ? validationErrors.accountID : ''}
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
            onBlur={handleAccountIdBlur}
            type="text"
            aria-label="New Relic Account ID"
            aria-describedby="account-id-help"
            aria-invalid={!!validationErrors.accountID && (hasInteracted.accountID || hasSaveAttempted) && !secureJsonFields?.accountID}
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
          labelWidth={16}
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
    </div>
  );
}
