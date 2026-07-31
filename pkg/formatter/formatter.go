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

	log.DefaultLogger.Debug("Result count", "count", len(results.Results))

	if len(results.Results) == 0 {
		return resp
	}

	// Route to appropriate formatter based on query type
	if isCompareWithQuery(results) {
		return formatCompareWithQuery(results, query)
	} else if isSimpleCountQuery(results) {
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

// isCompareWithQuery checks if the results contain COMPARE WITH data.
// NR returns a "comparison" field with values "current" and "previous".
func isCompareWithQuery(results *nrdb.NRDBResultContainer) bool {
	if len(results.Results) < 2 {
		return false
	}
	_, hasComparison := results.Results[0]["comparison"]
	return hasComparison
}

// formatCompareWithQuery formats COMPARE WITH results into separate current/previous frames.
// Current values are displayed as primary metrics; previous values get a "previous" label.
func formatCompareWithQuery(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	// Separate current and previous results
	var currentResults, previousResults []nrdb.NRDBResult
	for _, result := range results.Results {
		if comp, ok := result["comparison"].(string); ok {
			switch comp {
			case "current":
				currentResults = append(currentResults, result)
			case "previous":
				previousResults = append(previousResults, result)
			}
		}
	}

	log.DefaultLogger.Debug("COMPARE WITH query detected",
		"currentCount", len(currentResults),
		"previousCount", len(previousResults))

	// Collect metric field names from current results (exclude system fields)
	excludeFields := map[string]bool{
		"comparison": true, "timestamp": true, "inspectedCount": true,
		"beginTimeSeconds": true, "endTimeSeconds": true,
	}

	var metricFields []string
	if len(currentResults) > 0 {
		for key := range currentResults[0] {
			if !excludeFields[key] {
				metricFields = append(metricFields, key)
			}
		}
	}

	now := time.Now()

	// Create frame for current values (primary display)
	currentFrame := data.NewFrame("current")
	currentFrame.Fields = append(currentFrame.Fields,
		data.NewField("time", nil, []time.Time{now}))

	for _, fieldName := range metricFields {
		if len(currentResults) > 0 {
			if val, ok := currentResults[0][fieldName].(float64); ok {
				currentFrame.Fields = append(currentFrame.Fields,
					data.NewField(fieldName, nil, []float64{val}))
			}
		}
	}
	resp.Frames = append(resp.Frames, currentFrame)

	// Create frame for previous values with comparison label
	if len(previousResults) > 0 {
		previousFrame := data.NewFrame("previous")
		previousFrame.Fields = append(previousFrame.Fields,
			data.NewField("time", nil, []time.Time{now.Add(-time.Hour)}))

		for _, fieldName := range metricFields {
			if val, ok := previousResults[0][fieldName].(float64); ok {
				labels := map[string]string{"comparison": "previous"}
				previousFrame.Fields = append(previousFrame.Fields,
					data.NewField(fieldName, labels, []float64{val}))
			}
		}
		resp.Frames = append(resp.Frames, previousFrame)
	}

	return resp
}

// isSimpleCountQuery checks if the results represent a simple count query.
// It verifies that "count" is the ONLY meaningful metric field in the result.
// This prevents misdetecting apdex queries (which also have "count" alongside
// "score", "s", "t", "f", and "apdex.duration") as simple count queries.
func isSimpleCountQuery(results *nrdb.NRDBResultContainer) bool {
	if len(results.Results) != 1 ||
		results.Results[0][utils.CountFieldName] == nil ||
		results.Results[0][utils.FacetFieldName] != nil {
		return false
	}

	// Verify count is the ONLY metric field (exclude internal fields)
	internalFields := map[string]bool{
		"count": true, "inspectedCount": true,
		"beginTimeSeconds": true, "endTimeSeconds": true,
		"timestamp": true,
	}
	for key := range results.Results[0] {
		if !internalFields[key] {
			return false // Has additional fields (apdex score, s, t, f, etc.)
		}
	}
	return true
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
	// For faceted timeseries queries, the data is in OtherResult/TotalResult, not Results
	// Check if this has facet metadata and timeseries data in OtherResult
	hasFacet := len(results.Metadata.Facets) > 0
	hasTimeseries := hasTimeseriesDataMulti(results)

	log.DefaultLogger.Debug("Detection Logic Debug",
		"facetsInMetadata", results.Metadata.Facets,
		"hasFacet", hasFacet,
		"hasTimeseries", hasTimeseries,
		"resultsLength", len(results.Results),
		"otherResultLength", len(results.OtherResult),
		"totalResultLength", len(results.TotalResult))

	// For faceted timeseries, we need both facets and timeseries data
	// The aggregation field can be anything (count, sum.duration, average.duration, etc.)
	result := hasFacet && hasTimeseries
	log.DefaultLogger.Debug("Final detection result", "isFacetedTimeseries", result)
	return result
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
	// Check OtherResult first (this is where faceted timeseries data is stored)
	if len(results.OtherResult) > 0 {
		_, hasBeginTime := results.OtherResult[0]["beginTimeSeconds"]
		_, hasEndTime := results.OtherResult[0]["endTimeSeconds"]
		log.DefaultLogger.Debug("Checking OtherResult for timeseries",
			"hasBeginTime", hasBeginTime,
			"hasEndTime", hasEndTime,
			"firstEntry", results.OtherResult[0])
		if hasBeginTime || hasEndTime {
			return true
		}
	}

	// Fallback to Results if OtherResult is empty
	if len(results.Results) > 0 {
		_, hasBeginTime := results.Results[0]["beginTimeSeconds"]
		_, hasEndTime := results.Results[0]["endTimeSeconds"]
		log.DefaultLogger.Debug("Checking Results for timeseries (fallback)",
			"hasBeginTime", hasBeginTime,
			"hasEndTime", hasEndTime)
		return hasBeginTime || hasEndTime
	}

	log.DefaultLogger.Debug("No timeseries data found in either OtherResult or Results")
	return false
}

// formatSimpleCountQuery formats results from a simple count query.
// Returns a single frame with the count value. Grafana stat/table panels
// display this directly. For time series panels, users should use TIMESERIES.
func formatSimpleCountQuery(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	// Extract count value
	count := extractCountValue(results.Results[0])

	// Single frame with count value - works in stat, table, and gauge panels
	valueFrame := data.NewFrame("count",
		data.NewField("count", nil, []float64{count}),
	)
	valueFrame.Meta = &data.FrameMeta{
		PreferredVisualization: data.VisTypeTable,
	}

	resp.Frames = append(resp.Frames, valueFrame)
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

// Helper function for tests - multi version of extractCountValue
func extractCountValueMulti(result nrdb.NRDBResult) float64 {
	if countValue, ok := result[utils.CountFieldName].(float64); ok {
		return countValue
	}
	return 0.0
}

// formatFacetedCountQuery formats results from a faceted count query
func formatFacetedCountQuery(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	// Get facet names
	facetNames := extractFacetNames(results)
	log.DefaultLogger.Debug("Facet names extracted", "facetNames", facetNames)

	// Extract data
	counts, facetFields := extractFacetedData(results, facetNames)
	log.DefaultLogger.Debug("Extracted faceted data", "counts", counts, "facetFields", facetFields)

	// Create separate frames for each result with ALL facet dimensions as labels
	if len(facetNames) > 0 {
		for i := range results.Results {
			frame := data.NewFrame("")

			// Add time field
			now := time.Now()
			frame.Fields = append(frame.Fields,
				data.NewField("time", nil, []time.Time{now}))

			// Build labels from ALL facet dimensions (not just the first one)
			labels := make(map[string]string)
			for _, facetName := range facetNames {
				if values, ok := facetFields[facetName]; ok && i < len(values) {
					labels[facetName] = values[i]
				}
			}

			countField := data.NewField("count", labels, []float64{counts[i]})
			frame.Fields = append(frame.Fields, countField)

			resp.Frames = append(resp.Frames, frame)
		}
	}

	log.DefaultLogger.Debug("Total frames in response", "frameCount", len(resp.Frames))

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

		log.DefaultLogger.Debug("Creating pie chart frame", "facet", facetName, "labels", labels, "counts", counts)

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

// extractAliasesFromMetadataMulti extracts metric field names from results for Multi type
// Since the flattened response doesn't preserve alias metadata, we extract field names from actual results
// Returns an array of metric field names like ["Errors", "Success", "Error %"] or ["count", "sum.duration", etc.]
func extractAliasesFromMetadataMulti(results *nrdb.NRDBResultContainerMultiResultCustomized) []string {
	var fieldNames []string
	seenFields := make(map[string]bool)

	// Build a set of facet names to exclude facet value fields (e.g., "host", "appName")
	// These are duplicates of the "facet" field and should not be treated as metrics
	facetNameSet := make(map[string]bool)
	for _, fn := range extractFacetNamesMulti(results) {
		facetNameSet[fn] = true
	}

	// Get the results to process
	resultsToProcess := results.Results
	if len(resultsToProcess) == 0 {
		resultsToProcess = results.OtherResult
	}

	// Extract unique field names from the first few results
	// (they should all have the same structure)
	for i, result := range resultsToProcess {
		if i >= 5 { // Only check first 5 results for efficiency
			break
		}

		for key := range result {
			// Skip non-metric fields AND facet value fields (e.g., "host", "request.uri")
			if key != "timestamp" && key != "beginTimeSeconds" && key != "endTimeSeconds" &&
			   key != "facet" && key != "inspectedCount" && !seenFields[key] && !facetNameSet[key] {
				fieldNames = append(fieldNames, key)
				seenFields[key] = true
			}
		}
	}

	log.DefaultLogger.Debug("Extracted metric field names from results (Multi)", "fieldNames", fieldNames, "count", len(fieldNames))
	return fieldNames
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
		// Create labels for this field
		labels := make(map[string]string)
		// If we have values for this facet, set the first one as a label
		if len(facetFields[facetName]) > 0 {
			labels[facetName] = facetFields[facetName][0]
		}
		timeSeriesFrame.Fields = append(timeSeriesFrame.Fields,
			data.NewField(facetName, labels, facetFields[facetName]))
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

	// Add time field with bucket interval
	times := createTimeField(results, query)
	timeField := data.NewField(utils.TimeFieldName, nil, times)
	setTimeFieldInterval(timeField, calculateBucketIntervalMs(results.Results))
	frame.Fields = append(frame.Fields, timeField)

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

	// Group results by ALL facet values (not just the first one)
	facetData := groupResultsByAllFacets(results, facetNames)
	log.DefaultLogger.Debug("Faceted aggregation - Grouped facet groups", "groupCount", len(facetData))

	// Get all field names and filter to only include aggregation fields
	allFieldNames := extractFieldNames(results)

	var aggregationFields []string
	for _, fieldName := range allFieldNames {
		// Exclude facet-related fields and only include aggregation fields
		if fieldName != utils.FacetFieldName && !isFacetFieldName(fieldName, facetNames) && isAggregationField(fieldName) {
			aggregationFields = append(aggregationFields, fieldName)
		}
	}

	log.DefaultLogger.Debug("Faceted aggregation - Aggregation fields", "fields", aggregationFields)

	// Create separate frames for each facet combination
	for facetKey, facetInfo := range facetData {
		frame := data.NewFrame(facetKey)
		times := createTimeField(&nrdb.NRDBResultContainer{Results: facetInfo.Results}, query)
		timeField := data.NewField("time", nil, times)
		setTimeFieldInterval(timeField, calculateBucketIntervalMs(facetInfo.Results))
		frame.Fields = append(frame.Fields, timeField)

		// Add aggregation fields with facet labels
		for _, fieldName := range aggregationFields {
			// Handle different aggregation field types
			if strings.HasPrefix(fieldName, "percentile.") {
				// Handle percentile objects - extract individual percentile values
				addPercentileFieldsWithLabels(frame, facetInfo.Results, fieldName, facetInfo.Labels)
			} else {
				// Handle regular aggregation fields (sum.duration, average.duration, etc.)
				addRegularAggregationFieldWithLabels(frame, facetInfo.Results, fieldName, facetInfo.Labels)
			}
		}

		resp.Frames = append(resp.Frames, frame)
	}

	log.DefaultLogger.Debug("Faceted aggregation - Total frames in response", "frameCount", len(resp.Frames))
	return resp
}

// addPercentileFields handles percentile objects by extracting individual percentile values
func addPercentileFields(frame *data.Frame, facetResults []nrdb.NRDBResult, fieldName, facetName, facetValue string) {
	// First pass: collect all percentile keys across all results
	percentileKeys := make(map[string]bool)
	for _, result := range facetResults {
		if result[fieldName] != nil {
			if objVal, ok := result[fieldName].(map[string]interface{}); ok {
				for key := range objVal {
					percentileKeys[key] = true
				}
			}
		}
	}

	// Create a field for each percentile (e.g., percentile.duration.95)
	for percentileKey := range percentileKeys {
		fieldNameWithPercentile := fmt.Sprintf("%s.%s", fieldName, percentileKey)
		labels := map[string]string{
			facetName: facetValue,
		}

		// Extract values for this specific percentile
		values := make([]*float64, len(facetResults))
		for i, result := range facetResults {
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

		// Create field with facet label
		field := data.NewField(fieldNameWithPercentile, labels, values)
		frame.Fields = append(frame.Fields, field)
	}
}

// addRegularAggregationField handles regular aggregation fields (sum.duration, average.duration, etc.)
func addRegularAggregationField(frame *data.Frame, facetResults []nrdb.NRDBResult, fieldName, facetName, facetValue string) {
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

	// Create field with facet label
	field := data.NewField(fieldName, labels, values)
	frame.Fields = append(frame.Fields, field)
}

// addPercentileFieldsWithLabels handles percentile objects with multiple facet labels
func addPercentileFieldsWithLabels(frame *data.Frame, facetResults []nrdb.NRDBResult, fieldName string, labels map[string]string) {
	// First pass: collect all percentile keys across all results
	percentileKeys := make(map[string]bool)
	for _, result := range facetResults {
		if result[fieldName] != nil {
			if objVal, ok := result[fieldName].(map[string]interface{}); ok {
				for key := range objVal {
					percentileKeys[key] = true
				}
			}
		}
	}

	// Create a field for each percentile (e.g., percentile.duration.95)
	for percentileKey := range percentileKeys {
		fieldNameWithPercentile := fmt.Sprintf("%s.%s", fieldName, percentileKey)

		// Extract values for this specific percentile
		values := make([]*float64, len(facetResults))
		for i, result := range facetResults {
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

		// Create field with all facet labels
		field := data.NewField(fieldNameWithPercentile, labels, values)
		frame.Fields = append(frame.Fields, field)
	}
}

// addRegularAggregationFieldWithLabels handles regular aggregation fields with multiple facet labels
func addRegularAggregationFieldWithLabels(frame *data.Frame, facetResults []nrdb.NRDBResult, fieldName string, labels map[string]string) {
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

	// Create field with all facet labels
	field := data.NewField(fieldName, labels, values)
	frame.Fields = append(frame.Fields, field)
}

// isFacetFieldName checks if a field name corresponds to a facet field name
func isFacetFieldName(fieldName string, facetNames []string) bool {
	for _, facetName := range facetNames {
		if fieldName == facetName {
			return true
		}
	}
	return false
}

// FacetGroupInfo holds results and labels for a facet group
type FacetGroupInfo struct {
	Results []nrdb.NRDBResult
	Labels  map[string]string
}

// groupResultsByAllFacets groups results by ALL facet values (multi-dimensional)
func groupResultsByAllFacets(results *nrdb.NRDBResultContainer, facetNames []string) map[string]*FacetGroupInfo {
	grouped := make(map[string]*FacetGroupInfo)

	for _, result := range results.Results {
		// Extract all facet values
		facetValues := extractAllFacetValues(result, facetNames)

		if len(facetValues) == 0 {
			continue
		}

		// Create a composite key from all facet values
		facetKey := createCompositeFacetKey(facetValues, facetNames)

		// Create labels map
		labels := make(map[string]string)
		for i, facetName := range facetNames {
			if i < len(facetValues) {
				labels[facetName] = facetValues[i]
			}
		}

		// Group by composite key
		if _, exists := grouped[facetKey]; !exists {
			grouped[facetKey] = &FacetGroupInfo{
				Results: []nrdb.NRDBResult{},
				Labels:  labels,
			}
		}
		grouped[facetKey].Results = append(grouped[facetKey].Results, result)
	}

	return grouped
}

// extractAllFacetValues extracts all facet values from a result
func extractAllFacetValues(result map[string]interface{}, facetNames []string) []string {
	var values []string

	facetData, hasFacet := result[utils.FacetFieldName]
	if !hasFacet {
		return values
	}

	// Handle facet as array (multiple facets)
	if facetArray, ok := facetData.([]interface{}); ok {
		for _, value := range facetArray {
			values = append(values, fmt.Sprintf("%v", value))
		}
	} else {
		// Handle single facet value
		values = append(values, fmt.Sprintf("%v", facetData))
	}

	return values
}

// createCompositeFacetKey creates a unique key from multiple facet values
func createCompositeFacetKey(facetValues []string, facetNames []string) string {
	if len(facetValues) == 0 {
		return ""
	}

	// For single facet, return just the value
	if len(facetValues) == 1 {
		return facetValues[0]
	}

	// For multiple facets, create a descriptive key
	var parts []string
	for i, value := range facetValues {
		if i < len(facetNames) {
			// Use facet name for clarity: "Server Error, grafana-traffic-generator, host1"
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, ", ")
}

// groupResultsByFacet groups results by facet value for aggregation queries (legacy - single facet)
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

// extractFieldNames extracts unique field names from results.
// When an apdex.* object field is present, it filters out the duplicate top-level
// fields (count, f, s, score, t) that NR returns redundantly alongside the nested object.
func extractFieldNames(results *nrdb.NRDBResultContainer) []string {
	fieldNamesMap := make(map[string]struct{})
	hasApdexObject := false

	for _, result := range results.Results {
		for key := range result {
			// Exclude timestamp fields and New Relic TIMESERIES fields from data fields
			if key != utils.TimestampFieldName && key != "beginTimeSeconds" && key != "endTimeSeconds" {
				fieldNamesMap[key] = struct{}{}
			}
			if strings.HasPrefix(key, "apdex.") {
				hasApdexObject = true
			}
		}
	}

	// When apdex object is present, NR duplicates its sub-fields at top level.
	// Remove the duplicates to avoid showing each value twice.
	if hasApdexObject {
		apdexDuplicates := []string{"count", "f", "s", "score", "t"}
		for _, dup := range apdexDuplicates {
			delete(fieldNamesMap, dup)
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

// calculateBucketIntervalMs calculates the timeseries bucket interval in milliseconds
// from the first result that has both beginTimeSeconds and endTimeSeconds.
// Returns 0 if no interval can be determined.
func calculateBucketIntervalMs(results []nrdb.NRDBResult) float64 {
	for _, result := range results {
		beginTs, hasBegin := result["beginTimeSeconds"].(float64)
		endTs, hasEnd := result["endTimeSeconds"].(float64)
		if hasBegin && hasEnd && endTs > beginTs {
			return (endTs - beginTs) * 1000 // Convert seconds to milliseconds
		}
	}
	return 0
}

// setTimeFieldInterval sets the bucket interval on a time field's config if interval > 0
func setTimeFieldInterval(timeField *data.Field, intervalMs float64) {
	if intervalMs > 0 {
		if timeField.Config == nil {
			timeField.Config = &data.FieldConfig{}
		}
		timeField.Config.Interval = intervalMs
	}
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
				// Handle arrays (like histogram, uniques) - convert to comma-separated string
				values := make([]string, len(results.Results))
				for i, result := range results.Results {
					if result[fieldName] != nil {
						if arrayVal, ok := result[fieldName].([]interface{}); ok {
							parts := make([]string, len(arrayVal))
							for j, elem := range arrayVal {
								parts[j] = fmt.Sprintf("%v", elem)
							}
							values[i] = strings.Join(parts, ", ")
						} else {
							values[i] = fmt.Sprintf("%v", result[fieldName])
						}
					}
				}
				frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))

			case "object":
				// Handle nested objects - flatten into individual fields for numeric sub-values
				if strings.HasPrefix(fieldName, "percentile.") || strings.HasPrefix(fieldName, "apdex.") {
					// Flatten percentile and apdex objects into individual fields
					// e.g., percentile.duration → percentile.duration.50, percentile.duration.90
					// e.g., apdex.duration → apdex.duration.score, apdex.duration.s, apdex.duration.t
					handleNestedObjectField(frame, results, fieldName)
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

// handleNestedObjectField handles nested object fields (percentile, apdex, etc.)
// by creating separate numeric fields for each sub-key.
// e.g., "percentile.duration" with {"50": 0.01, "90": 0.08} → "percentile.duration.50", "percentile.duration.90"
// e.g., "apdex.duration" with {"score": 0.99, "s": 530908} → "apdex.duration.score", "apdex.duration.s"
func handleNestedObjectField(frame *data.Frame, results *nrdb.NRDBResultContainer, fieldName string) {
	// Collect all sub-keys from all results
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
	log.DefaultLogger.Debug("Faceted timeseries - Facet names extracted", "facetNames", facetNames)

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
	log.DefaultLogger.Debug("Faceted timeseries - Facet names extracted", "facetNames", facetNames)

	if len(facetNames) == 0 {
		return formatStandardQueryMulti(results, query)
	}

	// Use ALL facets, not just the first one
	facetData := groupTimeseriesByAllFacetsMulti(results, facetNames)
	log.DefaultLogger.Debug("Faceted timeseries - Grouped facet groups", "groupCount", len(facetData))

	// Debug facet data
	keys := make([]string, 0, len(facetData))
	for k := range facetData {
		keys = append(keys, k)
	}
	log.DefaultLogger.Debug("Facet data keys", "keys", keys)

	// Extract aliases from metadata for proper metric naming
	aliases := extractAliasesFromMetadataMulti(results)
	log.DefaultLogger.Debug("Extracted aliases from metadata", "aliases", aliases, "count", len(aliases))

	for facetKey, facetInfo := range facetData {
		frame := data.NewFrame(facetKey)
		times := createTimeField(&nrdb.NRDBResultContainer{Results: facetInfo.Results}, query)
		timeField := data.NewField("time", nil, times)
		setTimeFieldInterval(timeField, calculateBucketIntervalMs(facetInfo.Results))
		frame.Fields = append(frame.Fields, timeField)

		// Extract ALL metric fields from the results, not just "count"
		// Use the aliases we extracted (which are the actual field names from results)
		// This preserves the order and names as they appear in the query
		metricFieldNames := aliases
		if len(metricFieldNames) == 0 {
			// Fallback: collect unique metric names from results
			metricNamesSet := make(map[string]bool)
			for _, result := range facetInfo.Results {
				for key := range result {
					// Skip non-metric fields
					if key != "timestamp" && key != "beginTimeSeconds" && key != "endTimeSeconds" &&
					   key != "facet" && key != "inspectedCount" {
						metricNamesSet[key] = true
					}
				}
			}
			// Convert to slice
			for name := range metricNamesSet {
				metricFieldNames = append(metricFieldNames, name)
			}
		}

		// Create a field for each metric
		for _, metricName := range metricFieldNames {
			values := make([]*float64, len(facetInfo.Results))

			for i, result := range facetInfo.Results {
				if val, ok := result[metricName].(float64); ok {
					values[i] = &val
				}
			}

			// Use all facet labels, not just the first one
			field := data.NewField(metricName, facetInfo.Labels, values)
			frame.Fields = append(frame.Fields, field)
		}

		resp.Frames = append(resp.Frames, frame)
	}

	log.DefaultLogger.Debug("Faceted timeseries - Total frames in response", "frameCount", len(resp.Frames))

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

// groupTimeseriesByAllFacetsMulti groups timeseries results by ALL facet values for Multi type
func groupTimeseriesByAllFacetsMulti(results *nrdb.NRDBResultContainerMultiResultCustomized, facetNames []string) map[string]*FacetGroupInfo {
	grouped := make(map[string]*FacetGroupInfo)

	// Process data from OtherResult first, as it's the preferred location for faceted timeseries
	resultsToProcess := results.OtherResult
	if len(resultsToProcess) == 0 {
		// Fallback to Results if OtherResult is empty
		resultsToProcess = results.Results
	}

	for _, result := range resultsToProcess {
		// Extract all facet values
		facetValues := extractAllFacetValues(result, facetNames)

		if len(facetValues) == 0 {
			continue
		}

		// Create a composite key from all facet values
		facetKey := createCompositeFacetKey(facetValues, facetNames)

		// Create labels map
		labels := make(map[string]string)
		for i, facetName := range facetNames {
			if i < len(facetValues) {
				labels[facetName] = facetValues[i]
			}
		}

		// Group by composite key
		if _, exists := grouped[facetKey]; !exists {
			grouped[facetKey] = &FacetGroupInfo{
				Results: []nrdb.NRDBResult{},
				Labels:  labels,
			}
		}
		grouped[facetKey].Results = append(grouped[facetKey].Results, result)
	}

	return grouped
}

// Multi version for NRDBResultContainerMultiResultCustomized (legacy - single facet)
func groupTimeseriesByFacetMulti(results *nrdb.NRDBResultContainerMultiResultCustomized, facetName string) map[string][]nrdb.NRDBResult {
	grouped := make(map[string][]nrdb.NRDBResult)

	// Process data from OtherResult first, as it's the preferred location for faceted timeseries
	resultsToProcess := results.OtherResult
	if len(resultsToProcess) == 0 {
		// Fallback to Results if OtherResult is empty
		resultsToProcess = results.Results
	}

	for _, result := range resultsToProcess {
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

	log.DefaultLogger.Debug("FormatFacetedTimeseriesResults",
		"resultCount", len(results.Results))

	if !isFacetedTimeseriesQueryMulti(results) {
		resp := &backend.DataResponse{}
		resp.Error = fmt.Errorf("results are not a faceted timeseries query")
		return resp
	}

	// For faceted timeseries queries, the data can be in either OtherResult or Results
	// Convert to standard format and use the enhanced faceted aggregation formatter
	var actualResults []nrdb.NRDBResult

	// First check Results as it's the preferred location for faceted timeseries
	if len(results.Results) > 0 {
		actualResults = results.Results
		log.DefaultLogger.Debug("Using Results", "entryCount", len(actualResults))
	} else {
		actualResults = results.OtherResult
		log.DefaultLogger.Debug("Using OtherResult", "entryCount", len(actualResults))
	}

	standardResults := &nrdb.NRDBResultContainer{
		Results:  make([]nrdb.NRDBResult, len(actualResults)),
		Metadata: results.Metadata,
	}

	// Copy the actual results data
	for i, result := range actualResults {
		standardResults.Results[i] = result
	}

	// Get facet names from metadata (not from result data)
	facetNames := extractFacetNames(standardResults)
	if len(facetNames) == 0 {
		// No facets found, fall back to standard query
		return formatStandardQuery(standardResults, query)
	}

	// Use the enhanced faceted aggregation formatter
	return formatFacetedAggregationQuery(standardResults, query, facetNames)
}

// isAggregationField checks if a field name represents a metric/aggregation result.
// Uses a blacklist approach: any field that isn't an internal/system field is considered
// an aggregation field. This supports arbitrary user-defined aliases (e.g., 'Total', 'My Metric').
func isAggregationField(fieldName string) bool {
	// Exclude known internal/system fields that are never metrics
	internalFields := map[string]bool{
		"facet":            true,
		"inspectedCount":   true,
		"timestamp":        true,
		"beginTimeSeconds": true,
		"endTimeSeconds":   true,
	}

	if internalFields[fieldName] {
		return false
	}

	// Any non-internal field is treated as a metric/aggregation field
	return true
}

// detectFieldType analyzes a field across all results to determine the best data type.
// Uses field name prefixes for known NR types, then inspects actual values for all others.
func detectFieldType(results []nrdb.NRDBResult, fieldName string) string {
	// Handle known NR field name patterns that have specific types
	if strings.HasPrefix(fieldName, "histogram.") {
		return "array"
	}
	if strings.HasPrefix(fieldName, "uniques.") {
		return "array"
	}
	if strings.HasPrefix(fieldName, "percentile.") {
		parts := strings.Split(fieldName, ".")
		if len(parts) >= 3 {
			return "number" // specific percentile like "percentile.duration.95"
		}
		return "object" // generic percentile object like "percentile.duration"
	}
	if strings.HasPrefix(fieldName, "apdex.") {
		parts := strings.Split(fieldName, ".")
		if len(parts) >= 3 {
			return "number" // specific apdex field like "apdex.duration.score"
		}
		return "object" // generic apdex object like "apdex.duration"
	}
	if strings.HasPrefix(fieldName, "earliest.timestamp") || strings.HasPrefix(fieldName, "latest.timestamp") {
		return "timestamp"
	}

	// For all other fields, scan actual values to determine type
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
	timeField := data.NewField(utils.TimeFieldName, nil, times)
	// Convert multi-result Results to standard NRDBResult slice for interval calculation
	standardResults := make([]nrdb.NRDBResult, len(results.Results))
	for i, r := range results.Results {
		standardResults[i] = r
	}
	setTimeFieldInterval(timeField, calculateBucketIntervalMs(standardResults))
	frame.Fields = append(frame.Fields, timeField)
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
				// Handle arrays (like histogram, uniques) - convert to comma-separated string
				values := make([]string, len(results.Results))
				for i, result := range results.Results {
					if result[fieldName] != nil {
						if arrayVal, ok := result[fieldName].([]interface{}); ok {
							parts := make([]string, len(arrayVal))
							for j, elem := range arrayVal {
								parts[j] = fmt.Sprintf("%v", elem)
							}
							values[i] = strings.Join(parts, ", ")
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

// getMapKeys returns the keys of a map as a slice for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// HandlePercentileFieldMulti is an exported version of handlePercentileFieldMulti for testing
func HandlePercentileFieldMulti(frame *data.Frame, results *nrdb.NRDBResultContainerMultiResultCustomized, fieldName string) {
	handlePercentileFieldMulti(frame, results, fieldName)
}
