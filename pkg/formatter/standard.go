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

// formatStandardQuery formats standard query results into a Grafana data frame.
func formatStandardQuery(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	// Create a new frame
	frame := data.NewFrame(constant.StandardResponseFrameName)

	// Extract field names from the first result
	fieldNames := extractFieldNames(results)

	// Create time field
	timeField := createTimeField(results, query)
	frame.Fields = append(frame.Fields, timeField)

	// Add data fields
	addDataFields(frame, results, fieldNames)

	resp.Frames = append(resp.Frames, frame)
	return resp
}

// extractFieldNames extracts field names from query results
func extractFieldNames(results *nrdb.NRDBResultContainer) []string {
	fieldNames := []string{}
	if len(results.Results) > 0 {
		for fieldName := range results.Results[0] {
			if fieldName != constant.TimestampFieldName {
				fieldNames = append(fieldNames, fieldName)
			}
		}
	}
	return fieldNames
}

// createTimeField creates a time field from query results
func createTimeField(results *nrdb.NRDBResultContainer, query backend.DataQuery) *data.Field {
	timePoints := make([]time.Time, len(results.Results))
	for i, result := range results.Results {
		if timestamp, ok := result[constant.TimestampFieldName].(float64); ok {
			timePoints[i] = time.Unix(int64(timestamp), 0)
		} else {
			timePoints[i] = query.TimeRange.From
		}
	}
	return data.NewField(constant.TimeFieldName, nil, timePoints)
}

// addDataFields adds data fields to the frame
func addDataFields(frame *data.Frame, results *nrdb.NRDBResultContainer, fieldNames []string) {
	for _, fieldName := range fieldNames {
		values := make([]interface{}, len(results.Results))
		for i, result := range results.Results {
			values[i] = result[fieldName]
		}

		// Convert values to appropriate type
		convertedValues := make([]float64, len(values))
		for i, v := range values {
			switch val := v.(type) {
			case float64:
				convertedValues[i] = val
			case int:
				convertedValues[i] = float64(val)
			case int64:
				convertedValues[i] = float64(val)
			default:
				convertedValues[i] = 0
			}
		}

		frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, convertedValues))
	}

	// Set visualization type
	frame.Meta = &data.FrameMeta{
		PreferredVisualization: data.VisTypeGraph,
	}
}
