import {
  validateApiKey,
  validateAccountId,
  validateNrqlQuery,
  validateUrl,
  sanitizeInput,
  validateConfiguration,
  validateApiKeyDetailed,
  validateAccountIdDetailed,
} from '../validation';

describe('Validation Utils', () => {
  describe('validateNrqlQuery', () => {
    it('should accept all valid NRQL patterns and let New Relic API handle syntax validation', () => {
      // Basic SELECT queries
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT * FROM Span').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT average(duration) FROM Transaction').isValid).toBe(true);
      
      // Queries with WHERE clause
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction WHERE appName = "MyApp"').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction WHERE duration > 100').isValid).toBe(true);
      
      // Queries with FACET clause
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction FACET appName').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction FACET appName, host').isValid).toBe(true);
      
      // Queries with time clauses
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction SINCE 1 hour ago').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction SINCE 1 hour ago UNTIL 30 minutes ago').isValid).toBe(true);
      
      // Queries with TIMESERIES
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction TIMESERIES').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction TIMESERIES AUTO').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction TIMESERIES 5 minutes').isValid).toBe(true);
      
      // Queries with LIMIT
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction LIMIT 100').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction LIMIT 1000').isValid).toBe(true);
      
      // Complex queries
      const complexQuery = 'SELECT average(duration) FROM Transaction WHERE appName = "MyApp" FACET host SINCE 1 hour ago TIMESERIES AUTO LIMIT 100';
      expect(validateNrqlQuery(complexQuery).isValid).toBe(true);
      
      // Case insensitive
      expect(validateNrqlQuery('select count(*) from Transaction').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT COUNT(*) FROM TRANSACTION').isValid).toBe(true);
      expect(validateNrqlQuery('Select Average(duration) From Transaction').isValid).toBe(true);
      
      // Aggregation functions
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT average(duration) FROM Transaction').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT sum(duration) FROM Transaction').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT min(duration) FROM Transaction').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT max(duration) FROM Transaction').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT percentile(duration, 95) FROM Transaction').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT uniqueCount(userId) FROM Transaction').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT latest(timestamp) FROM Transaction').isValid).toBe(true);
      
      // Multiple attributes
      expect(validateNrqlQuery('SELECT duration, responseTime FROM Transaction').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT appName, host, duration FROM Transaction').isValid).toBe(true);
      
      // Special characters
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction WHERE appName = "My-App_2023"').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT count(*) FROM Transaction WHERE host LIKE "%prod%"').isValid).toBe(true);
    });

    it('should now accept queries that would previously be considered invalid - let New Relic API validate', () => {
      // These would previously fail but now pass - New Relic API will handle validation
      expect(validateNrqlQuery('INVALID QUERY').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT FROM').isValid).toBe(true);
      expect(validateNrqlQuery('FROM Transaction').isValid).toBe(true);
      expect(validateNrqlQuery('count(*) FROM Transaction').isValid).toBe(true);
      expect(validateNrqlQuery('FROM Transaction SINCE 1 hour ago').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT count(*)').isValid).toBe(true);
      expect(validateNrqlQuery('SELECT * WHERE appName = "test"').isValid).toBe(true);
    });

    it('should reject empty queries', () => {
      expect(validateNrqlQuery('').isValid).toBe(false);
      expect(validateNrqlQuery('').message).toBe('Query is required');
    });

    it('should reject queries with only whitespace', () => {
      expect(validateNrqlQuery('   ').isValid).toBe(false);
      expect(validateNrqlQuery('   ').message).toBe('Query cannot be empty');
      expect(validateNrqlQuery('\t\n').isValid).toBe(false);
      expect(validateNrqlQuery('\t\n').message).toBe('Query cannot be empty');
    });

    it('should handle null and undefined values', () => {
      expect(validateNrqlQuery(null as any).isValid).toBe(false);
      expect(validateNrqlQuery(null as any).message).toBe('Query is required');
      expect(validateNrqlQuery(undefined as any).isValid).toBe(false);
      expect(validateNrqlQuery(undefined as any).message).toBe('Query is required');
    });

    it('should reject dangerous database operations for security', () => {
      const dangerousQueries = [
        'DROP TABLE Transaction',
        'DELETE FROM Transaction',
        'INSERT INTO Transaction',
        'UPDATE Transaction SET',
        'CREATE TABLE Test',
        'ALTER TABLE Transaction',
        'drop table users',
        'delete from logs',
        'insert into data',
        'update records set',
        'create index on',
        'alter column name',
      ];

      dangerousQueries.forEach(query => {
        const result = validateNrqlQuery(query);
        expect(result.isValid).toBe(false);
        expect(result.message).toBe('Query contains potentially dangerous operations');
      });
    });
  });

  describe('validateApiKey (boolean)', () => {
    it('should validate valid API keys', () => {
      expect(validateApiKey('NRAK-ABC123DEF456')).toBe(true);
      expect(validateApiKey('NRAK-1234567890ABCDEF')).toBe(true);
    });

    it('should validate keys with different lengths', () => {
      expect(validateApiKey('NRAK-' + 'A'.repeat(30))).toBe(true);
      expect(validateApiKey('NRAK-' + 'B'.repeat(50))).toBe(true);
    });

    it('should reject invalid API key formats', () => {
      expect(validateApiKey('')).toBe(false);
      expect(validateApiKey('invalid-key')).toBe(false);
      expect(validateApiKey('ABC123DEF456')).toBe(false);
      expect(validateApiKey('NRAK-')).toBe(false);
    });

    it('should reject keys without NRAK prefix', () => {
      expect(validateApiKey('KEY-ABC123DEF456')).toBe(false);
      expect(validateApiKey('API-ABC123DEF456')).toBe(false);
      expect(validateApiKey('ABC123DEF456')).toBe(false);
    });

    it('should handle whitespace', () => {
      expect(validateApiKey('   ')).toBe(false);
      expect(validateApiKey('\t\n')).toBe(false);
      expect(validateApiKey('  NRAK-ABC123DEF456  ')).toBe(true); // Trims spaces
    });

    it('should be case sensitive for prefix', () => {
      expect(validateApiKey('nrak-ABC123DEF456')).toBe(false);
      expect(validateApiKey('Nrak-ABC123DEF456')).toBe(false);
      expect(validateApiKey('NRAK-abc123def456')).toBe(true); // Only prefix case sensitive
    });

    it('should reject keys that are too short', () => {
      expect(validateApiKey('NRAK-A')).toBe(false);
      expect(validateApiKey('NRAK-AB')).toBe(false);
      expect(validateApiKey('NRAK-ABC')).toBe(false);
    });

    it('should accept alphanumeric characters after prefix', () => {
      expect(validateApiKey('NRAK-ABC123DEF456GHI789')).toBe(true);
      expect(validateApiKey('NRAK-0123456789ABCDEF')).toBe(true);
    });

    it('should reject special characters in key part', () => {
      expect(validateApiKey('NRAK-ABC123-DEF456')).toBe(false);
      expect(validateApiKey('NRAK-ABC123_DEF456')).toBe(false);
      expect(validateApiKey('NRAK-ABC123@DEF456')).toBe(false);
    });
  });

  describe('validateApiKeyDetailed', () => {
    it('should validate correct API key format', () => {
      const validKey = 'NRAK-1234567890ABCDEF1234567890ABCDEF123';
      const result = validateApiKeyDetailed(validKey);
      expect(result.isValid).toBe(true);
    });

    it('should validate another correct API key format', () => {
      const validKey = 'NRAK-1234567890ABCDEF1234567890ABCDEF123';
      const result = validateApiKeyDetailed(validKey);
      expect(result.isValid).toBe(true);
    });

    it('should reject empty API key', () => {
      const result = validateApiKeyDetailed('');
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('API key is required');
    });

    it('should reject API key without NRAK- prefix', () => {
      const invalidKey = '1234567890abcdef1234567890abcdef1234';
      const result = validateApiKeyDetailed(invalidKey);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('New Relic API key must start with "NRAK-"');
    });

    it('should reject API key that is too short', () => {
      const shortKey = 'NRAK-123';
      const result = validateApiKeyDetailed(shortKey);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('API key is too short. It should be at least 10 characters long.');
    });

    it('should reject API key with special characters', () => {
      const invalidKey = 'NRAK-1234567890abcdef1234567890abcdef123!';
      const result = validateApiKeyDetailed(invalidKey);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('API key contains invalid characters. Only alphanumeric characters are allowed after "NRAK-".');
    });

    it('should reject API key with hyphens in the key part', () => {
      const invalidKey = 'NRAK-1234567890-abcdef1234567890abcdef123';
      const result = validateApiKeyDetailed(invalidKey);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('API key contains invalid characters. Only alphanumeric characters are allowed after "NRAK-".');
    });

    it('should reject API key that is too long', () => {
      const longKey = 'NRAK-' + 'A'.repeat(50); // 55 characters total
      const result = validateApiKeyDetailed(longKey);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('API key length appears invalid. New Relic API keys are typically 30-50 characters long.');
    });

    it('should reject API key that is too short overall', () => {
      const shortKey = 'NRAK-' + 'A'.repeat(20); // 25 characters total
      const result = validateApiKeyDetailed(shortKey);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('API key length appears invalid. New Relic API keys are typically 30-50 characters long.');
    });

    it('should handle null/undefined input', () => {
      const result = validateApiKeyDetailed(null as any);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('API key is required');
    });
  });

  describe('validateAccountId (boolean)', () => {
    it('should validate valid account IDs', () => {
      expect(validateAccountId(123456)).toBe(true);
      expect(validateAccountId(1234567890)).toBe(true);
      expect(validateAccountId(999999999999)).toBe(true);
    });

    it('should validate account ID strings', () => {
      expect(validateAccountId('123456')).toBe(true);
      expect(validateAccountId('1234567890')).toBe(true);
      expect(validateAccountId('999999999999')).toBe(true);
    });

    it('should reject invalid account IDs', () => {
      expect(validateAccountId(0)).toBe(false);
      expect(validateAccountId(-123)).toBe(false);
      expect(validateAccountId(12.34)).toBe(false);
    });

    it('should reject non-numeric strings', () => {
      expect(validateAccountId('abc')).toBe(false);
      expect(validateAccountId('123abc')).toBe(false);
      expect(validateAccountId('12.34')).toBe(false);
      expect(validateAccountId('')).toBe(false);
    });

    it('should handle null and undefined values', () => {
      expect(validateAccountId(null as any)).toBe(false);
      expect(validateAccountId(undefined as any)).toBe(false);
    });

    it('should reject account IDs that are too small', () => {
      expect(validateAccountId(1)).toBe(false);
      expect(validateAccountId(12)).toBe(false);
      expect(validateAccountId(123)).toBe(false);
      expect(validateAccountId(1234)).toBe(false);
      expect(validateAccountId(12345)).toBe(false);
    });

    it('should accept account IDs with minimum length', () => {
      expect(validateAccountId(123456)).toBe(true);
      expect(validateAccountId(100000)).toBe(true);
    });

    it('should handle very large account IDs', () => {
      expect(validateAccountId(999999999999999)).toBe(true);
      expect(validateAccountId('999999999999999')).toBe(true);
    });

    it('should reject infinity and NaN', () => {
      expect(validateAccountId(Infinity)).toBe(false);
      expect(validateAccountId(-Infinity)).toBe(false);
      expect(validateAccountId(NaN)).toBe(false);
    });

    it('should handle edge cases with string conversion', () => {
      expect(validateAccountId('0123456')).toBe(true); // Leading zeros should be ok
      expect(validateAccountId('123456.0')).toBe(false); // Decimal points not allowed
      expect(validateAccountId('1e6')).toBe(true); // Scientific notation converts to number
    });
  });

  describe('validateAccountIdDetailed', () => {
    it('should validate correct account ID', () => {
      const result = validateAccountIdDetailed('1234567');
      expect(result.isValid).toBe(true);
    });

    it('should validate numeric account ID', () => {
      const result = validateAccountIdDetailed(1234567);
      expect(result.isValid).toBe(true);
    });

    it('should reject empty account ID', () => {
      const result = validateAccountIdDetailed('');
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Account ID is required');
    });

    it('should reject non-numeric account ID', () => {
      const result = validateAccountIdDetailed('abc123');
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Account ID must be a positive number');
    });

    it('should reject account ID that is too short', () => {
      const result = validateAccountIdDetailed('123');
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Account ID must be between 6 and 10 digits');
    });

    it('should reject account ID that is too long', () => {
      const result = validateAccountIdDetailed('12345678901');
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Account ID must be between 6 and 10 digits');
    });

    it('should reject negative account ID', () => {
      const result = validateAccountIdDetailed(-123456);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Account ID must be a positive number');
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
        apiKey: 'NRAK-1234567890ABCDEF1234567890ABCDEF123',
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
      expect(result.message).toBe('New Relic API key must start with "NRAK-"');
    });

    it('should reject invalid account ID', () => {
      const config = {
        apiKey: 'NRAK-1234567890ABCDEF1234567890ABCDEF123',
        accountId: 'invalid',
        region: 'US',
      };
      const result = validateConfiguration(config);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Account ID must be a positive number');
    });

    it('should reject invalid region', () => {
      const config = {
        apiKey: 'NRAK-1234567890ABCDEF1234567890ABCDEF123',
        accountId: '1234567',
        region: 'INVALID',
      };
      const result = validateConfiguration(config);
      expect(result.isValid).toBe(false);
      expect(result.message).toBe('Region must be either US or EU');
    });

    it('should handle missing region', () => {
      const config = {
        apiKey: 'NRAK-1234567890ABCDEF1234567890ABCDEF123',
        accountId: '1234567',
      };
      const result = validateConfiguration(config);
      expect(result.isValid).toBe(true);
    });
  });

  describe('edge cases and error handling', () => {
    it('should handle all validation functions with various data types', () => {
      const invalidInputs = [
        null,
        undefined,
        {},
        [],
        true,
        false,
        function() {},
      ];

      invalidInputs.forEach(input => {
        expect(validateNrqlQuery(input as any).isValid).toBe(false);
        expect(validateApiKey(input as any)).toBe(false);
        expect(validateAccountId(input as any)).toBe(false);
      });
    });

    it('should be resilient to prototype pollution attempts', () => {
      const maliciousInput = '{"__proto__": {"isValid": true}}';
      // NRQL validation now accepts any non-empty string - let New Relic API validate
      expect(validateNrqlQuery(maliciousInput).isValid).toBe(true);
      expect(validateApiKey(maliciousInput)).toBe(false);
      expect(validateAccountId(maliciousInput)).toBe(false);
    });
  });
}); 