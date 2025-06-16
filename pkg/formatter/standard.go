// Package formatter provides functionality for formatting New Relic query results
// into Grafana data frames.
package formatter

import (
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/constants"
)

// formatStandardQuery formats standard query results into a Grafana data frame.
func formatStandardQuery(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	// Create a new frame
	frame := data.NewFrame(constants.StandardResponseFrameName)

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
			if fieldName != constants.TimestampFieldName {
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
		if timestamp, ok := result[constants.TimestampFieldName].(float64); ok {
			timePoints[i] = time.Unix(int64(timestamp), 0)
		} else {
			timePoints[i] = query.TimeRange.From
		}
	}
	return data.NewField(constants.TimeFieldName, nil, timePoints)
}

// addDataFields adds data fields to the frame
func addDataFields(frame *data.Frame, results *nrdb.NRDBResultContainer, fieldNames []string) {
	for _, fieldName := range fieldNames {
		values := make([]interface{}, len(results.Results))
		for i, result := range results.Results {
			values[i] = result[fieldName]
		}
		frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))
	}
} 