# Grafana Time Picker Integration with NRQL

This document explains how the New Relic Grafana plugin integrates with Grafana's time picker to automatically update NRQL queries when the dashboard time range changes.

## Overview

The time picker integration allows your NRQL queries to automatically use the time range selected in Grafana's dashboard time picker. When users change the time range, your queries will automatically update without manual intervention.

## How It Works

### 1. Frontend Integration

In the query editor, users can toggle "Use Grafana Time Picker" to enable automatic time range integration:

```typescript
// QueryEditor component includes a toggle switch
<Switch
  value={useGrafanaTime}
  onChange={(e) => handleTimeIntegrationToggle(e.currentTarget.checked)}
  data-testid="grafana-time-toggle"
/>
```

### 2. Template Variables

When time picker integration is enabled, the plugin uses Grafana's built-in template variables:

- `$__from` - Start time as Unix timestamp in milliseconds
- `$__to` - End time as Unix timestamp in milliseconds  
- `$__timeFilter()` - Complete time filter clause
- `$__interval` - Appropriate interval based on time range
- `$__fromISOString` - Start time as ISO string
- `$__toISOString` - End time as ISO string

### 3. Backend Processing

The backend processes these template variables and converts them to NRQL-compatible time conditions:

```go
// ProcessNRQLWithTimeVariables replaces Grafana variables
func ProcessNRQLWithTimeVariables(query string, timeRange backend.TimeRange) string {
    fromMs, toMs := ConvertGrafanaTimeToNRQL(timeRange)
    
    replacements := map[string]string{
        "$__from": fromMs,
        "$__to":   toMs,
        // ... other variables
    }
    
    // Apply replacements and return processed query
}
```

## Usage Examples

### Basic Query with Time Integration

**Manual time query:**
```sql
SELECT count(*) FROM Transaction SINCE 1 hour ago
```

**With Grafana time picker:**
```sql
SELECT count(*) FROM Transaction WHERE timestamp >= $__from AND timestamp <= $__to
```

### Time Series Query

**Manual:**
```sql
SELECT count(*) FROM Transaction TIMESERIES 5 minutes SINCE 1 hour ago
```

**With time picker:**
```sql
SELECT count(*) FROM Transaction TIMESERIES $__interval WHERE timestamp >= $__from AND timestamp <= $__to
```

### Complex Query with Conditions

**Manual:**
```sql
SELECT average(duration) FROM Transaction 
WHERE appName = 'MyApp' AND duration > 1 
SINCE 6 hours ago 
FACET host
```

**With time picker:**
```sql
SELECT average(duration) FROM Transaction 
WHERE appName = 'MyApp' AND duration > 1 AND timestamp >= $__from AND timestamp <= $__to
FACET host
```

## Query Builder Integration

The visual query builder automatically adapts based on the time picker setting:

- **Time picker enabled**: Manual time range controls are hidden, shows "Using Grafana Dashboard Time Picker" indicator
- **Time picker disabled**: Shows manual SINCE/UNTIL time range selectors

## Benefits

1. **Automatic Updates**: Queries automatically reflect dashboard time range changes
2. **Consistent Time Ranges**: All panels use the same time range automatically
3. **User Experience**: Users can change time ranges without editing individual queries
4. **Dashboard Sharing**: Time ranges are preserved when sharing dashboards

## Configuration

### Enabling Time Picker Integration

1. In the query editor, toggle "Use Grafana Time Picker" on
2. The query will automatically be updated with Grafana template variables
3. Manual time range controls will be hidden in the query builder

### Disabling Time Picker Integration

1. Toggle "Use Grafana Time Picker" off
2. The query will revert to manual time clauses (default: "SINCE 1 hour ago")
3. Manual time range controls will be shown in the query builder

## Best Practices

### 1. Use Appropriate Time Fields

Ensure your NRQL queries use the correct timestamp field for your data type:

```sql
-- For most event data
WHERE timestamp >= $__from AND timestamp <= $__to

-- For custom timestamp fields
WHERE myCustomTimestamp >= $__from AND myCustomTimestamp <= $__to
```

### 2. Combine with Other Conditions

Time conditions work well with other WHERE clauses:

```sql
SELECT count(*) FROM Transaction 
WHERE appName = 'MyApp' 
  AND responseCode = 200 
  AND timestamp >= $__from 
  AND timestamp <= $__to
FACET region
```

### 3. Use Appropriate Intervals

The `$__interval` variable automatically adjusts based on the time range:

- 1 hour range: AUTO (New Relic decides)
- 6 hour range: 5m intervals
- 24 hour range: 15m intervals
- 7 day range: 1h intervals
- Longer ranges: 1d intervals

## Troubleshooting

### Query Not Updating with Time Changes

1. Verify "Use Grafana Time Picker" is enabled
2. Check that the query contains Grafana template variables (`$__from`, `$__to`)
3. Ensure the backend is processing template variables correctly

### Time Range Not Applied Correctly

1. Verify timestamp field name matches your data
2. Check that time values are in the correct format (Unix milliseconds)
3. Review backend logs for time variable processing

### Manual Time Clauses Not Working

1. Ensure "Use Grafana Time Picker" is disabled
2. Check NRQL syntax for SINCE/UNTIL clauses
3. Verify time formats match NRQL expectations ("1 hour ago", "30 minutes ago")

## Advanced Usage

### Custom Time Handling

You can create custom time handling logic by modifying the time utilities:

```typescript
// Custom time integration
const customTimeQuery = buildNRQLWithTimeIntegration(
  baseQuery,
  useGrafanaTime,
  'SINCE 2 hours ago UNTIL 1 hour ago'
);
```

### Mixing Template Variables

Combine time variables with other Grafana template variables:

```sql
SELECT count(*) FROM Transaction 
WHERE appName = '$app' 
  AND region = '$region'
  AND timestamp >= $__from 
  AND timestamp <= $__to
FACET host
```

## API Reference

### Frontend Functions

- `hasGrafanaTimeVariables(query: string): boolean` - Check if query has time variables
- `buildNRQLWithTimeIntegration(baseQuery: string, useGrafanaTime: boolean): string` - Build query with time integration
- `convertGrafanaTimeToNRQL(grafanaTime: string): string` - Convert time formats

### Backend Functions

- `ProcessNRQLWithTimeVariables(query string, timeRange backend.TimeRange): string` - Process template variables
- `HasGrafanaTimeVariables(query string): bool` - Check for template variables
- `ConvertGrafanaTimeToNRQL(timeRange backend.TimeRange): (string, string)` - Convert time range 