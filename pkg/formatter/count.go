// Package formatter provides functionality for formatting New Relic query results
// into Grafana data frames.
package formatter

import (
	"time"

	"newrelic-grafana-plugin/pkg/constant"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
)

// IsCountQuery checks if the NRDB result container represents a single count query.
func IsCountQuery(results *nrdb.NRDBResultContainer) bool {
	return len(results.Results) == 1 && results.Results[0]["count"] != nil
}

// isSimpleCountQuery checks if the results represent a simple count query
func isSimpleCountQuery(results *nrdb.NRDBResultContainer) bool {
	return len(results.Results) == 1 &&
		results.Results[0][constant.CountFieldName] != nil &&
		results.Results[0][constant.FacetFieldName] == nil
}

// formatSimpleCountQuery formats results from a simple count query
func formatSimpleCountQuery(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	// Extract count value
	count := extractCountValue(results.Results[0])

	// Frame 1: For table/stat panels - just the count with no time
	valueFrame := data.NewFrame("count",
		data.NewField("count", nil, []float64{count}),
	)
	valueFrame.Meta = &data.FrameMeta{
		PreferredVisualization: data.VisTypeTable,
	}

	// Frame 2: For time series/graph panels - count with time points
	graphFrame := createCountTimeSeriesFrame(count, query)

	// Add both frames to the response
	resp.Frames = append(resp.Frames, valueFrame, graphFrame)
	return resp
}

// createCountTimeSeriesFrame creates a time series frame for count queries
func createCountTimeSeriesFrame(count float64, query backend.DataQuery) *data.Frame {
	graphFrame := data.NewFrame(constant.CountTimeSeriesFrameName)

	// Add time points spanning the query range
	timePoints := []time.Time{query.TimeRange.From, query.TimeRange.To}
	graphFrame.Fields = append(graphFrame.Fields,
		data.NewField("time", nil, timePoints))

	// Add corresponding count values
	countValues := []float64{count, count}
	graphFrame.Fields = append(graphFrame.Fields,
		data.NewField("count", nil, countValues))

	// Mark this frame explicitly as preferring graph visualization
	graphFrame.Meta = &data.FrameMeta{
		PreferredVisualization: data.VisTypeGraph,
	}

	return graphFrame
}

// extractCountValue safely extracts a count value from a result
func extractCountValue(result map[string]interface{}) float64 {
	count := float64(0)
	if countValue, ok := result[constant.CountFieldName].(float64); ok {
		count = countValue
	}
	return count
}
