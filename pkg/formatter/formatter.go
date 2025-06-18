// Package formatter handles the conversion of New Relic API responses
// into Grafana data frames. It supports different query types including
// count queries, faceted queries, and standard time series data.
package formatter

import (
	"encoding/json"
	"fmt"
	"time"

	"newrelic-grafana-plugin/pkg/utils"

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
	} else if isFacetedTimeseriesQuery(results) {
		return formatFacetedTimeseriesQuery(results, query)
	} else if isFacetedCountQuery(results) {
		return formatFacetedCountQuery(results, query)
	} else {
		return formatStandardQuery(results, query)
	}
}

// isSimpleCountQuery checks if the results represent a simple count query
func isSimpleCountQuery(results *nrdb.NRDBResultContainer) bool {
	return len(results.Results) == 1 &&
		results.Results[0][utils.CountFieldName] != nil &&
		results.Results[0][utils.FacetFieldName] == nil
}

// isFacetedCountQuery checks if the results represent a faceted count query
func isFacetedCountQuery(results *nrdb.NRDBResultContainer) bool {
	return len(results.Results) > 0 &&
		results.Results[0][utils.CountFieldName] != nil &&
		results.Results[0][utils.FacetFieldName] != nil &&
		!hasTimeseriesData(results)
}

// isFacetedTimeseriesQuery checks if the results represent a faceted timeseries query
func isFacetedTimeseriesQuery(results *nrdb.NRDBResultContainer) bool {
	return len(results.Results) > 0 &&
		results.Results[0][utils.CountFieldName] != nil &&
		results.Results[0][utils.FacetFieldName] != nil &&
		hasTimeseriesData(results)
}

// hasTimeseriesData checks if the results contain timeseries data (beginTimeSeconds or endTimeSeconds)
func hasTimeseriesData(results *nrdb.NRDBResultContainer) bool {
	if len(results.Results) == 0 {
		return false
	}
	_, hasBeginTime := results.Results[0]["beginTimeSeconds"]
	_, hasEndTime := results.Results[0]["endTimeSeconds"]
	return hasBeginTime || hasEndTime
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
	graphFrame := data.NewFrame(utils.CountTimeSeriesFrameName)

	// Add time points using current time (since we removed time range processing)
	now := time.Now()
	timePoints := []time.Time{now.Add(-time.Hour), now}
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
	if countValue, ok := result[utils.CountFieldName].(float64); ok {
		count = countValue
	}
	return count
}

// formatFacetedCountQuery formats results from a faceted count query
func formatFacetedCountQuery(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	// Get facet names
	facetNames := extractFacetNames(results)
	log.DefaultLogger.Debug("Facet names extracted: %v", facetNames)

	// Extract data
	counts, facetFields := extractFacetedData(results, facetNames)
	log.DefaultLogger.Debug("Counts: %v, Facet fields: %v", counts, facetFields)

	// Create separate frames for each facet value (like Grafana Cloud plugin)
	if len(facetNames) > 0 {
		facetName := facetNames[0] // Use the first facet for labels
		facetValues := facetFields[facetName]

		for i, facetValue := range facetValues {
			// Create a frame for each facet value
			frame := data.NewFrame("")

			// Add time field
			now := time.Now()
			frame.Fields = append(frame.Fields,
				data.NewField("time", nil, []time.Time{now}))

			// Add count field with facet label (matching Grafana Cloud plugin)
			countField := data.NewField("count", map[string]string{
				facetName: facetValue,
			}, []float64{counts[i]})
			frame.Fields = append(frame.Fields, countField)

			resp.Frames = append(resp.Frames, frame)
		}
	}

	log.DefaultLogger.Debug("Total frames in response: %d", len(resp.Frames))

	return resp
}

// createPieChartFrame creates a frame optimized for pie chart visualization
func createPieChartFrame(facetNames []string, counts []float64, facetFields map[string][]string) *data.Frame {
	// Use a distinctive name for the pie chart frame
	pieFrame := data.NewFrame("pie_chart_data")

	if len(facetNames) > 0 && len(counts) > 0 {
		// For pie charts, we want facet values as labels
		facetName := facetNames[0] // Use the first facet for pie chart labels
		labels := facetFields[facetName]

		log.DefaultLogger.Debug("Creating pie chart frame: facet=%s, labels=%v, counts=%v", facetName, labels, counts)

		// Create fields for pie chart - labels first, then values
		pieFrame.Fields = append(pieFrame.Fields,
			data.NewField("label", nil, labels))
		pieFrame.Fields = append(pieFrame.Fields,
			data.NewField("value", nil, counts))
	}

	// Set visualization preference to table (pie chart VisType not available in SDK)
	// But use a custom type version to hint at pie chart usage
	pieFrame.Meta = &data.FrameMeta{
		PreferredVisualization: data.VisTypeTable,
		Custom: map[string]interface{}{
			"chartType": "pie",
		},
	}

	return pieFrame
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
		if countValue, ok := result[utils.CountFieldName].(float64); ok {
			counts[i] = countValue
		}

		// Get facet values
		extractFacetValues(result, facetNames, facetFields, i)
	}

	return counts, facetFields
}

// extractFacetValues extracts facet values from a single result
func extractFacetValues(result map[string]interface{}, facetNames []string, facetFields map[string][]string, index int) {
	if facetArray, ok := result[utils.FacetFieldName].([]interface{}); ok {
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
		facetFields[facetNames[0]][index] = fmt.Sprintf("%v", result[utils.FacetFieldName])
	}
}

// createFacetTableFrame creates a table frame for faceted count queries
func createFacetTableFrame(facetNames []string, counts []float64, facetFields map[string][]string) *data.Frame {
	facetFrame := data.NewFrame(utils.FacetedFrameName)

	// Add facet fields
	for _, facetName := range facetNames {
		facetFrame.Fields = append(facetFrame.Fields,
			data.NewField(facetName, nil, facetFields[facetName]))
	}

	// Add count field
	facetFrame.Fields = append(facetFrame.Fields, data.NewField(utils.CountFieldName, nil, counts))

	// Set visualization preference
	facetFrame.Meta = &data.FrameMeta{
		PreferredVisualization: data.VisTypeTable,
	}

	return facetFrame
}

// createFacetTimeSeriesFrame creates a time series frame for faceted count queries
func createFacetTimeSeriesFrame(facetNames []string, counts []float64, facetFields map[string][]string, query backend.DataQuery) *data.Frame {
	timeSeriesFrame := data.NewFrame(utils.FacetedTimeSeriesFrameName)

	// Create time points using current time
	now := time.Now()
	timePoints := make([]time.Time, len(counts))
	for i := range timePoints {
		timePoints[i] = now
	}

	// Add time field
	timeSeriesFrame.Fields = append(timeSeriesFrame.Fields,
		data.NewField(utils.TimeFieldName, nil, timePoints))

	// Add facet fields
	for _, facetName := range facetNames {
		timeSeriesFrame.Fields = append(timeSeriesFrame.Fields,
			data.NewField(facetName, nil, facetFields[facetName]))
	}

	// Add count field
	timeSeriesFrame.Fields = append(timeSeriesFrame.Fields,
		data.NewField(utils.CountFieldName, nil, counts))

	// Set visualization preference
	timeSeriesFrame.Meta = &data.FrameMeta{
		PreferredVisualization: data.VisTypeGraph,
	}

	return timeSeriesFrame
}

// formatStandardQuery formats standard query results (time series or other data)
func formatStandardQuery(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}
	frame := data.NewFrame(utils.StandardResponseFrameName)

	// Extract field names
	fieldNames := extractFieldNames(results)

	// Add time field
	times := createTimeField(results, query)
	frame.Fields = append(frame.Fields, data.NewField(utils.TimeFieldName, nil, times))

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
			// Exclude timestamp fields and New Relic TIMESERIES fields from data fields
			if key != utils.TimestampFieldName && key != "beginTimeSeconds" && key != "endTimeSeconds" {
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
	now := time.Now()

	for i, result := range results.Results {
		// First check for standard timestamp field
		if ts, ok := result[utils.TimestampFieldName].(float64); ok {
			times[i] = time.Unix(int64(ts/1000), 0)
		} else if beginTs, ok := result["beginTimeSeconds"].(float64); ok {
			// Handle New Relic TIMESERIES data which uses beginTimeSeconds
			times[i] = time.Unix(int64(beginTs), 0)
		} else {
			// Fallback to current time instead of query time range
			times[i] = now
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

// formatFacetedTimeseriesQuery formats results from a faceted timeseries query
func formatFacetedTimeseriesQuery(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	// Get facet names
	facetNames := extractFacetNames(results)
	log.DefaultLogger.Debug("Faceted timeseries - Facet names extracted: %v", facetNames)

	if len(facetNames) == 0 {
		// No facets, fall back to standard query
		return formatStandardQuery(results, query)
	}

	// Group results by facet value
	facetData := groupTimeseriesByFacet(results, facetNames[0])
	log.DefaultLogger.Debug("Faceted timeseries - Grouped into %d facet groups", len(facetData))

	// Create separate frames for each facet value (like Grafana Cloud plugin)
	for facetValue, facetResults := range facetData {
		frame := data.NewFrame("")

		// Create time field from the facet results
		times := createTimeField(&nrdb.NRDBResultContainer{Results: facetResults}, query)
		frame.Fields = append(frame.Fields, data.NewField("time", nil, times))

		// Create count field with facet label
		counts := make([]float64, len(facetResults))
		for i, result := range facetResults {
			if countValue, ok := result[utils.CountFieldName].(float64); ok {
				counts[i] = countValue
			}
		}

		countField := data.NewField("count", map[string]string{
			facetNames[0]: facetValue,
		}, counts)
		frame.Fields = append(frame.Fields, countField)

		resp.Frames = append(resp.Frames, frame)
	}

	log.DefaultLogger.Debug("Faceted timeseries - Total frames in response: %d", len(resp.Frames))

	return resp
}

// groupTimeseriesByFacet groups timeseries results by facet value
func groupTimeseriesByFacet(results *nrdb.NRDBResultContainer, facetName string) map[string][]nrdb.NRDBResult {
	grouped := make(map[string][]nrdb.NRDBResult)

	for _, result := range results.Results {
		facetValue := ""
		if facetArray, ok := result[utils.FacetFieldName].([]interface{}); ok && len(facetArray) > 0 {
			facetValue = fmt.Sprintf("%v", facetArray[0])
		} else if result[utils.FacetFieldName] != nil {
			facetValue = fmt.Sprintf("%v", result[utils.FacetFieldName])
		}

		if facetValue != "" {
			grouped[facetValue] = append(grouped[facetValue], result)
		}
	}

	return grouped
}
