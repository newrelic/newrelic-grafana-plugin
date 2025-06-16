// Package formatter provides functionality for formatting New Relic query results
// into Grafana data frames.
package formatter

import (
	"fmt"
	"time"

	"newrelic-grafana-plugin/pkg/constant"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
)

// isFacetedCountQuery checks if the results represent a faceted count query
func isFacetedCountQuery(results *nrdb.NRDBResultContainer) bool {
	return len(results.Results) > 0 &&
		results.Results[0][constant.CountFieldName] != nil &&
		results.Results[0][constant.FacetFieldName] != nil
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
