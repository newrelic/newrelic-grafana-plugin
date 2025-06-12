package dataformatter

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/newrelic/newrelic-client-go/pkg/nrdb"
)

// IsCountQuery checks if the NRDB result container represents a single count query.
func IsCountQuery(results *nrdb.NRDBResultContainer) bool {
	return len(results.Results) == 1 && results.Results[0]["count"] != nil
}

// FormatCountQueryResults creates a Grafana DataFrame for a New Relic count query result.
// It specifically handles single count values.
func FormatCountQueryResults(results *nrdb.NRDBResultContainer) *backend.DataResponse {
	resp := &backend.DataResponse{}

	count := float64(0)
	if len(results.Results) > 0 {
		if countValue, ok := results.Results[0]["count"].(float64); ok {
			count = countValue
		}
	}

	frame := data.NewFrame("",
		data.NewField("count", nil, []float64{count}),
	)

	resp.Frames = append(resp.Frames, frame)
	return resp
}

// FormatRegularQueryResults creates a Grafana DataFrame from New Relic NRDB query results
// for regular time series or detail queries.
func FormatRegularQueryResults(results *nrdb.NRDBResultContainer, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	if len(results.Results) == 0 {
		return resp
	}

	frame := data.NewFrame("response")

	// Collect all unique field names, excluding "timestamp"
	fieldNamesMap := make(map[string]struct{})
	for _, result := range results.Results {
		for key := range result {
			if key != "timestamp" {
				fieldNamesMap[key] = struct{}{}
			}
		}
	}

	// Convert map keys to a sorted slice for consistent field order
	var fieldNames []string
	for key := range fieldNamesMap {
		fieldNames = append(fieldNames, key)
	}
	// Optionally sort fieldNames for consistent output
	// sort.Strings(fieldNames) // Uncomment if consistent order is important

	// Pre-allocate slice for times
	times := make([]time.Time, len(results.Results))
	for i, result := range results.Results {
		if ts, ok := result["timestamp"].(float64); ok {
			times[i] = time.Unix(int64(ts/1000), 0)
		} else {
			// If no timestamp, use query start time as a fallback
			times[i] = query.TimeRange.From
		}
	}
	frame.Fields = append(frame.Fields, data.NewField("time", nil, times))

	// Dynamically create fields based on detected types
	for _, fieldName := range fieldNames {
		// Determine the common type for the field across all results
		var commonType interface{}
		if len(results.Results) > 0 {
			commonType = results.Results[0][fieldName]
		}

		switch commonType.(type) {
		case float64:
			values := make([]float64, len(results.Results))
			for i, result := range results.Results {
				if val, ok := result[fieldName].(float64); ok {
					values[i] = val
				}
			}
			frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))
		case bool:
			values := make([]bool, len(results.Results))
			for i, result := range results.Results {
				if val, ok := result[fieldName].(bool); ok {
					values[i] = val
				}
			}
			frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))
		case string:
			values := make([]string, len(results.Results))
			for i, result := range results.Results {
				if val, ok := result[fieldName].(string); ok {
					values[i] = val
				} else if result[fieldName] != nil {
					values[i] = fmt.Sprintf("%v", result[fieldName]) // Convert non-string to string
				}
			}
			frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))
		default:
			// Fallback for other types, convert to string
			values := make([]string, len(results.Results))
			for i, result := range results.Results {
				if result[fieldName] != nil {
					values[i] = fmt.Sprintf("%v", result[fieldName])
				}
			}
			frame.Fields = append(frame.Fields, data.NewField(fieldName, nil, values))
		}
	}

	resp.Frames = append(resp.Frames, frame)
	return resp
}
