// Package utils provides utility functions for the New Relic Grafana plugin
package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// ConvertGrafanaTimeToNRQL converts Grafana time range to NRQL time clauses
func ConvertGrafanaTimeToNRQL(timeRange backend.TimeRange) (string, string) {
	from := timeRange.From
	to := timeRange.To

	// Calculate duration in milliseconds for NRQL
	fromMs := from.UnixMilli()
	toMs := to.UnixMilli()

	return fmt.Sprintf("%d", fromMs), fmt.Sprintf("%d", toMs)
}

// ProcessNRQLWithTimeVariables replaces Grafana template variables in NRQL queries
func ProcessNRQLWithTimeVariables(query string, timeRange backend.TimeRange) string {
	fromMs, toMs := ConvertGrafanaTimeToNRQL(timeRange)

	// Replace Grafana template variables
	replacements := map[string]string{
		"$__from":          fromMs,
		"$__to":            toMs,
		"$__fromISOString": timeRange.From.Format(time.RFC3339),
		"$__toISOString":   timeRange.To.Format(time.RFC3339),
	}

	result := query
	for variable, value := range replacements {
		result = strings.ReplaceAll(result, variable, value)
	}

	// Handle $__timeFilter() - convert to proper NRQL timestamp conditions
	timeFilterRegex := regexp.MustCompile(`\$__timeFilter\(\)`)
	if timeFilterRegex.MatchString(result) {
		timeFilter := fmt.Sprintf("timestamp >= %s AND timestamp <= %s", fromMs, toMs)
		result = timeFilterRegex.ReplaceAllString(result, timeFilter)
	}

	// Handle $__interval variable (convert to appropriate NRQL interval)
	intervalRegex := regexp.MustCompile(`\$__interval`)
	if intervalRegex.MatchString(result) {
		interval := calculateNRQLInterval(timeRange)
		result = intervalRegex.ReplaceAllString(result, interval)
	}

	return result
}

// calculateNRQLInterval calculates an appropriate TIMESERIES interval based on time range
func calculateNRQLInterval(timeRange backend.TimeRange) string {
	duration := timeRange.To.Sub(timeRange.From)

	switch {
	case duration <= time.Hour:
		return "AUTO" // Let New Relic decide for short ranges
	case duration <= 6*time.Hour:
		return "5m"
	case duration <= 24*time.Hour:
		return "15m"
	case duration <= 7*24*time.Hour:
		return "1h"
	default:
		return "1d"
	}
}

// ConvertRelativeTimeToNRQL converts relative time strings to NRQL format
func ConvertRelativeTimeToNRQL(relativeTime string) string {
	// Common Grafana relative time formats to NRQL
	timeMap := map[string]string{
		"now-5m":  "5 minutes ago",
		"now-15m": "15 minutes ago",
		"now-30m": "30 minutes ago",
		"now-1h":  "1 hour ago",
		"now-3h":  "3 hours ago",
		"now-6h":  "6 hours ago",
		"now-12h": "12 hours ago",
		"now-24h": "24 hours ago",
		"now-1d":  "1 day ago",
		"now-7d":  "7 days ago",
		"now-30d": "30 days ago",
	}

	if nrqlTime, exists := timeMap[relativeTime]; exists {
		return nrqlTime
	}

	// Try to parse custom relative time formats
	re := regexp.MustCompile(`now-(\d+)([mhd])`)
	matches := re.FindStringSubmatch(relativeTime)
	if len(matches) == 3 {
		value, err := strconv.Atoi(matches[1])
		if err == nil {
			unit := matches[2]
			switch unit {
			case "m":
				return fmt.Sprintf("%d minutes ago", value)
			case "h":
				return fmt.Sprintf("%d hours ago", value)
			case "d":
				return fmt.Sprintf("%d days ago", value)
			}
		}
	}

	return relativeTime
}

// HasGrafanaTimeVariables checks if a query contains Grafana time template variables
func HasGrafanaTimeVariables(query string) bool {
	patterns := []string{
		`\$__from`,
		`\$__to`,
		`\$__timeFilter\(\)`,
		`\$__interval`,
		`\$__fromISOString`,
		`\$__toISOString`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, query)
		if matched {
			return true
		}
	}

	return false
}

// BuildNRQLWithGrafanaTime builds NRQL query with Grafana time integration
func BuildNRQLWithGrafanaTime(baseQuery string, timeRange backend.TimeRange) string {
	// If query already has Grafana variables, process them
	if HasGrafanaTimeVariables(baseQuery) {
		return ProcessNRQLWithTimeVariables(baseQuery, timeRange)
	}

	// If query has manual SINCE/UNTIL clauses, replace them with Grafana time
	sinceRegex := regexp.MustCompile(`(?i)SINCE\s+[^)]*ago(?:\s+UNTIL\s+[^)]*ago)?`)
	if sinceRegex.MatchString(baseQuery) {
		fromMs, toMs := ConvertGrafanaTimeToNRQL(timeRange)
		timeCondition := fmt.Sprintf("WHERE timestamp >= %s AND timestamp <= %s", fromMs, toMs)

		// Replace SINCE clause with timestamp conditions
		result := sinceRegex.ReplaceAllString(baseQuery, "")

		// Add WHERE clause or append to existing WHERE
		whereRegex := regexp.MustCompile(`(?i)\bWHERE\b`)
		if whereRegex.MatchString(result) {
			result = whereRegex.ReplaceAllStringFunc(result, func(match string) string {
				return fmt.Sprintf("%s timestamp >= %s AND timestamp <= %s AND", match, fromMs, toMs)
			})
		} else {
			result = strings.TrimSpace(result) + " " + timeCondition
		}

		return result
	}

	// Add time conditions to query without time clauses
	fromMs, toMs := ConvertGrafanaTimeToNRQL(timeRange)
	return fmt.Sprintf("%s WHERE timestamp >= %s AND timestamp <= %s", baseQuery, fromMs, toMs)
}
