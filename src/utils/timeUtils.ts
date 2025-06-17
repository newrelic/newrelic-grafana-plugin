/**
 * Time utilities for handling Grafana time picker integration with NRQL queries
 */

import { TimeRange } from '@grafana/data';

/**
 * Converts Grafana time range to NRQL SINCE/UNTIL clauses
 */
export interface NRQLTimeClause {
  since?: string;
  until?: string;
}

/**
 * Converts Grafana TimeRange to NRQL time clauses
 * @param timeRange - Grafana time range object
 * @returns NRQL time clauses
 */
export function convertTimeRangeToNRQL(timeRange: TimeRange): NRQLTimeClause {
  const from = timeRange.from;
  const to = timeRange.to;
  
  // If we have a relative time range (like "now-1h"), use it directly
  if (from.toString().startsWith('now-')) {
    const relativeTime = from.toString().replace('now-', '');
    return {
      since: convertGrafanaTimeToNRQL(relativeTime),
    };
  }
  
  // For absolute time ranges, calculate the duration
  const duration = to.valueOf() - from.valueOf();
  const durationInSeconds = Math.floor(duration / 1000);
  
  return {
    since: formatDurationForNRQL(durationInSeconds),
  };
}

/**
 * Converts Grafana time format to NRQL format
 * @param grafanaTime - Grafana time string (e.g., "1h", "30m", "7d")
 * @returns NRQL time string
 */
export function convertGrafanaTimeToNRQL(grafanaTime: string): string {
  // Handle common Grafana time formats
  const timeMap: Record<string, string> = {
    '5m': '5 minutes',
    '15m': '15 minutes',
    '30m': '30 minutes',
    '1h': '1 hour',
    '3h': '3 hours',
    '6h': '6 hours',
    '12h': '12 hours',
    '24h': '24 hours',
    '1d': '1 day',
    '7d': '7 days',
    '30d': '30 days',
  };
  
  return timeMap[grafanaTime] || grafanaTime;
}

/**
 * Formats duration in seconds to NRQL time format
 * @param seconds - Duration in seconds
 * @returns NRQL time string
 */
export function formatDurationForNRQL(seconds: number): string {
  if (seconds < 3600) {
    const minutes = Math.floor(seconds / 60);
    return `${minutes} minutes`;
  } else if (seconds < 86400) {
    const hours = Math.floor(seconds / 3600);
    return `${hours} hours`;
  } else {
    const days = Math.floor(seconds / 86400);
    return `${days} days`;
  }
}

/**
 * Template variables for NRQL queries that integrate with Grafana time picker
 */
export const GRAFANA_TIME_VARIABLES = {
  /** Use Grafana's time range automatically */
  TIME_FILTER: '$__timeFilter()',
  /** Start time as Unix timestamp */
  FROM_TIMESTAMP: '$__from',
  /** End time as Unix timestamp */
  TO_TIMESTAMP: '$__to',
  /** Start time as ISO string */
  FROM_ISO: '$__fromISOString',
  /** End time as ISO string */
  TO_ISO: '$__toISOString',
  /** Time range interval */
  INTERVAL: '$__interval',
} as const;

/**
 * Builds NRQL query with Grafana time picker integration
 * @param baseQuery - Base NRQL query without time clauses
 * @param useGrafanaTime - Whether to use Grafana time picker
 * @param customTimeClause - Custom time clause if not using Grafana time
 * @returns Complete NRQL query with time integration
 */
export function buildNRQLWithTimeIntegration(
  baseQuery: string,
  useGrafanaTime: boolean = true,
  customTimeClause?: string
): string {
  if (!baseQuery.trim()) {
    return baseQuery;
  }

  if (useGrafanaTime) {
    // Check if query already has Grafana time variables
    if (hasGrafanaTimeVariables(baseQuery)) {
      return baseQuery; // Already has Grafana time variables
    }

    // Remove existing SINCE/UNTIL clauses when adding Grafana time
    let query = baseQuery.replace(/\s*SINCE\s+[^)]*ago\s*/gi, ' ');
    query = query.replace(/\s*UNTIL\s+[^)]*ago\s*/gi, ' ');
    query = query.replace(/\s+/g, ' ').trim(); // Clean up multiple spaces

    // Add Grafana time variables to WHERE clause
    const hasWhere = /\bWHERE\b/i.test(query);
    
    if (hasWhere) {
      // Add to existing WHERE clause
      query = query.replace(
        /(\bWHERE\b\s+)/i,
        '$1timestamp >= $__from AND timestamp <= $__to AND '
      );
    } else {
      // Add new WHERE clause
      query = `${query} WHERE timestamp >= $__from AND timestamp <= $__to`;
    }
    
    return query;
  } else if (customTimeClause) {
    return `${baseQuery} ${customTimeClause}`;
  }
  
  return baseQuery;
}

/**
 * Checks if a query already has Grafana time variables
 * @param query - NRQL query string
 * @returns True if query contains Grafana time variables
 */
export function hasGrafanaTimeVariables(query: string): boolean {
  return /\$__(?:from|to|timeFilter|interval)/i.test(query);
} 