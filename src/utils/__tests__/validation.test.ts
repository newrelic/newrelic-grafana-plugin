import {
  validateApiKey,
  validateAccountId,
  validateNrqlQuery,
  validateUrl,
  sanitizeInput,
  validateConfiguration,
} from '../validation';

describe('Validation Utils', () => {
  describe('validateApiKey', () => {
    it('should validate correct API key format', () => {
      const validKey = 'NRAK1234567890abcdef1234567890abcdef1234';
      const result = validateApiKey(validKey);
      expect(result.isValid).toBe(true);
    });

    it('should reject empty API key', () => {
      const result = validateApiKey('');
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('API key is required');
    });

    it('should reject API key with wrong length', () => {
      const shortKey = 'NRAK123';
      const result = validateApiKey(shortKey);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('API key must be 40 characters long and contain only alphanumeric characters');
    });

    it('should reject API key with special characters', () => {
      const invalidKey = 'NRAK1234567890abcdef1234567890abcdef123!';
      const result = validateApiKey(invalidKey);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('API key must be 40 characters long and contain only alphanumeric characters');
    });

    it('should handle null/undefined input', () => {
      const result = validateApiKey(null as any);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('API key is required');
    });
  });

  describe('validateAccountId', () => {
    it('should validate correct account ID', () => {
      const result = validateAccountId('1234567');
      expect(result.isValid).toBe(true);
    });

    it('should validate numeric account ID', () => {
      const result = validateAccountId(1234567);
      expect(result.isValid).toBe(true);
    });

    it('should reject empty account ID', () => {
      const result = validateAccountId('');
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Account ID is required');
    });

    it('should reject non-numeric account ID', () => {
      const result = validateAccountId('abc123');
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Account ID must be a positive number');
    });

    it('should reject account ID that is too short', () => {
      const result = validateAccountId('123');
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Account ID must be between 6 and 10 digits');
    });

    it('should reject account ID that is too long', () => {
      const result = validateAccountId('12345678901');
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Account ID must be between 6 and 10 digits');
    });

    it('should reject negative account ID', () => {
      const result = validateAccountId(-123456);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Account ID must be a positive number');
    });
  });

  describe('validateNrqlQuery', () => {
    it('should validate correct NRQL query', () => {
      const query = 'SELECT count(*) FROM Transaction SINCE 1 hour ago';
      const result = validateNrqlQuery(query);
      expect(result.isValid).toBe(true);
    });

    it('should validate SELECT * query', () => {
      const query = 'SELECT * FROM Transaction SINCE 1 hour ago LIMIT 100';
      const result = validateNrqlQuery(query);
      expect(result.isValid).toBe(true);
    });

    it('should reject empty query', () => {
      const result = validateNrqlQuery('');
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Query is required');
    });

    it('should reject query without SELECT', () => {
      const query = 'FROM Transaction SINCE 1 hour ago';
      const result = validateNrqlQuery(query);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('NRQL query must start with SELECT');
    });

    it('should reject query without FROM', () => {
      const query = 'SELECT count(*) SINCE 1 hour ago';
      const result = validateNrqlQuery(query);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('NRQL query must contain a FROM clause');
    });

    it('should reject dangerous operations', () => {
      const dangerousQueries = [
        'DROP TABLE Transaction',
        'DELETE FROM Transaction',
        'INSERT INTO Transaction',
        'UPDATE Transaction SET',
        'CREATE TABLE Test',
        'ALTER TABLE Transaction',
      ];

      dangerousQueries.forEach(query => {
        const result = validateNrqlQuery(query);
        expect(result.isValid).toBe(false);
        expect(result.message).toBe('Query contains potentially dangerous operations');
      });
    });

    it('should handle null/undefined input', () => {
      const result = validateNrqlQuery(null as any);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Query is required');
    });

    it('should handle whitespace-only query', () => {
      const result = validateNrqlQuery('   ');
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Query cannot be empty');
    });
  });

  describe('validateUrl', () => {
    it('should validate correct HTTPS URL', () => {
      const url = 'https://api.newrelic.com/graphql';
      const result = validateUrl(url);
      expect(result.isValid).toBe(true);
    });

    it('should reject HTTP URL', () => {
      const url = 'http://api.newrelic.com/graphql';
      const result = validateUrl(url);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('URL must use HTTPS protocol');
    });

    it('should reject invalid URL format', () => {
      const url = 'not-a-url';
      const result = validateUrl(url);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Invalid URL format');
    });

    it('should reject empty URL', () => {
      const result = validateUrl('');
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('URL is required');
    });
  });

  describe('sanitizeInput', () => {
    it('should remove HTML tags', () => {
      const input = '<script>alert("xss")</script>Hello';
      const result = sanitizeInput(input);
      expect(result).toBe('scriptalert("xss")/scriptHello');
    });

    it('should remove javascript: protocol', () => {
      const input = 'javascript:alert("xss")';
      const result = sanitizeInput(input);
      expect(result).toBe('alert("xss")');
    });

    it('should remove event handlers', () => {
      const input = 'onclick=alert("xss") Hello';
      const result = sanitizeInput(input);
      expect(result).toBe('Hello');
    });

    it('should handle empty input', () => {
      const result = sanitizeInput('');
      expect(result).toBe('');
    });

    it('should handle null/undefined input', () => {
      expect(sanitizeInput(null as any)).toBe('');
      expect(sanitizeInput(undefined as any)).toBe('');
    });

    it('should trim whitespace', () => {
      const input = '  Hello World  ';
      const result = sanitizeInput(input);
      expect(result).toBe('Hello World');
    });
  });

  describe('validateConfiguration', () => {
    it('should validate complete configuration', () => {
      const config = {
        apiKey: 'NRAK1234567890abcdef1234567890abcdef1234',
        accountId: '1234567',
        region: 'US',
      };
      const result = validateConfiguration(config);
      expect(result.isValid).toBe(true);
    });

    it('should reject invalid API key', () => {
      const config = {
        apiKey: 'invalid',
        accountId: '1234567',
        region: 'US',
      };
      const result = validateConfiguration(config);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('API key must be 40 characters long and contain only alphanumeric characters');
    });

    it('should reject invalid account ID', () => {
      const config = {
        apiKey: 'NRAK1234567890abcdef1234567890abcdef1234',
        accountId: 'invalid',
        region: 'US',
      };
      const result = validateConfiguration(config);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Account ID must be a positive number');
    });

    it('should reject invalid region', () => {
      const config = {
        apiKey: 'NRAK1234567890abcdef1234567890abcdef1234',
        accountId: '1234567',
        region: 'INVALID',
      };
      const result = validateConfiguration(config);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Region must be either US or EU');
    });

    it('should handle missing region', () => {
      const config = {
        apiKey: 'NRAK1234567890abcdef1234567890abcdef1234',
        accountId: '1234567',
      };
      const result = validateConfiguration(config);
      expect(result.isValid).toBe(true);
    });
  });
}); 