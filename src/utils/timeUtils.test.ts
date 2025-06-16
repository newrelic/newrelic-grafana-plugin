/**
 * Tests for time utility functions
 */

import { 
  buildNRQLWithTimeIntegration, 
  hasGrafanaTimeVariables, 
  convertGrafanaTimeToNRQL,
  GRAFANA_TIME_VARIABLES 
} from './timeUtils';
import { TimeRange, dateTime } from '@grafana/data';

describe('timeUtils', () => {
  const mockTimeRange: TimeRange = {
    from: dateTime('2024-01-01T10:00:00Z'),
    to: dateTime('2024-01-01T11:00:00Z'),
    raw: {
      from: 'now-1h',
      to: 'now'
    }
  };

  describe('hasGrafanaTimeVariables', () => {
    it('should detect Grafana time variables', () => {
      expect(hasGrafanaTimeVariables('SELECT count(*) FROM Transaction WHERE timestamp >= $__from')).toBe(true);
      expect(hasGrafanaTimeVariables('SELECT count(*) FROM Transaction WHERE $__timeFilter()')).toBe(true);
      expect(hasGrafanaTimeVariables('SELECT count(*) FROM Transaction SINCE 1 hour ago')).toBe(false);
    });
  });

  describe('buildNRQLWithTimeIntegration', () => {
    it('should add Grafana time variables to query without WHERE clause', () => {
      const baseQuery = 'SELECT count(*) FROM Transaction SINCE 1 hour ago';
      const result = buildNRQLWithTimeIntegration(baseQuery, true);
      
      expect(result).toBe('SELECT count(*) FROM Transaction WHERE timestamp >= $__from AND timestamp <= $__to');
    });

    it('should add Grafana time variables to existing WHERE clause', () => {
      const baseQuery = 'SELECT count(*) FROM Transaction WHERE appName = "test" SINCE 1 hour ago';
      const result = buildNRQLWithTimeIntegration(baseQuery, true);
      
      expect(result).toBe('SELECT count(*) FROM Transaction WHERE timestamp >= $__from AND timestamp <= $__to AND appName = "test"');
    });

    it('should not modify query that already has Grafana time variables', () => {
      const baseQuery = 'SELECT count(*) FROM Transaction WHERE timestamp >= $__from AND timestamp <= $__to';
      const result = buildNRQLWithTimeIntegration(baseQuery, true);
      
      expect(result).toBe('SELECT count(*) FROM Transaction WHERE timestamp >= $__from AND timestamp <= $__to');
    });

    it('should handle complex queries with FACET and LIMIT', () => {
      const baseQuery = 'SELECT average(duration) FROM Transaction WHERE appName = "test" FACET host SINCE 1 hour ago LIMIT 100';
      const result = buildNRQLWithTimeIntegration(baseQuery, true);
      
      expect(result).toBe('SELECT average(duration) FROM Transaction WHERE timestamp >= $__from AND timestamp <= $__to AND appName = "test" FACET host LIMIT 100');
    });

    it('should remove SINCE and UNTIL clauses when adding Grafana time', () => {
      const baseQuery = 'SELECT count(*) FROM Transaction SINCE 2 hours ago UNTIL 1 hour ago';
      const result = buildNRQLWithTimeIntegration(baseQuery, true);
      
      expect(result).toBe('SELECT count(*) FROM Transaction WHERE timestamp >= $__from AND timestamp <= $__to');
    });

    it('should return query unchanged when useGrafanaTime is false', () => {
      const baseQuery = 'SELECT count(*) FROM Transaction SINCE 1 hour ago';
      const result = buildNRQLWithTimeIntegration(baseQuery, false);
      
      expect(result).toBe('SELECT count(*) FROM Transaction SINCE 1 hour ago');
    });

    it('should handle empty query gracefully', () => {
      const result = buildNRQLWithTimeIntegration('', true);
      expect(result).toBe('');
    });
  });

  describe('convertGrafanaTimeToNRQL', () => {
    it('should convert relative time to NRQL format', () => {
      expect(convertGrafanaTimeToNRQL('1h')).toBe('1 hour');
      expect(convertGrafanaTimeToNRQL('30m')).toBe('30 minutes');
      expect(convertGrafanaTimeToNRQL('7d')).toBe('7 days');
    });

    it('should handle unknown formats gracefully', () => {
      expect(convertGrafanaTimeToNRQL('unknown')).toBe('unknown');
    });
  });

  describe('GRAFANA_TIME_VARIABLES', () => {
    it('should contain all expected variables', () => {
      expect(GRAFANA_TIME_VARIABLES.FROM_TIMESTAMP).toBe('$__from');
      expect(GRAFANA_TIME_VARIABLES.TO_TIMESTAMP).toBe('$__to');
      expect(GRAFANA_TIME_VARIABLES.TIME_FILTER).toBe('$__timeFilter()');
      expect(GRAFANA_TIME_VARIABLES.INTERVAL).toBe('$__interval');
    });
  });
}); 