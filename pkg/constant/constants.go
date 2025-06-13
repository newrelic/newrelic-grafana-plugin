package constant

const (
	// Field names used in NRDB query results and Grafana DataFrames
	CountFieldName     = "count"
	FacetFieldName     = "facet"
	TimeFieldName      = "time"
	TimestampFieldName = "timestamp"

	// Frame names used for Grafana DataFrames
	CountTimeSeriesFrameName   = "count_time_series"
	StandardResponseFrameName  = "response"
	FacetedFrameName           = ""                  // Name for the faceted table frame (often left empty by Grafana SDK default)
	FacetedTimeSeriesFrameName = "facet_time_series" // Added for the explicit name "facet_time_series"
)
