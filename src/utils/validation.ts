import { ValidationResult } from '../types';

/**
 * Simple boolean validation for API keys (used by some components)
 * @param apiKey - The API key to validate
 * @returns Boolean indicating if the API key is valid
 */
export function validateApiKey(apiKey: string): boolean {
  if (typeof apiKey === 'undefined' || apiKey === null) {
    return false;
  }
  
  if (typeof apiKey !== 'string') {
    return false;
  }

  const trimmed = apiKey.trim();
  
  // Basic validation - must start with NRAK- and have content after
  if (!trimmed.startsWith('NRAK-') || trimmed.length < 10) {
    return false;
  }

  // Allow alphanumeric characters after NRAK-
  const keyPart = trimmed.substring(5); // Remove NRAK- prefix
  const alphanumericRegex = /^[A-Za-z0-9]+$/;
  
  return alphanumericRegex.test(keyPart);
}

/**
 * Detailed API key validation with error messages
 * @param apiKey - The API key to validate
 * @returns Validation result with success status and optional error message
 */
export function validateApiKeyDetailed(apiKey: string): ValidationResult {
  if (!apiKey || typeof apiKey !== 'string') {
    return {
      isValid: false,
      message: 'API key is required',
    };
  }

  // New Relic API keys are typically 40 characters long and contain only alphanumeric characters
  const apiKeyRegex = /^[A-Za-z0-9]{40}$/;
  
  if (!apiKeyRegex.test(apiKey.trim())) {
    return {
      isValid: false,
      message: 'API key must be 40 characters long and contain only alphanumeric characters',
    };
  }

  return { isValid: true };
}

/**
 * Simple boolean validation for account IDs
 * @param accountId - The account ID to validate
 * @returns Boolean indicating if the account ID is valid
 */
export function validateAccountId(accountId: string | number): boolean {
  if (typeof accountId === 'undefined' || accountId === null) {
    return false;
  }

  // Check for Infinity and NaN directly
  if (typeof accountId === 'number') {
    if (!isFinite(accountId) || isNaN(accountId)) {
      return false;
    }
  }

  // If it's a string, check for decimal points first
  if (typeof accountId === 'string') {
    // Reject strings with decimal points, but allow scientific notation
    if (accountId.includes('.') && !accountId.toLowerCase().includes('e')) {
      return false;
    }
  }

  const numericAccountId = typeof accountId === 'string' ? Number(accountId) : accountId;

  // Additional check after conversion
  if (!isFinite(numericAccountId) || isNaN(numericAccountId) || numericAccountId <= 0) {
    return false;
  }

  // New Relic account IDs are typically at least 6 digits
  return numericAccountId >= 100000;
}

/**
 * Detailed account ID validation with error messages
 * @param accountId - The account ID to validate
 * @returns Validation result with success status and optional error message
 */
export function validateAccountIdDetailed(accountId: string | number): ValidationResult {
  if (!accountId) {
    return {
      isValid: false,
      message: 'Account ID is required',
    };
  }

  const numericAccountId = typeof accountId === 'string' ? Number(accountId) : accountId;

  if (isNaN(numericAccountId) || numericAccountId <= 0) {
    return {
      isValid: false,
      message: 'Account ID must be a positive number',
    };
  }

  // New Relic account IDs are typically 6-10 digits
  if (numericAccountId < 100000 || numericAccountId > 9999999999) {
    return {
      isValid: false,
      message: 'Account ID must be between 6 and 10 digits',
    };
  }

  return { isValid: true };
}

/**
 * Validates an NRQL query string
 * @param query - The NRQL query to validate
 * @returns Validation result with success status and optional error message
 */
export function validateNrqlQuery(query: string): ValidationResult {
  if (!query || typeof query !== 'string') {
    return {
      isValid: false,
      message: 'Query is required',
    };
  }

  const trimmedQuery = query.trim();

  if (trimmedQuery.length === 0) {
    return {
      isValid: false,
      message: 'Query cannot be empty',
    };
  }

  // Check for potentially dangerous operations first (basic security)
  const dangerousPatterns = [
    /DROP\s+/i,
    /DELETE\s+/i,
    /INSERT\s+/i,
    /UPDATE\s+/i,
    /CREATE\s+/i,
    /ALTER\s+/i,
  ];

  for (const pattern of dangerousPatterns) {
    if (pattern.test(trimmedQuery)) {
      return {
        isValid: false,
        message: 'Query contains potentially dangerous operations',
      };
    }
  }

  // Basic NRQL validation - must start with SELECT
  if (!trimmedQuery.toUpperCase().startsWith('SELECT')) {
    return {
      isValid: false,
      message: 'NRQL query must start with SELECT',
    };
  }

  // Must contain FROM clause
  if (!trimmedQuery.toUpperCase().includes(' FROM ')) {
    return {
      isValid: false,
      message: 'NRQL query must contain a FROM clause',
    };
  }

  return { isValid: true };
}

/**
 * Validates a URL format
 * @param url - The URL to validate
 * @returns Validation result with success status and optional error message
 */
export function validateUrl(url: string): ValidationResult {
  if (!url || typeof url !== 'string') {
    return {
      isValid: false,
      message: 'URL is required',
    };
  }

  try {
    const urlObj = new URL(url);
    
    // Must be HTTPS for security
    if (urlObj.protocol !== 'https:') {
      return {
        isValid: false,
        message: 'URL must use HTTPS protocol',
      };
    }

    return { isValid: true };
  } catch {
    return {
      isValid: false,
      message: 'Invalid URL format',
    };
  }
}

/**
 * Sanitizes user input to prevent XSS attacks
 * @param input - The input string to sanitize
 * @returns Sanitized string
 */
export function sanitizeInput(input: string): string {
  if (!input || typeof input !== 'string') {
    return '';
  }

  return input
    .replace(/[<>]/g, '') // Remove potential HTML tags
    .replace(/javascript:/gi, '') // Remove javascript: protocol
    .replace(/\bon\w*\s*=\s*[^>\s]*/gi, '') // Remove event handlers like onclick=
    .trim();
}

/**
 * Validates configuration completeness
 * @param config - Configuration object to validate
 * @returns Validation result with success status and optional error message
 */
export function validateConfiguration(config: {
  apiKey?: string;
  accountId?: string | number;
  region?: string;
}): ValidationResult {
  const apiKeyValidation = validateApiKeyDetailed(config.apiKey || '');
  if (!apiKeyValidation.isValid) {
    return apiKeyValidation;
  }

  const accountIdValidation = validateAccountIdDetailed(config.accountId || '');
  if (!accountIdValidation.isValid) {
    return accountIdValidation;
  }

  if (config.region && !['US', 'EU'].includes(config.region)) {
    return {
      isValid: false,
      message: 'Region must be either US or EU',
    };
  }

  return { isValid: true };
} 