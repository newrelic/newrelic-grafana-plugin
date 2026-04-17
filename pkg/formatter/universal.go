// Package formatter handles the conversion of New Relic API responses
// into Grafana data frames using a universal, metadata-driven approach.
package formatter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// FlexibleResults handles JSON fields that may be either a single object or an
// array of objects. The New Relic API returns OtherResult/TotalResult as a
// single map for non-TIMESERIES FACET queries, but as an array for TIMESERIES
// FACET queries. This type normalises both shapes into []map[string]interface{}.
type FlexibleResults []map[string]interface{}

func (f *FlexibleResults) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	// Array → unmarshal directly
	if data[0] == '[' {
		var arr []map[string]interface{}
		if err := json.Unmarshal(data, &arr); err != nil {
			return err
		}
		*f = arr
		return nil
	}

	// Single object → wrap in a slice
	if data[0] == '{' {
		var obj map[string]interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			return err
		}
		*f = []map[string]interface{}{obj}
		return nil
	}

	return fmt.Errorf("unexpected JSON type for FlexibleResults: %c", data[0])
}

// NormalizedDataPoint represents a single data point after normalization
type NormalizedDataPoint struct {
	Timestamp      time.Time
	FacetValues    map[string]string    // Facet dimension values
	Metrics        map[string]float64   // All metric values (flattened)
	IntervalMs     float64              // Bucket interval in milliseconds (0 if unknown)
}

// UniversalResponse is a generic structure to handle any New Relic response
type UniversalResponse struct {
	Results      FlexibleResults  `json:"Results"`
	OtherResult  FlexibleResults  `json:"OtherResult"`
	TotalResult  FlexibleResults  `json:"TotalResult"`
	Facets       []FacetGroup     `json:"facets"`
	Metadata     ResponseMetadata `json:"metadata"`
}

// FacetGroup represents a faceted timeseries group
type FacetGroup struct {
	Name             interface{}          `json:"name"` // Can be string or array
	BeginTimeSeconds int64                `json:"beginTimeSeconds"`
	EndTimeSeconds   int64                `json:"endTimeSeconds"`
	TimeSeries       []TimeSeriesBucket   `json:"timeSeries"`
	Results          []map[string]interface{} `json:"results"`
}

// TimeSeriesBucket represents a single timeseries bucket
type TimeSeriesBucket struct {
	Results          []map[string]interface{} `json:"results"`
	BeginTimeSeconds int64                    `json:"beginTimeSeconds"`
	EndTimeSeconds   int64                    `json:"endTimeSeconds"`
	InspectedCount   int                      `json:"inspectedCount"`
}

// ResponseMetadata holds metadata about the query
type ResponseMetadata struct {
	Facets   []string                `json:"facets"`
	Facet    []string                `json:"facet"` // Can be array in some responses
	Contents MetadataContents        `json:"contents"`
}

// MetadataContents describes the query structure
type MetadataContents struct {
	TimeSeries   TimeSeriesMetadata  `json:"timeSeries"`
	Function     string              `json:"function"`
	Messages     []interface{}       `json:"messages"`
}

// TimeSeriesMetadata describes timeseries query structure
type TimeSeriesMetadata struct {
	Contents []FunctionMetadata `json:"contents"`
	Messages []interface{}      `json:"messages"`
}

// FunctionMetadata describes a function in the query
type FunctionMetadata struct {
	Function   string              `json:"function"`
	Alias      string              `json:"alias"`
	Attribute  string              `json:"attribute"`
	Contents   *FunctionMetadata   `json:"contents,omitempty"`
	Thresholds []float64           `json:"thresholds,omitempty"`
}

// FormatUniversal is the universal entry point for formatting any New Relic response
func FormatUniversal(result interface{}, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	// Convert to JSON and back to generic structure for introspection
	jsonData, err := json.Marshal(result)
	if err != nil {
		resp.Error = fmt.Errorf("failed to marshal result: %w", err)
		log.DefaultLogger.Error("Failed to marshal result", "error", err)
		return resp
	}

	log.DefaultLogger.Debug("Universal formatter received response", "size", len(jsonData))

	var universal UniversalResponse
	if err := json.Unmarshal(jsonData, &universal); err != nil {
		resp.Error = fmt.Errorf("failed to unmarshal result: %w", err)
		log.DefaultLogger.Error("Failed to unmarshal result", "error", err)
		return resp
	}

	// Detect response structure and normalize
	dataPoints := normalizeResponse(&universal)

	if len(dataPoints) == 0 {
		log.DefaultLogger.Debug("No data points extracted from response")
		return resp
	}

	log.DefaultLogger.Debug("Normalized data points", "count", len(dataPoints))

	// Extract facet names
	facetNames := extractFacetNamesFromResponse(&universal)

	// Build data frames
	frames := buildUniversalFrames(dataPoints, facetNames, query)
	resp.Frames = frames

	log.DefaultLogger.Debug("Built data frames", "frameCount", len(frames))

	return resp
}

// normalizeResponse converts any response format into normalized data points
func normalizeResponse(resp *UniversalResponse) []NormalizedDataPoint {
	var points []NormalizedDataPoint

	// Case 1: Nested facet structure (e.g., cdfPercentage with FACET TIMESERIES)
	if len(resp.Facets) > 0 {
		log.DefaultLogger.Debug("Detected nested facet structure", "facetCount", len(resp.Facets))
		points = normalizeNestedFacets(resp)
	} else if len(resp.OtherResult) > 0 {
		// Case 2: Multi-result structure (FACET + TIMESERIES)
		log.DefaultLogger.Debug("Detected multi-result structure", "otherResultCount", len(resp.OtherResult))
		points = normalizeMultiResult(resp)
	} else if len(resp.Results) > 0 {
		// Case 3: Standard results structure
		log.DefaultLogger.Debug("Detected standard results structure", "resultCount", len(resp.Results))
		points = normalizeStandardResults(resp)
	}

	return points
}

// normalizeNestedFacets handles the nested facet structure
func normalizeNestedFacets(resp *UniversalResponse) []NormalizedDataPoint {
	var points []NormalizedDataPoint

	facetNames := extractFacetNamesFromResponse(resp)

	// Extract aliases from metadata for proper metric naming
	aliases := extractAliasesFromMetadata(resp)

	for _, facetGroup := range resp.Facets {
		// Extract facet values - handle both string and array
		facetValues := extractFacetValuesFromName(facetGroup.Name, facetNames)

		// Process timeseries buckets
		for _, bucket := range facetGroup.TimeSeries {
			timestamp := time.Unix(bucket.BeginTimeSeconds, 0)
			var bucketIntervalMs float64
			if bucket.EndTimeSeconds > bucket.BeginTimeSeconds {
				bucketIntervalMs = float64(bucket.EndTimeSeconds-bucket.BeginTimeSeconds) * 1000
			}

			// Each result in the bucket is a separate metric (e.g., Errors, Success, Error %)
			// Don't aggregate them - keep them separate
			allMetrics := make(map[string]float64)

			for resultIdx, result := range bucket.Results {
				metrics := extractMetricsFromResult(result)

				// Use aliases from metadata if available, otherwise use indexed naming
				var metricPrefix string
				if resultIdx < len(aliases) && aliases[resultIdx] != "" {
					metricPrefix = aliases[resultIdx]
				} else if len(bucket.Results) > 1 {
					metricPrefix = fmt.Sprintf("metric_%d", resultIdx)
				} else {
					metricPrefix = ""
				}

				// Add metrics with appropriate naming
				for key, value := range metrics {
					var metricKey string
					if metricPrefix != "" {
						// Use alias as the metric name if we have one
						// For count fields with aliases, use the alias directly
						if key == "count" || key == "result" {
							metricKey = metricPrefix
						} else {
							metricKey = fmt.Sprintf("%s.%s", metricPrefix, key)
						}
					} else {
						metricKey = key
					}
					allMetrics[metricKey] = value
				}
			}

			if len(allMetrics) > 0 {
				points = append(points, NormalizedDataPoint{
					Timestamp:   timestamp,
					FacetValues: facetValues,
					Metrics:     allMetrics,
					IntervalMs:  bucketIntervalMs,
				})
			}
		}

		// Also handle non-timeseries facet results
		if len(facetGroup.Results) > 0 {
			timestamp := time.Now()
			if facetGroup.BeginTimeSeconds > 0 {
				timestamp = time.Unix(facetGroup.BeginTimeSeconds, 0)
			}

			for _, result := range facetGroup.Results {
				metrics := extractMetricsFromResult(result)
				if len(metrics) > 0 {
					points = append(points, NormalizedDataPoint{
						Timestamp:   timestamp,
						FacetValues: facetValues,
						Metrics:     metrics,
					})
				}
			}
		}
	}

	return points
}

// normalizeMultiResult handles the multi-result structure
func normalizeMultiResult(resp *UniversalResponse) []NormalizedDataPoint {
	var points []NormalizedDataPoint

	// Use OtherResult which typically contains the faceted timeseries data
	resultsToProcess := resp.OtherResult
	if len(resultsToProcess) == 0 {
		resultsToProcess = resp.Results
	}

	facetNames := extractFacetNamesFromResponse(resp)

	for _, result := range resultsToProcess {
		point := normalizeResultMap(result, facetNames)
		points = append(points, point)
	}

	return points
}

// normalizeStandardResults handles the standard results structure
func normalizeStandardResults(resp *UniversalResponse) []NormalizedDataPoint {
	var points []NormalizedDataPoint

	facetNames := extractFacetNamesFromResponse(resp)

	for _, result := range resp.Results {
		point := normalizeResultMap(result, facetNames)
		points = append(points, point)
	}

	return points
}

// normalizeResultMap converts a single result map into a normalized data point
func normalizeResultMap(result map[string]interface{}, facetNames []string) NormalizedDataPoint {
	point := NormalizedDataPoint{
		Timestamp:   extractTimestamp(result),
		FacetValues: extractUniversalFacetValues(result, facetNames),
		Metrics:     extractMetricsFromResult(result),
		IntervalMs:  extractBucketIntervalMs(result),
	}
	return point
}

// extractBucketIntervalMs calculates the bucket interval in milliseconds from a result map
func extractBucketIntervalMs(result map[string]interface{}) float64 {
	beginTs, hasBegin := result["beginTimeSeconds"].(float64)
	endTs, hasEnd := result["endTimeSeconds"].(float64)
	if hasBegin && hasEnd && endTs > beginTs {
		return (endTs - beginTs) * 1000
	}
	return 0
}

// extractTimestamp extracts timestamp from a result
func extractTimestamp(result map[string]interface{}) time.Time {
	// Try various timestamp fields
	if ts, ok := result["timestamp"].(float64); ok {
		return time.Unix(int64(ts/1000), 0)
	}
	if ts, ok := result["beginTimeSeconds"].(float64); ok {
		return time.Unix(int64(ts), 0)
	}
	if ts, ok := result["endTimeSeconds"].(float64); ok {
		return time.Unix(int64(ts), 0)
	}
	return time.Now()
}

// extractUniversalFacetValues extracts facet dimension values from a result
func extractUniversalFacetValues(result map[string]interface{}, facetNames []string) map[string]string {
	facetValues := make(map[string]string)

	facetData, hasFacet := result["facet"]
	if !hasFacet {
		return facetValues
	}

	// Handle facet as array
	if facetArray, ok := facetData.([]interface{}); ok {
		for i, value := range facetArray {
			if i < len(facetNames) {
				facetValues[facetNames[i]] = fmt.Sprintf("%v", value)
			} else {
				facetValues[fmt.Sprintf("facet_%d", i)] = fmt.Sprintf("%v", value)
			}
		}
	} else {
		// Handle single facet value
		if len(facetNames) > 0 {
			facetValues[facetNames[0]] = fmt.Sprintf("%v", facetData)
		} else {
			facetValues["facet"] = fmt.Sprintf("%v", facetData)
		}
	}

	return facetValues
}

// extractMetricsFromResult extracts all metric values from a result, flattening composite types
func extractMetricsFromResult(result map[string]interface{}) map[string]float64 {
	metrics := make(map[string]float64)

	for key, value := range result {
		// Skip internal/system fields
		if isInternalField(key) {
			continue
		}

		// Handle different value types
		switch v := value.(type) {
		case float64:
			metrics[key] = v
		case int:
			metrics[key] = float64(v)
		case int64:
			metrics[key] = float64(v)
		case map[string]interface{}:
			// Flatten nested objects (apdex, cdfpercentages, percentiles, etc.)
			flattenObject(key, v, metrics)
		case []interface{}:
			// Arrays (histogram, uniques) are handled by the standard formatter's addDataFields.
			// The universal formatter's metric map is float64-only, so skip here.
			continue
		case string:
			// Try to parse as number
			if num, err := parseNumericString(v); err == nil {
				metrics[key] = num
			}
		}
	}

	return metrics
}

// flattenObject flattens a nested object into separate metric fields
func flattenObject(prefix string, obj map[string]interface{}, metrics map[string]float64) {
	for key, value := range obj {
		fieldName := fmt.Sprintf("%s.%s", prefix, key)

		switch v := value.(type) {
		case float64:
			metrics[fieldName] = v
		case int:
			metrics[fieldName] = float64(v)
		case int64:
			metrics[fieldName] = float64(v)
		case string:
			if num, err := parseNumericString(v); err == nil {
				metrics[fieldName] = num
			}
		case map[string]interface{}:
			// Recursively flatten nested objects
			flattenObject(fieldName, v, metrics)
		}
	}
}

// isInternalField checks if a field is an internal/system field that should be excluded
func isInternalField(key string) bool {
	internalFields := []string{
		"beginTimeSeconds",
		"endTimeSeconds",
		"facet",
		"inspectedCount",
		"timestamp",
	}

	for _, internal := range internalFields {
		if key == internal {
			return true
		}
	}

	return false
}

// extractAliasesFromMetadata extracts function aliases from metadata
// Returns an array of aliases like ["Errors", "Success", "Error %"]
func extractAliasesFromMetadata(resp *UniversalResponse) []string {
	var aliases []string

	// Check if we have timeseries metadata with contents
	if resp.Metadata.Contents.TimeSeries.Contents != nil {
		for _, content := range resp.Metadata.Contents.TimeSeries.Contents {
			if content.Alias != "" {
				aliases = append(aliases, content.Alias)
			}
		}
	}

	log.DefaultLogger.Debug("Extracted aliases from metadata", "aliases", aliases, "count", len(aliases))
	return aliases
}

// extractFacetNamesFromResponse extracts facet names from metadata
func extractFacetNamesFromResponse(resp *UniversalResponse) []string {
	// Check Facets array first (for multi-facet queries)
	if len(resp.Metadata.Facets) > 0 {
		return resp.Metadata.Facets
	}
	// Check Facet field in metadata (for nested facet structure)
	if len(resp.Metadata.Facet) > 0 {
		// Facet can be a string or array
		return resp.Metadata.Facet
	}
	return nil
}

// extractFacetValuesFromName extracts facet values from the name field
// The name can be either a string, an array of strings, or an array with mixed types
func extractFacetValuesFromName(name interface{}, facetNames []string) map[string]string {
	facetValues := make(map[string]string)

	switch v := name.(type) {
	case string:
		// Single string - use first facet name or default
		if len(facetNames) > 0 {
			facetValues[facetNames[0]] = v
		} else {
			facetValues["facet"] = v
		}

	case []interface{}:
		// Array of values - map to facet names
		for i, val := range v {
			var strVal string
			if val == nil {
				strVal = "null"
			} else {
				strVal = fmt.Sprintf("%v", val)
			}

			if i < len(facetNames) {
				facetValues[facetNames[i]] = strVal
			} else {
				facetValues[fmt.Sprintf("facet_%d", i)] = strVal
			}
		}

	case []string:
		// Array of strings
		for i, val := range v {
			if i < len(facetNames) {
				facetValues[facetNames[i]] = val
			} else {
				facetValues[fmt.Sprintf("facet_%d", i)] = val
			}
		}

	default:
		// Fallback
		facetValues["facet"] = fmt.Sprintf("%v", name)
	}

	return facetValues
}

// buildUniversalFrames builds Grafana data frames from normalized data points
func buildUniversalFrames(points []NormalizedDataPoint, facetNames []string, query backend.DataQuery) []*data.Frame {
	// Group points by facet values
	grouped := groupPointsByFacet(points)

	var frames []*data.Frame

	for facetKey, facetPoints := range grouped {
		// Sort points by timestamp
		sortPointsByTime(facetPoints)

		// Extract all unique metric names
		metricNames := extractMetricNames(facetPoints)

		// Create frame for this facet group
		frame := createFrameFromPoints(facetKey, facetPoints, metricNames)
		frames = append(frames, frame)
	}

	log.DefaultLogger.Debug("Built frames from normalized points", "frameCount", len(frames))

	return frames
}

// groupPointsByFacet groups data points by their facet values
func groupPointsByFacet(points []NormalizedDataPoint) map[string][]NormalizedDataPoint {
	grouped := make(map[string][]NormalizedDataPoint)

	for _, point := range points {
		// Create a key from facet values
		key := createFacetKey(point.FacetValues)
		grouped[key] = append(grouped[key], point)
	}

	return grouped
}

// createFacetKey creates a unique key from facet values for grouping
func createFacetKey(facetValues map[string]string) string {
	if len(facetValues) == 0 {
		return ""
	}

	// Sort keys for consistent ordering
	var keys []string
	for k := range facetValues {
		keys = append(keys, k)
	}

	// For single facet, return just the value
	if len(keys) == 1 {
		return facetValues[keys[0]]
	}

	// For multiple facets, create composite key
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, facetValues[k]))
	}
	return strings.Join(parts, ",")
}

// extractMetricNames extracts all unique metric names from points
func extractMetricNames(points []NormalizedDataPoint) []string {
	nameSet := make(map[string]bool)

	for _, point := range points {
		for name := range point.Metrics {
			nameSet[name] = true
		}
	}

	var names []string
	for name := range nameSet {
		names = append(names, name)
	}

	return names
}

// sortPointsByTime sorts data points by timestamp
func sortPointsByTime(points []NormalizedDataPoint) {
	// Simple bubble sort for now
	for i := 0; i < len(points)-1; i++ {
		for j := 0; j < len(points)-i-1; j++ {
			if points[j].Timestamp.After(points[j+1].Timestamp) {
				points[j], points[j+1] = points[j+1], points[j]
			}
		}
	}
}

// createFrameFromPoints creates a data frame from normalized points
func createFrameFromPoints(facetKey string, points []NormalizedDataPoint, metricNames []string) *data.Frame {
	// Use facet key as frame name, or "response" if no facets
	frameName := facetKey
	if frameName == "" {
		frameName = "response"
	}

	frame := data.NewFrame(frameName)

	// Add time field with bucket interval
	times := make([]time.Time, len(points))
	for i, point := range points {
		times[i] = point.Timestamp
	}
	timeField := data.NewField("time", nil, times)
	// Use the interval from the first point that has one
	for _, point := range points {
		if point.IntervalMs > 0 {
			if timeField.Config == nil {
				timeField.Config = &data.FieldConfig{}
			}
			timeField.Config.Interval = point.IntervalMs
			break
		}
	}
	frame.Fields = append(frame.Fields, timeField)

	// Get facet labels from first point
	var labels map[string]string
	if len(points) > 0 && len(points[0].FacetValues) > 0 {
		labels = points[0].FacetValues
	}

	// Add metric fields
	for _, metricName := range metricNames {
		values := make([]*float64, len(points))

		for i, point := range points {
			if value, exists := point.Metrics[metricName]; exists {
				values[i] = &value
			}
			// If metric doesn't exist in this point, leave as nil
		}

		field := data.NewField(metricName, labels, values)
		frame.Fields = append(frame.Fields, field)
	}

	return frame
}
