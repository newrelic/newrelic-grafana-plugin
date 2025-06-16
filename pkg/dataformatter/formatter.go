package dataformatter

import (
	"encoding/json"
	"fmt"
	"time"

	"newrelic-grafana-plugin/pkg/constant"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
)

// IsCountQuery checks if the NRDB result container represents a single count query.
func IsCountQuery(results *nrdb.NRDBResultContainer) bool {
	return len(results.Results) == 1 && results.Results[0]["count"] != nil
}

// FormatQueryResults creates a unified Grafana DataFrame from New Relic NRDB query results
// that works for both count and regular queries, supporting both tabular and time series formats
func FormatQueryResults(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	// Print results as JSON for debugging
	resultsJSON, _ := json.MarshalIndent(results, "", "  ")
	log.DefaultLogger.Debug("Result count: %d\nResults:\n%s",
		len(results.Results), string(resultsJSON))

	if len(results.Results) == 0 {
		return resp
	}

	// Route to appropriate formatter based on query type
	if isSimpleCountQuery(results) {
		return formatSimpleCountQuery(results, query)
	} else if isFacetedCountQuery(results) {
		return formatFacetedCountQuery(results, query)
	} else {
		return formatStandardQuery(results, query)
	}
}

// isSimpleCountQuery checks if the results represent a simple count query
func isSimpleCountQuery(results *nrdb.NRDBResultContainer) bool {
	return len(results.Results) == 1 &&
		results.Results[0][constant.CountFieldName] != nil &&
		results.Results[0][constant.FacetFieldName] == nil
}

// isFacetedCountQuery checks if the results represent a faceted count query
func isFacetedCountQuery(results *nrdb.NRDBResultContainer) bool {
	return len(results.Results) > 0 &&
		results.Results[0][constant.CountFieldName] != nil &&
		results.Results[0][constant.FacetFieldName] != nil
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

// formatFacetedCountQuery formats results from a faceted count query
func formatFacetedCountQuery(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	// Get facet names
	facetNames := extractFacetNames(results)

	// Extract data
	counts, facetFields := extractFacetedData(results, facetNames)

	// Create table frame
	facetFrame := createFacetTableFrame(facetNames, counts, facetFields)

	// Create time series frame
	timeSeriesFrame := createFacetTimeSeriesFrame(facetNames, counts, facetFields, query)

	// Add both frames to the response
	resp.Frames = append(resp.Frames, facetFrame, timeSeriesFrame)
	return resp
}

// extractFacetNames extracts facet names from query metadata
func extractFacetNames(results *nrdb.NRDBResultContainer) []string {
	facetNames := []string{}
	if results.Metadata.Facets != nil {
		facetNames = results.Metadata.Facets
	}
	return facetNames
}

// extractFacetedData extracts counts and facet values from results
func extractFacetedData(results *nrdb.NRDBResultContainer, facetNames []string) ([]float64, map[string][]string) {
	counts := make([]float64, len(results.Results))
	facetFields := make(map[string][]string)

	for _, facetName := range facetNames {
		facetFields[facetName] = make([]string, len(results.Results))
	}

	for i, result := range results.Results {
		// Get count
		if countValue, ok := result[constant.CountFieldName].(float64); ok {
			counts[i] = countValue
		}

		// Get facet values
		extractFacetValues(result, facetNames, facetFields, i)
	}

	return counts, facetFields
}

// extractFacetValues extracts facet values from a single result
func extractFacetValues(result map[string]interface{}, facetNames []string, facetFields map[string][]string, index int) {
	if facetArray, ok := result[constant.FacetFieldName].([]interface{}); ok {
		for j, facetValue := range facetArray {
			if j < len(facetNames) {
				if strVal, ok := facetValue.(string); ok {
					facetFields[facetNames[j]][index] = strVal
				} else {
					facetFields[facetNames[j]][index] = fmt.Sprintf("%v", facetValue)
				}
			}
		}
	} else if result["facet"] != nil && len(facetNames) > 0 {
		// Handle single facet value case
		facetFields[facetNames[0]][index] = fmt.Sprintf("%v", result[constant.FacetFieldName])
	}
}

// createFacetTableFrame creates a table frame for faceted count queries
func createFacetTableFrame(facetNames []string, counts []float64, facetFields map[string][]string) *data.Frame {
	facetFrame := data.NewFrame(constant.FacetedFrameName)

	// Add facet fields
	for _, facetName := range facetNames {
		facetFrame.Fields = append(facetFrame.Fields,
			data.NewField(facetName, nil, facetFields[facetName]))
	}

	// Add count field
	facetFrame.Fields = append(facetFrame.Fields, data.NewField(constant.CountFieldName, nil, counts))

	// Set visualization preference
	facetFrame.Meta = &data.FrameMeta{
		PreferredVisualization: data.VisTypeTable,
	}

	return facetFrame
}

// createFacetTimeSeriesFrame creates a time series frame for faceted count queries
func createFacetTimeSeriesFrame(facetNames []string, counts []float64, facetFields map[string][]string, query backend.DataQuery) *data.Frame {
	timeSeriesFrame := data.NewFrame(constant.FacetedTimeSeriesFrameName)

	// Create time points
	timePoints := make([]time.Time, len(counts))
	for i := range timePoints {
		timePoints[i] = query.TimeRange.From
	}

	// Add time field
	timeSeriesFrame.Fields = append(timeSeriesFrame.Fields,
		data.NewField(constant.TimeFieldName, nil, timePoints))

	// Add facet fields
	for _, facetName := range facetNames {
		timeSeriesFrame.Fields = append(timeSeriesFrame.Fields,
			data.NewField(facetName, nil, facetFields[facetName]))
	}

	// Add count field
	timeSeriesFrame.Fields = append(timeSeriesFrame.Fields,
		data.NewField(constant.CountFieldName, nil, counts))

	// Set visualization preference
	timeSeriesFrame.Meta = &data.FrameMeta{
		PreferredVisualization: data.VisTypeGraph,
	}

	return timeSeriesFrame
}

// formatStandardQuery formats standard query results (time series or other data)
func formatStandardQuery(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}
	frame := data.NewFrame(constant.StandardResponseFrameName)

	// Extract field names
	fieldNames := extractFieldNames(results)

	// Add time field
	times := createTimeField(results, query)
	frame.Fields = append(frame.Fields, data.NewField(constant.TimeFieldName, nil, times))

	// Add data fields
	addDataFields(frame, results, fieldNames)

	resp.Frames = append(resp.Frames, frame)
	return resp
}

// extractFieldNames extracts unique field names from results
func extractFieldNames(results *nrdb.NRDBResultContainer) []string {
	fieldNamesMap := make(map[string]struct{})
	for _, result := range results.Results {
		for key := range result {
			if key != constant.TimestampFieldName {
				fieldNamesMap[key] = struct{}{}
			}
		}
	}

	var fieldNames []string
	for key := range fieldNamesMap {
		fieldNames = append(fieldNames, key)
	}

	return fieldNames
}

// createTimeField creates a time field from result timestamps
func createTimeField(results *nrdb.NRDBResultContainer, query backend.DataQuery) []time.Time {
	times := make([]time.Time, len(results.Results))
	for i, result := range results.Results {
		if ts, ok := result[constant.TimestampFieldName].(float64); ok {
			times[i] = time.Unix(int64(ts/1000), 0)
		} else {
			times[i] = query.TimeRange.From
		}
	}
	return times
}

// addDataFields adds data fields to the frame based on their types
func addDataFields(frame *data.Frame, results *nrdb.NRDBResultContainer, fieldNames []string) {
	for _, fieldName := range fieldNames {
		if len(results.Results) > 0 {
			switch results.Results[0][fieldName].(type) {
			case float64:
				values := make([]float64, len(results.Results))
				for i, result := range results.Results {
					if val, ok := result[fieldName].(float64); ok {
						values[i] = val
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))
			default:
				// Convert to string for other types
				values := make([]string, len(results.Results))
				for i, result := range results.Results {
					if result[fieldName] != nil {
						values[i] = fmt.Sprintf("%v", result[fieldName])
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))
			}
		}
	}
}
