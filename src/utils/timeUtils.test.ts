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
    it('should add Grafana time variables to query without time clauses', () => {
      const baseQuery = 'SELECT count(*) FROM Transaction';
      const result = buildNRQLWithTimeIntegration(baseQuery, true);
      
      expect(result).toContain('$__from');
      expect(result).toContain('$__to');
    });

    it('should replace existing SINCE clauses with Grafana variables', () => {
      const baseQuery = 'SELECT count(*) FROM Transaction SINCE 1 hour ago';
      const result = buildNRQLWithTimeIntegration(baseQuery, true);
      
      expect(result).toContain('$__from');
      expect(result).toContain('$__to');
      expect(result).not.toContain('SINCE 1 hour ago');
    });

    it('should return original query when not using Grafana time', () => {
      const baseQuery = 'SELECT count(*) FROM Transaction SINCE 1 hour ago';
      const result = buildNRQLWithTimeIntegration(baseQuery, false);
      
      expect(result).toBe(baseQuery);
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