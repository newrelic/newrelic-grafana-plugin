// Package formatter handles the conversion of New Relic API responses
// into Grafana data frames. It supports different query types including
// count queries, faceted queries, and standard time series data.
package formatter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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
	} else if isFacetedCountQuery(results) {
		return formatFacetedCountQuery(results, query)
	} else if isFacetedTimeseriesQuery(results) {
		// Handle faceted timeseries queries (e.g., "SELECT sum(duration) FROM Transaction facet request.uri TIMESERIES")
		return formatFacetedTimeseriesQuery(results, query)
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
	if len(results.Results) == 0 {
		return false
	}

	// Must have count field and facet field, but NOT timeseries data
	hasCount := results.Results[0][utils.CountFieldName] != nil
	hasFacet := results.Results[0][utils.FacetFieldName] != nil
	hasTimeseries := hasTimeseriesData(results)

	// Only treat as faceted count query if it has count, facet, but NO timeseries
	return hasCount && hasFacet && !hasTimeseries
}

// isFacetedTimeseriesQuery checks if the results represent a faceted timeseries query
func isFacetedTimeseriesQuery(results *nrdb.NRDBResultContainer) bool {
	if len(results.Results) == 0 {
		return false
	}

	// Check if this has facet data and timeseries data
	hasFacet := results.Results[0][utils.FacetFieldName] != nil
	hasTimeseries := hasTimeseriesData(results)

	// For faceted timeseries, we need both facets and timeseries data
	// The aggregation field can be anything (count, sum.duration, average.duration, etc.)
	return hasFacet && hasTimeseries
}

// Overloaded for NRDBResultContainerMultiResultCustomized
func isFacetedTimeseriesQueryMulti(results *nrdb.NRDBResultContainerMultiResultCustomized) bool {
	if len(results.Results) == 0 {
		return false
	}

	// Check if this has facet data and timeseries data
	hasFacet := results.Results[0][utils.FacetFieldName] != nil
	hasTimeseries := hasTimeseriesDataMulti(results)

	// For faceted timeseries, we need both facets and timeseries data
	// The aggregation field can be anything (count, sum.duration, average.duration, etc.)
	return hasFacet && hasTimeseries
}

func hasTimeseriesData(results *nrdb.NRDBResultContainer) bool {
	if len(results.Results) == 0 {
		return false
	}
	_, hasBeginTime := results.Results[0]["beginTimeSeconds"]
	_, hasEndTime := results.Results[0]["endTimeSeconds"]
	return hasBeginTime || hasEndTime
}

func hasTimeseriesDataMulti(results *nrdb.NRDBResultContainerMultiResultCustomized) bool {
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

// Helper for Multi type metadata
func getMetadataMulti(results *nrdb.NRDBResultContainerMultiResultCustomized) *nrdb.NRDBMetadata {
	return &results.Metadata
}

// Multi version for NRDBResultContainerMultiResultCustomized
func extractFacetNamesMulti(results *nrdb.NRDBResultContainerMultiResultCustomized) []string {
	metadata := getMetadataMulti(results)
	if metadata != nil && metadata.Facets != nil {
		return metadata.Facets
	}
	return nil
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
// Now enhanced to handle both regular and faceted aggregation fields
func formatStandardQuery(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	// Check if this is a faceted aggregation query (not count-based)
	facetNames := extractFacetNames(results)
	if len(facetNames) > 0 && !hasCountField(results) {
		// This is a faceted aggregation query like "SELECT sum(duration) FROM Transaction facet request.uri TIMESERIES"
		return formatFacetedAggregationQuery(results, query, facetNames)
	}

	// Standard single-frame response for non-faceted queries
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

// formatFacetedAggregationQuery handles faceted aggregation queries like Grafana Cloud
// Creates separate frames for each facet value with proper labels
func formatFacetedAggregationQuery(results *nrdb.NRDBResultContainer, query backend.DataQuery, facetNames []string) *backend.DataResponse {
	resp := &backend.DataResponse{}

	if len(facetNames) == 0 {
		return resp
	}

	facetName := facetNames[0] // Use the first facet for grouping

	// Group results by facet value
	facetData := groupResultsByFacet(results, facetName)
	log.DefaultLogger.Debug("Faceted aggregation - Grouped into %d facet groups", len(facetData))

	// Get all field names (excluding time and facet fields)
	allFieldNames := extractFieldNames(results)
	var aggregationFields []string
	for _, fieldName := range allFieldNames {
		if fieldName != utils.FacetFieldName {
			aggregationFields = append(aggregationFields, fieldName)
		}
	}

	// Create separate frames for each facet value
	for facetValue, facetResults := range facetData {
		frame := data.NewFrame("")
		times := createTimeField(&nrdb.NRDBResultContainer{Results: facetResults}, query)
		frame.Fields = append(frame.Fields, data.NewField("time", nil, times))

		// Add aggregation fields with facet labels
		for _, fieldName := range aggregationFields {
			// Create field with facet label (matching Grafana Cloud pattern)
			labels := map[string]string{
				facetName: facetValue,
			}

			// Extract values for this field
			values := make([]*float64, len(facetResults))
			for i, result := range facetResults {
				if result[fieldName] != nil && result[fieldName] != "" {
					if val, ok := result[fieldName].(float64); ok {
						values[i] = &val
					} else if strVal, ok := result[fieldName].(string); ok && strVal != "" {
						if parsed, err := parseNumericString(strVal); err == nil {
							values[i] = &parsed
						}
					} else if intVal, ok := result[fieldName].(int); ok {
						floatVal := float64(intVal)
						values[i] = &floatVal
					} else if int64Val, ok := result[fieldName].(int64); ok {
						floatVal := float64(int64Val)
						values[i] = &floatVal
					}
				}
			}

			// Create field with proper type info matching Grafana Cloud
			field := data.NewField(fieldName, labels, values)
			frame.Fields = append(frame.Fields, field)
		}

		resp.Frames = append(resp.Frames, frame)
	}

	log.DefaultLogger.Debug("Faceted aggregation - Total frames in response: %d", len(resp.Frames))
	return resp
}

// groupResultsByFacet groups results by facet value for aggregation queries
func groupResultsByFacet(results *nrdb.NRDBResultContainer, facetName string) map[string][]nrdb.NRDBResult {
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

// hasCountField checks if results contain count field
func hasCountField(results *nrdb.NRDBResultContainer) bool {
	if len(results.Results) == 0 {
		return false
	}
	_, hasCount := results.Results[0][utils.CountFieldName]
	return hasCount
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

// parseNumericString attempts to parse a string as a float64, handling scientific notation
func parseNumericString(s string) (float64, error) {
	// Handle scientific notation and regular floats
	return strconv.ParseFloat(s, 64)
}

// addDataFields adds data fields to the frame based on their types
func addDataFields(frame *data.Frame, results *nrdb.NRDBResultContainer, fieldNames []string) {
	for _, fieldName := range fieldNames {
		if len(results.Results) > 0 {
			// Use improved field type detection
			fieldType := detectFieldType(results.Results, fieldName)

			switch fieldType {
			case "number":
				// Use nullable float64 to handle nil/empty values properly
				values := make([]*float64, len(results.Results))
				for i, result := range results.Results {
					if result[fieldName] != nil && result[fieldName] != "" {
						if val, ok := result[fieldName].(float64); ok {
							values[i] = &val
						} else if strVal, ok := result[fieldName].(string); ok && strVal != "" {
							// Try to parse scientific notation strings
							if parsed, err := parseNumericString(strVal); err == nil {
								values[i] = &parsed
							}
						} else if intVal, ok := result[fieldName].(int); ok {
							floatVal := float64(intVal)
							values[i] = &floatVal
						} else if int64Val, ok := result[fieldName].(int64); ok {
							floatVal := float64(int64Val)
							values[i] = &floatVal
						}
					}
					// For nil/empty values, values[i] remains nil (which becomes null in JSON)
				}
				frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))

			case "timestamp":
				// Handle timestamp fields specially
				values := make([]*time.Time, len(results.Results))
				for i, result := range results.Results {
					if result[fieldName] != nil && result[fieldName] != "" {
						if timestampVal, ok := result[fieldName].(float64); ok {
							// Convert Unix timestamp to time.Time
							t := time.Unix(int64(timestampVal/1000), 0)
							values[i] = &t
						} else if timestampStr, ok := result[fieldName].(string); ok {
							// Try to parse timestamp string
							if parsed, err := strconv.ParseFloat(timestampStr, 64); err == nil {
								t := time.Unix(int64(parsed/1000), 0)
								values[i] = &t
							}
						}
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))

			case "array":
				// Handle arrays (like histogram, uniques) - convert to JSON string for display
				values := make([]string, len(results.Results))
				for i, result := range results.Results {
					if result[fieldName] != nil {
						if arrayVal, ok := result[fieldName].([]interface{}); ok {
							// Convert array to JSON string for better display
							if jsonBytes, err := json.Marshal(arrayVal); err == nil {
								values[i] = string(jsonBytes)
							} else {
								values[i] = fmt.Sprintf("%v", arrayVal)
							}
						} else {
							values[i] = fmt.Sprintf("%v", result[fieldName])
						}
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))

			case "object":
				// Handle objects (like percentile results) - convert to JSON string or extract values
				if strings.HasPrefix(fieldName, "percentile.") {
					// Special handling for percentile objects - try to extract individual percentile values
					handlePercentileField(frame, results, fieldName)
				} else {
					// General object handling - convert to JSON string
					values := make([]string, len(results.Results))
					for i, result := range results.Results {
						if result[fieldName] != nil {
							if objVal, ok := result[fieldName].(map[string]interface{}); ok {
								if jsonBytes, err := json.Marshal(objVal); err == nil {
									values[i] = string(jsonBytes)
								} else {
									values[i] = fmt.Sprintf("%v", objVal)
								}
							} else {
								values[i] = fmt.Sprintf("%v", result[fieldName])
							}
						}
					}
					frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))
				}

			case "boolean":
				// Handle boolean values
				values := make([]*bool, len(results.Results))
				for i, result := range results.Results {
					if result[fieldName] != nil {
						if boolVal, ok := result[fieldName].(bool); ok {
							values[i] = &boolVal
						}
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))

			default: // "string" and fallback
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

// handlePercentileField handles percentile objects by creating separate fields for each percentile
func handlePercentileField(frame *data.Frame, results *nrdb.NRDBResultContainer, fieldName string) {
	// Collect all percentile keys from all results
	percentileKeys := make(map[string]bool)
	for _, result := range results.Results {
		if result[fieldName] != nil {
			if objVal, ok := result[fieldName].(map[string]interface{}); ok {
				for key := range objVal {
					percentileKeys[key] = true
				}
			}
		}
	}

	// Create a field for each percentile
	for percentileKey := range percentileKeys {
		fieldNameWithPercentile := fmt.Sprintf("%s.%s", fieldName, percentileKey)
		values := make([]*float64, len(results.Results))

		for i, result := range results.Results {
			if result[fieldName] != nil {
				if objVal, ok := result[fieldName].(map[string]interface{}); ok {
					if percentileVal, exists := objVal[percentileKey]; exists {
						if floatVal, ok := percentileVal.(float64); ok {
							values[i] = &floatVal
						} else if strVal, ok := percentileVal.(string); ok {
							if parsed, err := parseNumericString(strVal); err == nil {
								values[i] = &parsed
							}
						}
					}
				}
			}
		}

		frame.Fields = append(frame.Fields, data.NewField(fieldNameWithPercentile, nil, values))
	}
}

// formatFacetedTimeseriesQuery formats results from a faceted timeseries query
func formatFacetedTimeseriesQuery(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	// Get facet names
	facetNames := extractFacetNames(results)
	log.DefaultLogger.Debug("Faceted timeseries - Facet names extracted: %v", facetNames)

	if len(facetNames) == 0 {
		// No facets, fall back to standard query
		return formatStandardQuery(results, query)
	}

	// Use the enhanced faceted aggregation formatter to handle any aggregation field
	// This handles count, sum.duration, average.duration, etc.
	return formatFacetedAggregationQuery(results, query, facetNames)
}

// Overloaded for NRDBResultContainerMultiResultCustomized
func formatFacetedTimeseriesQueryMulti(results *nrdb.NRDBResultContainerMultiResultCustomized, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	facetNames := extractFacetNamesMulti(results)
	log.DefaultLogger.Debug("Faceted timeseries - Facet names extracted: %v", facetNames)

	if len(facetNames) == 0 {
		return formatStandardQueryMulti(results, query)
	}

	facetData := groupTimeseriesByFacetMulti(results, facetNames[0])
	log.DefaultLogger.Debug("Faceted timeseries - Grouped into %d facet groups", len(facetData))

	for facetValue, facetResults := range facetData {
		frame := data.NewFrame("")
		times := createTimeField(&nrdb.NRDBResultContainer{Results: facetResults}, query)
		frame.Fields = append(frame.Fields, data.NewField("time", nil, times))

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

// Multi version for NRDBResultContainerMultiResultCustomized
func groupTimeseriesByFacetMulti(results *nrdb.NRDBResultContainerMultiResultCustomized, facetName string) map[string][]nrdb.NRDBResult {
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

// FormatFacetedTimeseriesResults returns a Grafana DataResponse for faceted timeseries queries
func FormatFacetedTimeseriesResults(results *nrdb.NRDBResultContainerMultiResultCustomized, query backend.DataQuery) *backend.DataResponse {

	resultsJSON, _ := json.MarshalIndent(results, "", "  ")
	log.DefaultLogger.Debug("FormatFacetedTimeseriesResults Result count: %d\nResults:\n%s",
		len(results.Results), string(resultsJSON))

	if !isFacetedTimeseriesQueryMulti(results) {
		resp := &backend.DataResponse{}
		resp.Error = fmt.Errorf("results are not a faceted timeseries query")
		return resp
	}

	// Convert to standard format and use the enhanced faceted aggregation formatter
	standardResults := &nrdb.NRDBResultContainer{
		Results:  make([]nrdb.NRDBResult, len(results.Results)),
		Metadata: results.Metadata,
	}

	// Copy results
	for i, result := range results.Results {
		standardResults.Results[i] = result
	}

	// Get facet names
	facetNames := extractFacetNames(standardResults)
	if len(facetNames) == 0 {
		// No facets found, fall back to standard query
		return formatStandardQuery(standardResults, query)
	}

	// Use the enhanced faceted aggregation formatter
	return formatFacetedAggregationQuery(standardResults, query, facetNames)
}

// isAggregationField checks if a field name represents an aggregation function result
func isAggregationField(fieldName string) bool {
	// Basic aggregation prefixes
	aggregationPrefixes := []string{
		"average.", "sum.", "min.", "max.", "count", "uniqueCount.", "latest.", "earliest.",
		"median.", "percentile.", "rate.", "apdex.", "histogram.", "uniques.",
		"getField.", "round.", "percentage.", "stddev.", "variance.",
	}

	// Check for exact matches (count is standalone)
	exactMatches := []string{
		"count",
	}

	// Check exact matches first
	for _, exact := range exactMatches {
		if fieldName == exact {
			return true
		}
	}

	// Check prefixes
	for _, prefix := range aggregationPrefixes {
		if strings.HasPrefix(fieldName, prefix) {
			return true
		}
	}

	return false
}

// detectFieldType analyzes a field across all results to determine the best data type
func detectFieldType(results []nrdb.NRDBResult, fieldName string) string {
	// Check if it's an aggregation field first
	if isAggregationField(fieldName) {
		// Special handling for specific aggregation types
		if strings.HasPrefix(fieldName, "histogram.") {
			return "array" // histogram returns array of numbers
		}
		if strings.HasPrefix(fieldName, "uniques.") {
			return "array" // uniques returns array of strings/values
		}
		if strings.HasPrefix(fieldName, "percentile.") {
			return "object" // percentile returns object with percentile values
		}
		if strings.HasPrefix(fieldName, "earliest.timestamp") || strings.HasPrefix(fieldName, "latest.timestamp") {
			return "timestamp" // timestamp fields
		}
		// Default for other aggregations is numeric
		return "number"
	}

	// For non-aggregation fields, scan all results to determine type
	var foundTypes = make(map[string]bool)

	for _, result := range results {
		if result[fieldName] != nil && result[fieldName] != "" {
			switch val := result[fieldName].(type) {
			case float64, int, int64:
				foundTypes["number"] = true
			case string:
				// Try to parse as number
				if _, err := strconv.ParseFloat(val, 64); err == nil {
					foundTypes["number"] = true
				} else {
					foundTypes["string"] = true
				}
			case []interface{}:
				foundTypes["array"] = true
			case map[string]interface{}:
				foundTypes["object"] = true
			case bool:
				foundTypes["boolean"] = true
			default:
				foundTypes["string"] = true // fallback
			}
		}
	}

	// Priority: number > string > array > object > boolean
	if foundTypes["number"] {
		return "number"
	}
	if foundTypes["string"] {
		return "string"
	}
	if foundTypes["array"] {
		return "array"
	}
	if foundTypes["object"] {
		return "object"
	}
	if foundTypes["boolean"] {
		return "boolean"
	}

	return "string" // default fallback
}

// Multi version for NRDBResultContainerMultiResultCustomized
func formatStandardQueryMulti(results *nrdb.NRDBResultContainerMultiResultCustomized, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}
	frame := data.NewFrame(utils.StandardResponseFrameName)

	fieldNames := extractFieldNamesMulti(results)
	times := createTimeFieldMulti(results, query)
	frame.Fields = append(frame.Fields, data.NewField(utils.TimeFieldName, nil, times))
	addDataFieldsMulti(frame, results, fieldNames)
	resp.Frames = append(resp.Frames, frame)
	return resp
}

// Multi version for NRDBResultContainerMultiResultCustomized
func extractFieldNamesMulti(results *nrdb.NRDBResultContainerMultiResultCustomized) []string {
	fieldNamesMap := make(map[string]struct{})
	for _, result := range results.Results {
		for key := range result {
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

// Multi version for NRDBResultContainerMultiResultCustomized
func createTimeFieldMulti(results *nrdb.NRDBResultContainerMultiResultCustomized, query backend.DataQuery) []time.Time {
	times := make([]time.Time, len(results.Results))
	now := time.Now()
	for i, result := range results.Results {
		if ts, ok := result[utils.TimestampFieldName].(float64); ok {
			times[i] = time.Unix(int64(ts/1000), 0)
		} else if beginTs, ok := result["beginTimeSeconds"].(float64); ok {
			times[i] = time.Unix(int64(beginTs), 0)
		} else {
			times[i] = now
		}
	}
	return times
}

// Multi version for NRDBResultContainerMultiResultCustomized
func addDataFieldsMulti(frame *data.Frame, results *nrdb.NRDBResultContainerMultiResultCustomized, fieldNames []string) {
	for _, fieldName := range fieldNames {
		if len(results.Results) > 0 {
			// Use improved field type detection (convert to regular results for type detection)
			regularResults := make([]nrdb.NRDBResult, len(results.Results))
			for i, result := range results.Results {
				regularResults[i] = result
			}
			fieldType := detectFieldType(regularResults, fieldName)

			switch fieldType {
			case "number":
				// Use nullable float64 to handle nil/empty values properly
				values := make([]*float64, len(results.Results))
				for i, result := range results.Results {
					if result[fieldName] != nil && result[fieldName] != "" {
						if val, ok := result[fieldName].(float64); ok {
							values[i] = &val
						} else if strVal, ok := result[fieldName].(string); ok && strVal != "" {
							// Try to parse scientific notation strings
							if parsed, err := parseNumericString(strVal); err == nil {
								values[i] = &parsed
							}
						} else if intVal, ok := result[fieldName].(int); ok {
							floatVal := float64(intVal)
							values[i] = &floatVal
						} else if int64Val, ok := result[fieldName].(int64); ok {
							floatVal := float64(int64Val)
							values[i] = &floatVal
						}
					}
					// For nil/empty values, values[i] remains nil (which becomes null in JSON)
				}
				frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))

			case "timestamp":
				// Handle timestamp fields specially
				values := make([]*time.Time, len(results.Results))
				for i, result := range results.Results {
					if result[fieldName] != nil && result[fieldName] != "" {
						if timestampVal, ok := result[fieldName].(float64); ok {
							// Convert Unix timestamp to time.Time
							t := time.Unix(int64(timestampVal/1000), 0)
							values[i] = &t
						} else if timestampStr, ok := result[fieldName].(string); ok {
							// Try to parse timestamp string
							if parsed, err := strconv.ParseFloat(timestampStr, 64); err == nil {
								t := time.Unix(int64(parsed/1000), 0)
								values[i] = &t
							}
						}
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))

			case "array":
				// Handle arrays (like histogram, uniques) - convert to JSON string for display
				values := make([]string, len(results.Results))
				for i, result := range results.Results {
					if result[fieldName] != nil {
						if arrayVal, ok := result[fieldName].([]interface{}); ok {
							// Convert array to JSON string for better display
							if jsonBytes, err := json.Marshal(arrayVal); err == nil {
								values[i] = string(jsonBytes)
							} else {
								values[i] = fmt.Sprintf("%v", arrayVal)
							}
						} else {
							values[i] = fmt.Sprintf("%v", result[fieldName])
						}
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))

			case "object":
				// Handle objects (like percentile results) - convert to JSON string or extract values
				if strings.HasPrefix(fieldName, "percentile.") {
					// Special handling for percentile objects - try to extract individual percentile values
					handlePercentileFieldMulti(frame, results, fieldName)
				} else {
					// General object handling - convert to JSON string
					values := make([]string, len(results.Results))
					for i, result := range results.Results {
						if result[fieldName] != nil {
							if objVal, ok := result[fieldName].(map[string]interface{}); ok {
								if jsonBytes, err := json.Marshal(objVal); err == nil {
									values[i] = string(jsonBytes)
								} else {
									values[i] = fmt.Sprintf("%v", objVal)
								}
							} else {
								values[i] = fmt.Sprintf("%v", result[fieldName])
							}
						}
					}
					frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))
				}

			case "boolean":
				// Handle boolean values
				values := make([]*bool, len(results.Results))
				for i, result := range results.Results {
					if result[fieldName] != nil {
						if boolVal, ok := result[fieldName].(bool); ok {
							values[i] = &boolVal
						}
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))

			default: // "string" and fallback
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

// handlePercentileFieldMulti handles percentile objects by creating separate fields for each percentile (Multi version)
func handlePercentileFieldMulti(frame *data.Frame, results *nrdb.NRDBResultContainerMultiResultCustomized, fieldName string) {
	// Collect all percentile keys from all results
	percentileKeys := make(map[string]bool)
	for _, result := range results.Results {
		if result[fieldName] != nil {
			if objVal, ok := result[fieldName].(map[string]interface{}); ok {
				for key := range objVal {
					percentileKeys[key] = true
				}
			}
		}
	}

	// Create a field for each percentile
	for percentileKey := range percentileKeys {
		fieldNameWithPercentile := fmt.Sprintf("%s.%s", fieldName, percentileKey)
		values := make([]*float64, len(results.Results))

		for i, result := range results.Results {
			if result[fieldName] != nil {
				if objVal, ok := result[fieldName].(map[string]interface{}); ok {
					if percentileVal, exists := objVal[percentileKey]; exists {
						if floatVal, ok := percentileVal.(float64); ok {
							values[i] = &floatVal
						} else if strVal, ok := percentileVal.(string); ok {
							if parsed, err := parseNumericString(strVal); err == nil {
								values[i] = &parsed
							}
						}
					}
				}
			}
		}

		frame.Fields = append(frame.Fields, data.NewField(fieldNameWithPercentile, nil, values))
	}
}
