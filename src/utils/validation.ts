import { ValidationResult } from '../types';

/**
 * Validates a New Relic API key format
 * @param apiKey - The API key to validate
 * @returns Validation result with success status and optional error message
 */
export function validateApiKey(apiKey: string): ValidationResult {
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
 * Validates a New Relic account ID
 * @param accountId - The account ID to validate
 * @returns Validation result with success status and optional error message
 */
export function validateAccountId(accountId: string | number): ValidationResult {
  if (!accountId) {
    return {
      isValid: false,
      message: 'Account ID is required',
    };
  }

  const numericAccountId = typeof accountId === 'string' ? parseInt(accountId, 10) : accountId;

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

  // Check for potentially dangerous operations (basic security)
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
    .replace(/on\w+=/gi, '') // Remove event handlers
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
  const apiKeyValidation = validateApiKey(config.apiKey || '');
  if (!apiKeyValidation.isValid) {
    return apiKeyValidation;
  }

  const accountIdValidation = validateAccountId(config.accountId || '');
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