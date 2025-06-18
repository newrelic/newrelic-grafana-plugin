// Package utils defines shared constants used throughout the New Relic Grafana plugin.
// These constants are used for field names in NRDB query results and Grafana DataFrames,
// as well as for naming different types of data frames.
package utils

const (
	// Field names used in NRDB query results and Grafana DataFrames
	CountFieldName     = "count"     // Field name for count values in query results
	FacetFieldName     = "facet"     // Field name for facet values in query results
	TimeFieldName      = "time"      // Field name for time values in time series data
	TimestampFieldName = "timestamp" // Field name for timestamp values in query results

	// Frame names used for Grafana DataFrames
	CountTimeSeriesFrameName   = "count_time_series" // Name for time series frames containing count data
	StandardResponseFrameName  = "response"          // Name for standard query response frames
	FacetedFrameName           = ""                  // Name for faceted table frames (empty by Grafana SDK default)
	FacetedTimeSeriesFrameName = "facet_time_series" // Name for time series frames containing faceted data
)
