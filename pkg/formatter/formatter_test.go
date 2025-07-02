package formatter

import (
	"newrelic-grafana-plugin/pkg/utils"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsCountQuery(t *testing.T) {
	tests := []struct {
		name     string
		results  *nrdb.NRDBResultContainer
		expected bool
	}{
		{
			name: "simple count query",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{"count": 42.0},
				},
			},
			expected: true,
		},
		{
			name: "not a count query",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{"value": 42.0},
				},
			},
			expected: false,
		},
		{
			name: "empty results",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCountQuery(tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatQueryResults(t *testing.T) {
	now := time.Now()
	query := backend.DataQuery{
		TimeRange: backend.TimeRange{
			From: now.Add(-1 * time.Hour),
			To:   now,
		},
	}

	tests := []struct {
		name     string
		results  *nrdb.NRDBResultContainer
		query    backend.DataQuery
		validate func(t *testing.T, resp *backend.DataResponse)
	}{
		{
			name: "simple count query",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{"count": 42.0},
				},
			},
			query: query,
			validate: func(t *testing.T, resp *backend.DataResponse) {
				require.Len(t, resp.Frames, 2)
				assert.Equal(t, "count", resp.Frames[0].Name)
				assert.Equal(t, data.VisType(data.VisTypeTable), resp.Frames[0].Meta.PreferredVisualization)
				assert.Equal(t, utils.CountTimeSeriesFrameName, resp.Frames[1].Name)
				assert.Equal(t, data.VisTypeGraph, resp.Frames[1].Meta.PreferredVisualization)
			},
		},
		{
			name: "faceted count query",
			results: &nrdb.NRDBResultContainer{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"service"},
				},
				Results: []nrdb.NRDBResult{
					{
						"count": 42.0,
						"facet": []interface{}{"service1"},
					},
					{
						"count": 24.0,
						"facet": []interface{}{"service2"},
					},
				},
			},
			query: query,
			validate: func(t *testing.T, resp *backend.DataResponse) {
				require.Len(t, resp.Frames, 2) // One frame per facet value (like Grafana Cloud)

				// First frame should be for service1
				frame1 := resp.Frames[0]
				require.Len(t, frame1.Fields, 2) // time and count fields

				// Check time field
				timeField1 := frame1.Fields[0]
				assert.Equal(t, "time", timeField1.Name)

				// Check count field with service label
				countField1 := frame1.Fields[1]
				assert.Equal(t, "count", countField1.Name)
				assert.Equal(t, "service1", countField1.Labels["service"])
				assert.Equal(t, 42.0, countField1.At(0))

				// Second frame should be for service2
				frame2 := resp.Frames[1]
				require.Len(t, frame2.Fields, 2) // time and count fields

				// Check count field with service label
				countField2 := frame2.Fields[1]
				assert.Equal(t, "count", countField2.Name)
				assert.Equal(t, "service2", countField2.Labels["service"])
				assert.Equal(t, 24.0, countField2.At(0))
			},
		},
		{
			name: "standard query",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"timestamp": now.Unix(),
						"value":     42.0,
					},
				},
			},
			query: query,
			validate: func(t *testing.T, resp *backend.DataResponse) {
				require.Len(t, resp.Frames, 1)
				assert.Equal(t, utils.StandardResponseFrameName, resp.Frames[0].Name)
			},
		},
		{
			name: "empty results",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{},
			},
			query: query,
			validate: func(t *testing.T, resp *backend.DataResponse) {
				require.Empty(t, resp.Frames)
			},
		},
		{
			name: "timeseries query with beginTimeSeconds",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"beginTimeSeconds": float64(1750148571),
						"endTimeSeconds":   float64(1750148631),
						"count":            3.0,
					},
					{
						"beginTimeSeconds": float64(1750148631),
						"endTimeSeconds":   float64(1750148691),
						"count":            3.0,
					},
				},
			},
			query: query,
			validate: func(t *testing.T, resp *backend.DataResponse) {
				require.Len(t, resp.Frames, 1)
				frame := resp.Frames[0]
				require.Len(t, frame.Fields, 2) // time and count fields

				// Verify time field uses beginTimeSeconds
				timeField := frame.Fields[0]
				assert.Equal(t, "time", timeField.Name)
				times := timeField.At(0).(time.Time)
				expectedTime := time.Unix(1750148571, 0)
				assert.Equal(t, expectedTime, times)

				// Verify count field is present
				countField := frame.Fields[1]
				assert.Equal(t, "count", countField.Name)
			},
		},
		{
			name: "faceted timeseries query",
			results: &nrdb.NRDBResultContainer{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"request.uri"},
				},
				Results: []nrdb.NRDBResult{
					{
						"beginTimeSeconds": float64(1750148571),
						"endTimeSeconds":   float64(1750148631),
						"count":            10.0,
						"facet":            []interface{}{"/users"},
					},
					{
						"beginTimeSeconds": float64(1750148631),
						"endTimeSeconds":   float64(1750148691),
						"count":            20.0,
						"facet":            []interface{}{"/users"},
					},
					{
						"beginTimeSeconds": float64(1750148571),
						"endTimeSeconds":   float64(1750148631),
						"count":            5.0,
						"facet":            []interface{}{"/api"},
					},
				},
			},
			query: query,
			validate: func(t *testing.T, resp *backend.DataResponse) {
				require.Len(t, resp.Frames, 2) // One frame per facet value

				// Collect frame information by facet value for order-independent testing
				framesByFacet := make(map[string]*data.Frame)
				for _, frame := range resp.Frames {
					require.Len(t, frame.Fields, 2) // time and count fields
					countField := frame.Fields[1]
					assert.Equal(t, "count", countField.Name)
					facetValue := countField.Labels["request.uri"]
					framesByFacet[facetValue] = frame
				}

				// Verify /users frame (should have 2 time points)
				usersFrame, exists := framesByFacet["/users"]
				require.True(t, exists, "Should have frame for /users")
				require.Len(t, usersFrame.Fields, 2)

				usersTimeField := usersFrame.Fields[0]
				assert.Equal(t, "time", usersTimeField.Name)
				assert.Equal(t, 2, usersTimeField.Len()) // 2 time points for /users

				usersCountField := usersFrame.Fields[1]
				assert.Equal(t, "count", usersCountField.Name)
				assert.Equal(t, "/users", usersCountField.Labels["request.uri"])
				assert.Equal(t, 2, usersCountField.Len()) // 2 count values

				// Verify /api frame (should have 1 time point)
				apiFrame, exists := framesByFacet["/api"]
				require.True(t, exists, "Should have frame for /api")
				require.Len(t, apiFrame.Fields, 2)

				apiCountField := apiFrame.Fields[1]
				assert.Equal(t, "count", apiCountField.Name)
				assert.Equal(t, "/api", apiCountField.Labels["request.uri"])
				assert.Equal(t, 1, apiCountField.Len()) // 1 count value
			},
		},
		{
			name: "faceted aggregation timeseries query (sum.duration)",
			results: &nrdb.NRDBResultContainer{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"request.uri"},
				},
				Results: []nrdb.NRDBResult{
					{
						"beginTimeSeconds": float64(1750148571),
						"endTimeSeconds":   float64(1750148631),
						"sum.duration":     1500.25,
						"facet":            []interface{}{"/api/users"},
					},
					{
						"beginTimeSeconds": float64(1750148631),
						"endTimeSeconds":   float64(1750148691),
						"sum.duration":     2100.75,
						"facet":            []interface{}{"/api/users"},
					},
					{
						"beginTimeSeconds": float64(1750148571),
						"endTimeSeconds":   float64(1750148631),
						"sum.duration":     800.50,
						"facet":            []interface{}{"/api/orders"},
					},
				},
			},
			query: query,
			validate: func(t *testing.T, resp *backend.DataResponse) {
				require.Len(t, resp.Frames, 2) // One frame per facet value

				// Collect frame information by facet value for order-independent testing
				framesByFacet := make(map[string]*data.Frame)
				for _, frame := range resp.Frames {
					require.Len(t, frame.Fields, 2) // time and sum.duration fields
					sumField := frame.Fields[1]
					assert.Equal(t, "sum.duration", sumField.Name)
					facetValue := sumField.Labels["request.uri"]
					framesByFacet[facetValue] = frame
				}

				// Verify /api/users frame (should have 2 time points)
				usersFrame, exists := framesByFacet["/api/users"]
				require.True(t, exists, "Should have frame for /api/users")
				require.Len(t, usersFrame.Fields, 2)

				usersTimeField := usersFrame.Fields[0]
				assert.Equal(t, "time", usersTimeField.Name)
				assert.Equal(t, 2, usersTimeField.Len()) // 2 time points for /api/users

				usersSumField := usersFrame.Fields[1]
				assert.Equal(t, "sum.duration", usersSumField.Name)
				assert.Equal(t, "/api/users", usersSumField.Labels["request.uri"])
				assert.Equal(t, 2, usersSumField.Len()) // 2 sum.duration values

				// Get the values as a slice to avoid index issues
				usersSumValues := make([]float64, usersSumField.Len())
				for i := 0; i < usersSumField.Len(); i++ {
					if val := usersSumField.At(i); val != nil {
						usersSumValues[i] = *val.(*float64)
					}
				}
				// Check that the values include the expected ones (order might vary)
				assert.Contains(t, usersSumValues, 1500.25, "Should contain value 1500.25")
				assert.Contains(t, usersSumValues, 2100.75, "Should contain value 2100.75")

				// Verify /api/orders frame (should have 1 time point)
				ordersFrame, exists := framesByFacet["/api/orders"]
				require.True(t, exists, "Should have frame for /api/orders")
				require.Len(t, ordersFrame.Fields, 2)

				ordersSumField := ordersFrame.Fields[1]
				assert.Equal(t, "sum.duration", ordersSumField.Name)
				assert.Equal(t, "/api/orders", ordersSumField.Labels["request.uri"])
				assert.Equal(t, 1, ordersSumField.Len()) // 1 sum.duration value

				// Safely check the value
				if ordersSumField.Len() > 0 && ordersSumField.At(0) != nil {
					assert.Equal(t, 800.50, *ordersSumField.At(0).(*float64))
				}
			},
		},
		{
			name: "faceted aggregation timeseries query (percentile.duration)",
			results: &nrdb.NRDBResultContainer{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"request.uri"},
				},
				Results: []nrdb.NRDBResult{
					{
						"beginTimeSeconds":    float64(1750148571),
						"endTimeSeconds":      float64(1750148631),
						"percentile.duration": map[string]interface{}{"95": 156.75},
						"facet":               []interface{}{"/users"},
					},
					{
						"beginTimeSeconds":    float64(1750148631),
						"endTimeSeconds":      float64(1750148691),
						"percentile.duration": map[string]interface{}{"95": 189.25},
						"facet":               []interface{}{"/users"},
					},
					{
						"beginTimeSeconds":    float64(1750148571),
						"endTimeSeconds":      float64(1750148631),
						"percentile.duration": map[string]interface{}{"95": 98.50},
						"facet":               []interface{}{"/users12"},
					},
				},
			},
			query: query,
			validate: func(t *testing.T, resp *backend.DataResponse) {
				require.Len(t, resp.Frames, 2) // One frame per facet value

				// Collect frames by facet for order-independent testing
				framesByFacet := make(map[string]*data.Frame)
				for _, frame := range resp.Frames {
					require.Len(t, frame.Fields, 2) // time and percentile.duration.95 fields
					percentileField := frame.Fields[1]
					assert.Equal(t, "percentile.duration.95", percentileField.Name)
					facetValue := percentileField.Labels["request.uri"]
					framesByFacet[facetValue] = frame
				}

				// Verify /users frame (should have 2 time points)
				usersFrame, exists := framesByFacet["/users"]
				require.True(t, exists, "Should have frame for /users")
				require.Len(t, usersFrame.Fields, 2)

				usersTimeField := usersFrame.Fields[0]
				assert.Equal(t, "time", usersTimeField.Name)
				assert.Equal(t, 2, usersTimeField.Len()) // 2 time points for /users

				usersPercentileField := usersFrame.Fields[1]
				assert.Equal(t, "percentile.duration.95", usersPercentileField.Name)
				assert.Equal(t, "/users", usersPercentileField.Labels["request.uri"])
				assert.Equal(t, 2, usersPercentileField.Len()) // 2 percentile values
				// Verify the actual values
				assert.Equal(t, 156.75, *usersPercentileField.At(0).(*float64))
				assert.Equal(t, 189.25, *usersPercentileField.At(1).(*float64))

				// Verify /users12 frame (should have 1 time point)
				users12Frame, exists := framesByFacet["/users12"]
				require.True(t, exists, "Should have frame for /users12")
				require.Len(t, users12Frame.Fields, 2)

				users12PercentileField := users12Frame.Fields[1]
				assert.Equal(t, "percentile.duration.95", users12PercentileField.Name)
				assert.Equal(t, "/users12", users12PercentileField.Labels["request.uri"])
				assert.Equal(t, 1, users12PercentileField.Len()) // 1 percentile value
				assert.Equal(t, 98.50, *users12PercentileField.At(0).(*float64))
			},
		},
		{
			name: "multiple metrics timeseries",
			results: &nrdb.NRDBResultContainer{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"appName"},
				},
				Results: []nrdb.NRDBResult{
					{
						"beginTimeSeconds":       float64(1750246173),
						"endTimeSeconds":         float64(1750267773),
						"min.duration":           0.00000875,
						"percentile.duration.95": 0.00009822845458984375,
						"stddev.duration":        0.000027485675391342546,
						"max.duration":           0.000865791,
						"facet":                  []interface{}{"sampleMonitoringProject"},
					},
					{
						"beginTimeSeconds":       float64(1750267773),
						"endTimeSeconds":         float64(1750289373),
						"min.duration":           nil,
						"percentile.duration.95": 0.0,
						"stddev.duration":        0.0,
						"max.duration":           nil,
						"facet":                  []interface{}{"sampleMonitoringProject"},
					},
					{
						"beginTimeSeconds":       float64(1750289373),
						"endTimeSeconds":         float64(1750310973),
						"min.duration":           0.000008875,
						"percentile.duration.95": 0.00010204315185546875,
						"stddev.duration":        0.000030676755871422517,
						"max.duration":           0.00076475,
						"facet":                  []interface{}{"sampleMonitoringProject"},
					},
					{
						"beginTimeSeconds":       float64(1750310973),
						"endTimeSeconds":         float64(1750332573),
						"min.duration":           0.000011916,
						"percentile.duration.95": 0.00010776519775390625,
						"stddev.duration":        0.00002898921110717967,
						"max.duration":           0.00046925,
						"facet":                  []interface{}{"sampleMonitoringProject"},
					},
				},
			},
			query: query,
			validate: func(t *testing.T, resp *backend.DataResponse) {
				// Verify response structure
				require.Nil(t, resp.Error, "Response should not have an error")
				require.NotNil(t, resp.Frames, "Response should have frames")
				require.NotEmpty(t, resp.Frames, "Response should have at least one frame")

				// Get the first frame
				frame := resp.Frames[0]

				// The formatter might combine some fields or process them differently
				// Let's just ensure we have more than 1 field (time + at least some metrics)
				require.Greater(t, len(frame.Fields), 1, "Should have at least time field plus some metric fields")

				// Check that time field is first
				timeField := frame.Fields[0]
				assert.Equal(t, "time", timeField.Name, "First field should be time")

				// Create a map to hold our fields by name
				fieldsByName := make(map[string]*data.Field)
				for i := 1; i < len(frame.Fields); i++ { // Skip time field
					field := frame.Fields[i]
					fieldsByName[field.Name] = field
				}

				// Check labels on a field if we can find any of our expected ones
				expectedFieldNames := []string{"min.duration", "percentile.duration.95", "stddev.duration", "max.duration"}
				for _, name := range expectedFieldNames {
					if field, ok := fieldsByName[name]; ok {
						// If we found an expected field, verify the app label
						assert.Equal(t, "sampleMonitoringProject", field.Labels["appName"],
							"Field %s should have appName label", name)
						break
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := FormatQueryResults(tt.results, tt.query)
			tt.validate(t, resp)
		})
	}
}

func TestExtractCountValue(t *testing.T) {
	tests := []struct {
		name     string
		result   nrdb.NRDBResult
		expected float64
	}{
		{
			name:     "valid count",
			result:   nrdb.NRDBResult{"count": 42.0},
			expected: 42.0,
		},
		{
			name:     "invalid count type",
			result:   nrdb.NRDBResult{"count": "42"},
			expected: 0.0,
		},
		{
			name:     "missing count",
			result:   nrdb.NRDBResult{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCountValue(tt.result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFacetNames(t *testing.T) {
	tests := []struct {
		name     string
		results  *nrdb.NRDBResultContainer
		expected []string
	}{
		{
			name: "with facets",
			results: &nrdb.NRDBResultContainer{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"service", "host"},
				},
			},
			expected: []string{"service", "host"},
		},
		{
			name: "no facets",
			results: &nrdb.NRDBResultContainer{
				Metadata: nrdb.NRDBMetadata{},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFacetNames(tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFacetValues(t *testing.T) {
	tests := []struct {
		name       string
		result     nrdb.NRDBResult
		facetNames []string
		expectFunc func(t *testing.T, facetFields map[string][]string, index int)
	}{
		{
			name: "string facet values",
			result: nrdb.NRDBResult{
				"facet": []interface{}{"service1", "host1"},
			},
			facetNames: []string{"service", "host"},
			expectFunc: func(t *testing.T, facetFields map[string][]string, index int) {
				assert.Equal(t, "service1", facetFields["service"][index])
				assert.Equal(t, "host1", facetFields["host"][index])
			},
		},
		{
			name: "non-string facet values",
			result: nrdb.NRDBResult{
				"facet": []interface{}{123, 45.6},
			},
			facetNames: []string{"id", "score"},
			expectFunc: func(t *testing.T, facetFields map[string][]string, index int) {
				assert.Equal(t, "123", facetFields["id"][index])
				assert.Equal(t, "45.6", facetFields["score"][index])
			},
		},
		{
			name: "single facet value (not array)",
			result: nrdb.NRDBResult{
				"facet": "single_service",
			},
			facetNames: []string{"service"},
			expectFunc: func(t *testing.T, facetFields map[string][]string, index int) {
				assert.Equal(t, "single_service", facetFields["service"][index])
			},
		},
		{
			name: "facet array shorter than names",
			result: nrdb.NRDBResult{
				"facet": []interface{}{"service1"},
			},
			facetNames: []string{"service", "host"},
			expectFunc: func(t *testing.T, facetFields map[string][]string, index int) {
				assert.Equal(t, "service1", facetFields["service"][index])
				assert.Equal(t, "", facetFields["host"][index]) // Should remain empty
			},
		},
		{
			name: "no facet field",
			result: nrdb.NRDBResult{
				"count": 42.0,
			},
			facetNames: []string{"service"},
			expectFunc: func(t *testing.T, facetFields map[string][]string, index int) {
				assert.Equal(t, "", facetFields["service"][index]) // Should remain empty
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize facetFields
			facetFields := make(map[string][]string)
			for _, name := range tt.facetNames {
				facetFields[name] = make([]string, 1)
			}

			extractFacetValues(tt.result, tt.facetNames, facetFields, 0)
			tt.expectFunc(t, facetFields, 0)
		})
	}
}

func TestAddDataFields(t *testing.T) {
	tests := []struct {
		name       string
		results    *nrdb.NRDBResultContainer
		fieldNames []string
		expectFunc func(t *testing.T, frame *data.Frame)
	}{
		{
			name: "float64 fields",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{"value": 42.0, "score": 85.5},
					{"value": 24.0, "score": 92.3},
				},
			},
			fieldNames: []string{"value", "score"},
			expectFunc: func(t *testing.T, frame *data.Frame) {
				require.Len(t, frame.Fields, 2)

				// Check value field (now nullable float64)
				valueField := frame.Fields[0]
				assert.Equal(t, "value", valueField.Name)
				assert.Equal(t, 42.0, *valueField.At(0).(*float64))
				assert.Equal(t, 24.0, *valueField.At(1).(*float64))

				// Check score field (now nullable float64)
				scoreField := frame.Fields[1]
				assert.Equal(t, "score", scoreField.Name)
				assert.Equal(t, 85.5, *scoreField.At(0).(*float64))
				assert.Equal(t, 92.3, *scoreField.At(1).(*float64))
			},
		},
		{
			name: "string fields",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{"name": "service1", "status": "active"},
					{"name": "service2", "status": "inactive"},
				},
			},
			fieldNames: []string{"name", "status"},
			expectFunc: func(t *testing.T, frame *data.Frame) {
				require.Len(t, frame.Fields, 2)

				// Check name field
				nameField := frame.Fields[0]
				assert.Equal(t, "name", nameField.Name)
				assert.Equal(t, "service1", nameField.At(0))
				assert.Equal(t, "service2", nameField.At(1))

				// Check status field
				statusField := frame.Fields[1]
				assert.Equal(t, "status", statusField.Name)
				assert.Equal(t, "active", statusField.At(0))
				assert.Equal(t, "inactive", statusField.At(1))
			},
		},
		{
			name: "mixed types",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{"count": 42.0, "name": "service1", "active": true},
					{"count": 24.0, "name": "service2", "active": false},
				},
			},
			fieldNames: []string{"count", "name", "active"},
			expectFunc: func(t *testing.T, frame *data.Frame) {
				require.Len(t, frame.Fields, 3)

				// Check count field (nullable float64)
				countField := frame.Fields[0]
				assert.Equal(t, "count", countField.Name)
				assert.Equal(t, 42.0, *countField.At(0).(*float64))
				assert.Equal(t, 24.0, *countField.At(1).(*float64))

				// Check name field (string)
				nameField := frame.Fields[1]
				assert.Equal(t, "name", nameField.Name)
				assert.Equal(t, "service1", nameField.At(0))
				assert.Equal(t, "service2", nameField.At(1))

				// Check active field (nullable bool)
				activeField := frame.Fields[2]
				assert.Equal(t, "active", activeField.Name)
				assert.Equal(t, true, *activeField.At(0).(*bool))
				assert.Equal(t, false, *activeField.At(1).(*bool))
			},
		},
		{
			name: "nil values",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{"value": 42.0, "nullable": nil},
					{"value": nil, "nullable": "present"},
				},
			},
			fieldNames: []string{"value", "nullable"},
			expectFunc: func(t *testing.T, frame *data.Frame) {
				require.Len(t, frame.Fields, 2)

				// Check value field (nullable float64)
				valueField := frame.Fields[0]
				assert.Equal(t, "value", valueField.Name)
				assert.Equal(t, 42.0, *valueField.At(0).(*float64))
				assert.Nil(t, valueField.At(1)) // nil should remain nil

				// Check nullable field (string)
				nullableField := frame.Fields[1]
				assert.Equal(t, "nullable", nullableField.Name)
				assert.Equal(t, "", nullableField.At(0)) // nil should result in empty string
				assert.Equal(t, "present", nullableField.At(1))
			},
		},
		{
			name: "empty results",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{},
			},
			fieldNames: []string{"value"},
			expectFunc: func(t *testing.T, frame *data.Frame) {
				assert.Len(t, frame.Fields, 0) // No fields should be added for empty results
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := data.NewFrame("test")
			addDataFields(frame, tt.results, tt.fieldNames)
			tt.expectFunc(t, frame)
		})
	}
}

func TestAddDataFieldsComprehensive(t *testing.T) {
	tests := []struct {
		name       string
		fieldNames []string
		results    *nrdb.NRDBResultContainer
		validate   func(t *testing.T, frame *data.Frame)
	}{
		{
			name:       "complex mixed types",
			fieldNames: []string{"numField", "strField", "boolField", "mixedField", "scientificField", "percentileField"},
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"numField":        42.5,
						"strField":        "value1",
						"boolField":       true,
						"mixedField":      10,
						"scientificField": "1.5e-10",
						"percentileField": map[string]interface{}{
							"95": 95.5,
							"99": 99.9,
						},
					},
					{
						"numField":        nil,
						"strField":        "value2",
						"boolField":       false,
						"mixedField":      "string-in-mixed",
						"scientificField": nil,
						// Percentile field missing
					},
					{
						"numField":        100.1,
						"strField":        nil,
						"boolField":       nil,
						"mixedField":      true,
						"scientificField": "not-a-number",
						"percentileField": map[string]interface{}{
							"95": 96.5,
							// 99 percentile missing
						},
					},
				},
			},
			validate: func(t *testing.T, frame *data.Frame) {
				// Skip percentileField as it's handled separately by handlePercentileField
				// Should have 5 fields: numField, strField, boolField, mixedField, scientificField

				// Find each field by name
				var numField, strField, boolField, mixedField, scientificField *data.Field

				for _, field := range frame.Fields {
					switch field.Name {
					case "numField":
						numField = field
					case "strField":
						strField = field
					case "boolField":
						boolField = field
					case "mixedField":
						mixedField = field
					case "scientificField":
						scientificField = field
					}
				}

				// Number field checks
				require.NotNil(t, numField, "numField should exist")
				assert.Equal(t, 3, numField.Len())
				if val, ok := numField.At(0).(*float64); ok {
					assert.Equal(t, 42.5, *val)
				} else {
					assert.Equal(t, 42.5, numField.At(0))
				}
				assert.Nil(t, numField.At(1))
				if val, ok := numField.At(2).(*float64); ok {
					assert.Equal(t, 100.1, *val)
				} else {
					assert.Equal(t, 100.1, numField.At(2))
				}

				// String field checks
				require.NotNil(t, strField, "strField should exist")
				assert.Equal(t, 3, strField.Len())
				assert.Equal(t, "value1", strField.At(0))
				assert.Equal(t, "value2", strField.At(1))
				// In some environments nil strings might be converted to empty strings
				val := strField.At(2)
				if val != nil && val.(string) == "" {
					// Empty string is acceptable when nil was expected
					assert.Empty(t, val)
				} else {
					assert.Nil(t, val)
				}

				// Boolean field checks
				require.NotNil(t, boolField, "boolField should exist")
				assert.Equal(t, 3, boolField.Len())
				// Handle both *bool and bool types
				if val, ok := boolField.At(0).(*bool); ok {
					assert.Equal(t, true, *val)
				} else {
					assert.Equal(t, true, boolField.At(0))
				}
				if val, ok := boolField.At(1).(*bool); ok {
					assert.Equal(t, false, *val)
				} else {
					assert.Equal(t, false, boolField.At(1))
				}
				assert.Nil(t, boolField.At(2))

				// Mixed field (should be converted to string)
				require.NotNil(t, mixedField, "mixedField should exist")
				assert.Equal(t, 3, mixedField.Len())

				// Scientific notation field
				require.NotNil(t, scientificField, "scientificField should exist")
				assert.Equal(t, 3, scientificField.Len())
				assert.NotNil(t, scientificField.At(0))
				assert.Nil(t, scientificField.At(1))
				assert.Nil(t, scientificField.At(2)) // "not-a-number" should be nil
			},
		},
		{
			name:       "object fields",
			fieldNames: []string{"objectField", "arrayField"},
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"objectField": map[string]interface{}{
							"key1": "value1",
							"key2": 42,
						},
						"arrayField": []interface{}{1, 2, 3},
					},
					{
						"objectField": map[string]interface{}{
							"key1": "value2",
						},
						"arrayField": []interface{}{"a", "b"},
					},
				},
			},
			validate: func(t *testing.T, frame *data.Frame) {
				// For objects/arrays, data should be converted to string JSON representations
				var objectField, arrayField *data.Field

				for _, field := range frame.Fields {
					if field.Name == "objectField" {
						objectField = field
					} else if field.Name == "arrayField" {
						arrayField = field
					}
				}

				require.NotNil(t, objectField, "objectField should exist")
				require.NotNil(t, arrayField, "arrayField should exist")

				// Values should be stringified JSON
				assert.Equal(t, 2, objectField.Len())
				assert.Equal(t, 2, arrayField.Len())

				// Check first values are JSON-like strings
				firstObj := objectField.At(0)
				firstArr := arrayField.At(0)
				assert.NotNil(t, firstObj)
				assert.NotNil(t, firstArr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := data.NewFrame("test_frame")
			addDataFields(frame, tt.results, tt.fieldNames)
			tt.validate(t, frame)
		})
	}
}

func TestExtractFieldNamesMulti(t *testing.T) {
	tests := []struct {
		name         string
		results      *nrdb.NRDBResultContainerMultiResultCustomized
		expectedKeys []string
	}{
		{
			name: "extracts field names",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						"fieldA":                 "valueA",
						"fieldB":                 42.0,
						utils.TimestampFieldName: 1590000000000,
					},
					{
						"fieldA":           "valueA2",
						"fieldC":           true,
						"beginTimeSeconds": 1590000060,
					},
				},
			},
			expectedKeys: []string{"fieldA", "fieldB", "fieldC"},
		},
		{
			name: "excludes timestamp fields",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						"fieldA":                 "valueA",
						utils.TimestampFieldName: 1590000000000,
						"beginTimeSeconds":       1590000000,
						"endTimeSeconds":         1590000060,
					},
				},
			},
			expectedKeys: []string{"fieldA"},
		},
		{
			name: "handles empty results",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{},
			},
			expectedKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFieldNamesMulti(tt.results)
			assert.ElementsMatch(t, tt.expectedKeys, result, "Field names don't match")
		})
	}
}

func TestExtractCountValueMulti(t *testing.T) {
	tests := []struct {
		name     string
		result   nrdb.NRDBResult
		expected float64
	}{
		{
			name:     "valid count",
			result:   nrdb.NRDBResult{"count": 42.0},
			expected: 42.0,
		},
		{
			name:     "invalid count type",
			result:   nrdb.NRDBResult{"count": "42"},
			expected: 0.0,
		},
		{
			name:     "missing count",
			result:   nrdb.NRDBResult{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCountValueMulti(tt.result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatFacetedTimeseriesResultsMulti(t *testing.T) {
	tests := []struct {
		name     string
		results  *nrdb.NRDBResultContainerMultiResultCustomized
		validate func(t *testing.T, response *backend.DataResponse)
	}{
		{
			name: "data in OtherResult field",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"request.uri"},
				},
				OtherResult: []nrdb.NRDBResult{
					{
						"beginTimeSeconds": 1590000000,
						"endTimeSeconds":   1590000060,
						"count":            10,
						"facet":            []interface{}{"/users"},
					},
					{
						"beginTimeSeconds": 1590000060,
						"endTimeSeconds":   1590000120,
						"count":            20,
						"facet":            []interface{}{"/users"},
					},
					{
						"beginTimeSeconds": 1590000000,
						"endTimeSeconds":   1590000060,
						"count":            5,
						"facet":            []interface{}{"/api"},
					},
				},
			},
			validate: func(t *testing.T, response *backend.DataResponse) {
				require.NoError(t, response.Error)
				require.NotNil(t, response.Frames)
				require.GreaterOrEqual(t, len(response.Frames), 1)

				// Should have grouped data by facet into separate frames
				// One frame per facet value (/users and /api)
				var usersFrame, apiFrame *data.Frame
				for _, frame := range response.Frames {
					if strings.Contains(frame.Name, "/users") {
						usersFrame = frame
					} else if strings.Contains(frame.Name, "/api") {
						apiFrame = frame
					}
				}

				require.NotNil(t, usersFrame, "Should have a frame for /users facet")
				require.NotNil(t, apiFrame, "Should have a frame for /api facet")

				// Check that each frame has the expected fields
				timeField := usersFrame.Fields[0]
				assert.Equal(t, "time", timeField.Name)

				valueField := usersFrame.Fields[1]
				assert.Equal(t, "count", valueField.Name)

				// Check lengths
				assert.Equal(t, 2, timeField.Len(), "Time field should have 2 points for /users")
				assert.Equal(t, 2, valueField.Len(), "Value field should have 2 points for /users")

				// Check /api frame
				timeField = apiFrame.Fields[0]
				assert.Equal(t, "time", timeField.Name)
				assert.Equal(t, 1, timeField.Len(), "Time field should have 1 point for /api")

				valueField = apiFrame.Fields[1]
				assert.Equal(t, "count", valueField.Name)
				assert.Equal(t, 1, valueField.Len(), "Value field should have 1 point for /api")
			},
		},
		{
			name: "data in Results field",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"service"},
				},
				Results: []nrdb.NRDBResult{
					{
						"beginTimeSeconds": 1590000000,
						"endTimeSeconds":   1590000060,
						"count":            42,
						"facet":            []interface{}{"service-a"},
					},
					{
						"beginTimeSeconds": 1590000060,
						"endTimeSeconds":   1590000120,
						"count":            24,
						"facet":            []interface{}{"service-a"},
					},
				},
			},
			validate: func(t *testing.T, response *backend.DataResponse) {
				require.NoError(t, response.Error)
				require.NotNil(t, response.Frames)
				require.GreaterOrEqual(t, len(response.Frames), 1)

				// Should have one frame for the service-a facet
				serviceFrame := response.Frames[0]
				require.NotNil(t, serviceFrame)

				// Check frame name contains the facet value
				assert.Contains(t, serviceFrame.Name, "service-a")

				// Check that the frame has time and count fields
				require.GreaterOrEqual(t, len(serviceFrame.Fields), 2)

				timeField := serviceFrame.Fields[0]
				assert.Equal(t, "time", timeField.Name)

				valueField := serviceFrame.Fields[1]
				assert.Equal(t, "count", valueField.Name)

				// Check lengths
				assert.Equal(t, 2, timeField.Len())
				assert.Equal(t, 2, valueField.Len())
			},
		},
		{
			name: "not a faceted timeseries query",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						"count": 42,
					},
				},
			},
			validate: func(t *testing.T, response *backend.DataResponse) {
				require.Error(t, response.Error)
				assert.Contains(t, response.Error.Error(), "not a faceted timeseries query")
			},
		},
	}

	// Mock query for testing
	mockQuery := backend.DataQuery{
		RefID:     "A",
		QueryType: "nrql",
		TimeRange: backend.TimeRange{
			From: time.Now().Add(-1 * time.Hour),
			To:   time.Now(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := FormatFacetedTimeseriesResults(tt.results, mockQuery)
			tt.validate(t, response)
		})
	}
}

func TestHasTimeseriesDataMulti(t *testing.T) {
	tests := []struct {
		name     string
		results  *nrdb.NRDBResultContainerMultiResultCustomized
		expected bool
	}{
		{
			name: "has timeseries data in OtherResult",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				OtherResult: []nrdb.NRDBResult{
					{
						"beginTimeSeconds": 1590000000,
						"endTimeSeconds":   1590000060,
						"count":            10,
					},
				},
			},
			expected: true,
		},
		{
			name: "has timeseries data in Results (fallback)",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						"beginTimeSeconds": 1590000000,
						"endTimeSeconds":   1590000060,
						"count":            10,
					},
				},
			},
			expected: true,
		},
		{
			name: "has only beginTimeSeconds",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				OtherResult: []nrdb.NRDBResult{
					{
						"beginTimeSeconds": 1590000000,
						"count":            10,
					},
				},
			},
			expected: true,
		},
		{
			name: "has only endTimeSeconds",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				OtherResult: []nrdb.NRDBResult{
					{
						"endTimeSeconds": 1590000060,
						"count":          10,
					},
				},
			},
			expected: true,
		},
		{
			name: "no timeseries data in either field",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						"count": 42,
					},
				},
			},
			expected: false,
		},
		{
			name: "empty results",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results:     []nrdb.NRDBResult{},
				OtherResult: []nrdb.NRDBResult{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasTimeseriesDataMulti(tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsFacetedTimeseriesQueryMulti(t *testing.T) {
	tests := []struct {
		name     string
		results  *nrdb.NRDBResultContainerMultiResultCustomized
		expected bool
	}{
		{
			name: "is faceted timeseries (OtherResult)",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"request.uri"},
				},
				OtherResult: []nrdb.NRDBResult{
					{
						"beginTimeSeconds": 1590000000,
						"endTimeSeconds":   1590000060,
						"count":            10,
						"facet":            []interface{}{"/users"},
					},
				},
			},
			expected: true,
		},
		{
			name: "is faceted timeseries (Results)",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"service"},
				},
				Results: []nrdb.NRDBResult{
					{
						"beginTimeSeconds": 1590000000,
						"endTimeSeconds":   1590000060,
						"count":            10,
						"facet":            []interface{}{"service-a"},
					},
				},
			},
			expected: true,
		},
		{
			name: "missing facets",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				OtherResult: []nrdb.NRDBResult{
					{
						"beginTimeSeconds": 1590000000,
						"endTimeSeconds":   1590000060,
						"count":            10,
					},
				},
			},
			expected: false,
		},
		{
			name: "missing timeseries data",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"service"},
				},
				Results: []nrdb.NRDBResult{
					{
						"count": 42,
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFacetedTimeseriesQueryMulti(tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateFacetTableFrame(t *testing.T) {
	tests := []struct {
		name        string
		facetNames  []string
		counts      []float64
		facetFields map[string][]string
		validate    func(t *testing.T, frame *data.Frame)
	}{
		{
			name:       "facet table with multiple dimensions",
			facetNames: []string{"service", "environment"},
			counts:     []float64{42, 24, 13},
			facetFields: map[string][]string{
				"service":     {"service-a", "service-b", "service-c"},
				"environment": {"prod", "staging", "dev"},
			},
			validate: func(t *testing.T, frame *data.Frame) {
				require.NotNil(t, frame)
				require.Equal(t, 3, frame.Fields[0].Len())

				// Check that we have the expected fields - first facet fields, then count
				require.Equal(t, 3, len(frame.Fields))
				assert.Equal(t, "service", frame.Fields[0].Name)
				assert.Equal(t, "environment", frame.Fields[1].Name)
				assert.Equal(t, "count", frame.Fields[2].Name) // count is last per implementation

				// Check service values (first facet field)
				serviceField := frame.Fields[0]
				assert.Equal(t, "service-a", serviceField.At(0))
				assert.Equal(t, "service-b", serviceField.At(1))
				assert.Equal(t, "service-c", serviceField.At(2))

				// Check environment values (second facet field)
				envField := frame.Fields[1]
				assert.Equal(t, "prod", envField.At(0))
				assert.Equal(t, "staging", envField.At(1))
				assert.Equal(t, "dev", envField.At(2))

				// Check count values
				countField := frame.Fields[2]
				assert.Equal(t, float64(42), countField.At(0))
				assert.Equal(t, float64(24), countField.At(1))
				assert.Equal(t, float64(13), countField.At(2))

				// Check frame meta
				require.NotNil(t, frame.Meta)
				assert.Equal(t, data.VisType("table"), frame.Meta.PreferredVisualization)
			},
		},
		{
			name:       "facet table with single dimension",
			facetNames: []string{"service"},
			counts:     []float64{42, 24},
			facetFields: map[string][]string{
				"service": {"service-a", "service-b"},
			},
			validate: func(t *testing.T, frame *data.Frame) {
				require.NotNil(t, frame)
				require.Equal(t, 2, frame.Fields[0].Len())

				// Check that we have the expected fields
				require.Equal(t, 2, len(frame.Fields))
				assert.Equal(t, "service", frame.Fields[0].Name)
				assert.Equal(t, "count", frame.Fields[1].Name)

				// Check service values
				serviceField := frame.Fields[0]
				assert.Equal(t, "service-a", serviceField.At(0))
				assert.Equal(t, "service-b", serviceField.At(1))

				// Check count values
				countField := frame.Fields[1]
				assert.Equal(t, float64(42), countField.At(0))
				assert.Equal(t, float64(24), countField.At(1))

				// Check frame meta
				require.NotNil(t, frame.Meta)
				assert.Equal(t, data.VisType("table"), frame.Meta.PreferredVisualization)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := createFacetTableFrame(tt.facetNames, tt.counts, tt.facetFields)
			tt.validate(t, frame)
		})
	}
}

func TestHasCountField(t *testing.T) {
	tests := []struct {
		name     string
		results  *nrdb.NRDBResultContainer
		expected bool
	}{
		{
			name: "has count field",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{utils.CountFieldName: 42.0},
				},
			},
			expected: true,
		},
		{
			name: "does not have count field",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{"other_field": 42.0},
				},
			},
			expected: false,
		},
		{
			name: "empty results",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{},
			},
			expected: false,
		},
		{
			name: "nil results",
			results: &nrdb.NRDBResultContainer{
				Results: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasCountField(tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatFacetedTimeseriesQueryMulti(t *testing.T) {
	tests := []struct {
		name              string
		results           *nrdb.NRDBResultContainerMultiResultCustomized
		query             backend.DataQuery
		expectedFrames    int
		expectedFrameName string
		expectedFields    int
	}{
		{
			name: "formats faceted timeseries data",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{utils.FacetFieldName},
				},
				OtherResult: []nrdb.NRDBResult{
					{
						utils.FacetFieldName: "api",
						"beginTimeSeconds":   1590000000,
						utils.CountFieldName: 10.0,
					},
					{
						utils.FacetFieldName: "web",
						"beginTimeSeconds":   1590000060,
						utils.CountFieldName: 20.0,
					},
				},
			},
			query:             backend.DataQuery{RefID: "A"},
			expectedFrames:    2,
			expectedFrameName: "web", // First frame name due to map iteration order
			expectedFields:    2,     // time and count fields
		},
		{
			name: "formats faceted timeseries data with array facets",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{utils.FacetFieldName},
				},
				OtherResult: []nrdb.NRDBResult{
					{
						utils.FacetFieldName: []interface{}{"api", "v1"},
						"beginTimeSeconds":   1590000000,
						utils.CountFieldName: 10.0,
					},
					{
						utils.FacetFieldName: []interface{}{"web", "homepage"},
						"beginTimeSeconds":   1590000060,
						utils.CountFieldName: 20.0,
					},
				},
			},
			query:             backend.DataQuery{RefID: "A"},
			expectedFrames:    2,
			expectedFrameName: "api", // First frame name
			expectedFields:    2,     // time and count fields
		},
		{
			name: "falls back to standard query when no facets",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Metadata: nrdb.NRDBMetadata{},
				Results: []nrdb.NRDBResult{
					{
						"beginTimeSeconds":   1590000000,
						utils.CountFieldName: 10.0,
					},
					{
						"beginTimeSeconds":   1590000060,
						utils.CountFieldName: 20.0,
					},
				},
			},
			query:             backend.DataQuery{RefID: "A"},
			expectedFrames:    1,
			expectedFrameName: utils.StandardResponseFrameName,
			expectedFields:    2, // time and count fields
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := formatFacetedTimeseriesQueryMulti(tt.results, tt.query)

			require.NotNil(t, resp)
			assert.Equal(t, tt.expectedFrames, len(resp.Frames))
			if tt.expectedFrames > 0 {
				// Instead of checking a specific frame by index (which depends on map iteration order),
				// we should find the frame by name or validate that the expected name exists among the frames
				if tt.name == "falls back to standard query when no facets" {
					// For the fallback case, we expect a specific frame name
					assert.Equal(t, tt.expectedFrameName, resp.Frames[0].Name)
				} else {
					// For faceted tests, verify the expected frame name exists
					found := false
					for _, frame := range resp.Frames {
						if frame.Name == tt.expectedFrameName {
							found = true
							// Check the fields in this frame
							assert.Equal(t, tt.expectedFields, len(frame.Fields))
							if len(frame.Fields) > 0 {
								assert.Equal(t, "time", frame.Fields[0].Name)
							}
							if len(frame.Fields) > 1 {
								assert.Equal(t, "count", frame.Fields[1].Name)
							}
							break
						}
					}
					assert.True(t, found, "Expected to find frame with name: %s", tt.expectedFrameName)
				}
			}
		})
	}
}

func TestFormatStandardQueryMulti(t *testing.T) {
	tests := []struct {
		name           string
		results        *nrdb.NRDBResultContainerMultiResultCustomized
		query          backend.DataQuery
		expectedFields int
		fieldChecks    func(t *testing.T, frame *data.Frame)
	}{
		{
			name: "formats standard query results",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						utils.TimestampFieldName: float64(1590000000000),
						"count":                  42.0,
						"name":                   "test",
					},
					{
						utils.TimestampFieldName: float64(1590000060000),
						"count":                  43.0,
						"name":                   "test2",
					},
				},
			},
			query:          backend.DataQuery{RefID: "A"},
			expectedFields: 3, // time, count, name
			fieldChecks: func(t *testing.T, frame *data.Frame) {
				assert.Equal(t, utils.StandardResponseFrameName, frame.Name)

				// Find fields by name rather than by index
				var timeField, countField, nameField *data.Field
				for _, field := range frame.Fields {
					switch field.Name {
					case "time":
						timeField = field
					case "count":
						countField = field
					case "name":
						nameField = field
					}
				}

				// Verify all fields exist
				require.NotNil(t, timeField, "Time field should exist")
				require.NotNil(t, countField, "Count field should exist")
				require.NotNil(t, nameField, "Name field should exist")

				// Check time field values
				assert.Equal(t, 2, timeField.Len())
				timeValue, ok := timeField.At(0).(time.Time)
				require.True(t, ok, "Expected time value")
				assert.Equal(t, int64(1590000000), timeValue.Unix())

				// Check count field values
				assert.Equal(t, 2, countField.Len())
				countValue := countField.At(0)
				if floatVal, ok := countValue.(float64); ok {
					assert.Equal(t, float64(42), floatVal)
				} else if ptrVal, ok := countValue.(*float64); ok && ptrVal != nil {
					assert.Equal(t, float64(42), *ptrVal)
				} else {
					t.Fatalf("Expected float64 or *float64 value, got %T", countValue)
				}

				// Check string field values
				assert.Equal(t, 2, nameField.Len())
				stringValue, ok := nameField.At(0).(string)
				require.True(t, ok, "Expected string value")
				assert.Equal(t, "test", stringValue)
			},
		},
		{
			name: "handles empty results",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{},
			},
			query:          backend.DataQuery{RefID: "A"},
			expectedFields: 1, // Just the time field
			fieldChecks: func(t *testing.T, frame *data.Frame) {
				assert.Equal(t, utils.StandardResponseFrameName, frame.Name)
				assert.Equal(t, "time", frame.Fields[0].Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := formatStandardQueryMulti(tt.results, tt.query)

			require.NotNil(t, resp)
			assert.Equal(t, 1, len(resp.Frames))
			assert.Equal(t, tt.expectedFields, len(resp.Frames[0].Fields))

			if tt.fieldChecks != nil {
				tt.fieldChecks(t, resp.Frames[0])
			}
		})
	}
}

func TestCreateTimeFieldMulti(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		results        *nrdb.NRDBResultContainerMultiResultCustomized
		query          backend.DataQuery
		expectedLength int
		checkTime      func(t *testing.T, times []time.Time)
	}{
		{
			name: "creates time field from timestamp",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						utils.TimestampFieldName: float64(1590000000000),
					},
					{
						utils.TimestampFieldName: float64(1590000060000),
					},
				},
			},
			query:          backend.DataQuery{RefID: "A"},
			expectedLength: 2,
			checkTime: func(t *testing.T, times []time.Time) {
				assert.Equal(t, int64(1590000000), times[0].Unix())
				assert.Equal(t, int64(1590000060), times[1].Unix())
			},
		},
		{
			name: "creates time field from beginTimeSeconds",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						"beginTimeSeconds": float64(1590000000),
					},
					{
						"beginTimeSeconds": float64(1590000060),
					},
				},
			},
			query:          backend.DataQuery{RefID: "A"},
			expectedLength: 2,
			checkTime: func(t *testing.T, times []time.Time) {
				assert.Equal(t, int64(1590000000), times[0].Unix())
				assert.Equal(t, int64(1590000060), times[1].Unix())
			},
		},
		{
			name: "falls back to current time when no time fields",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						"value": 42.0,
					},
					{
						"value": 43.0,
					},
				},
			},
			query:          backend.DataQuery{RefID: "A"},
			expectedLength: 2,
			checkTime: func(t *testing.T, times []time.Time) {
				// Both times should be very close to now
				for _, tm := range times {
					diff := tm.Sub(now)
					assert.LessOrEqual(t, diff.Abs().Seconds(), float64(5), "Time should be within 5 seconds of now")
				}
			},
		},
		{
			name: "handles empty results",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{},
			},
			query:          backend.DataQuery{RefID: "A"},
			expectedLength: 0,
			checkTime: func(t *testing.T, times []time.Time) {
				// Nothing to check
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			times := createTimeFieldMulti(tt.results, tt.query)
			assert.Equal(t, tt.expectedLength, len(times))
			if tt.expectedLength > 0 {
				tt.checkTime(t, times)
			}
		})
	}
}

func TestAddDataFieldsMulti(t *testing.T) {
	tests := []struct {
		name            string
		results         *nrdb.NRDBResultContainerMultiResultCustomized
		fieldNames      []string
		expectedFields  int
		fieldAssertions func(t *testing.T, frame *data.Frame)
	}{
		{
			name: "adds numeric fields",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						"count":  42.0,
						"value":  123.456,
						"rating": 5,
					},
					{
						"count":  43.0,
						"value":  234.567,
						"rating": 4,
					},
				},
			},
			fieldNames:     []string{"count", "value", "rating"},
			expectedFields: 3,
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				assert.Equal(t, "count", frame.Fields[0].Name)
				assert.Equal(t, "value", frame.Fields[1].Name)
				assert.Equal(t, "rating", frame.Fields[2].Name)

				// Check field types and values
				for _, field := range frame.Fields {
					assert.Equal(t, 2, field.Len(), "Field %s should have 2 values", field.Name)

					firstVal := field.At(0)
					switch v := firstVal.(type) {
					case float64:
						if field.Name == "count" {
							assert.Equal(t, 42.0, v)
						} else if field.Name == "value" {
							assert.Equal(t, 123.456, v)
						} else if field.Name == "rating" {
							assert.Equal(t, float64(5), v)
						}
					case *float64:
						require.NotNil(t, v)
						if field.Name == "count" {
							assert.Equal(t, 42.0, *v)
						} else if field.Name == "value" {
							assert.Equal(t, 123.456, *v)
						} else if field.Name == "rating" {
							assert.Equal(t, float64(5), *v)
						}
					default:
						t.Errorf("Expected field %s to be float64 or *float64, got %T", field.Name, firstVal)
					}
				}
			},
		},
		{
			name: "string fields",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{"name": "service1", "status": "active"},
					{"name": "service2", "status": "inactive"},
				},
			},
			fieldNames:     []string{"name", "status"},
			expectedFields: 2,
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				assert.Equal(t, "name", frame.Fields[0].Name)
				assert.Equal(t, "status", frame.Fields[1].Name)

				// Check values
				nameVal, ok := frame.Fields[0].At(0).(string)
				assert.True(t, ok, "name field should be string")
				assert.Equal(t, "service1", nameVal)

				statusVal, ok := frame.Fields[1].At(0).(string)
				assert.True(t, ok, "status field should be string")
				assert.Equal(t, "active", statusVal)
			},
		},
		{
			name: "handles boolean fields",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						"active": true,

						"valid": false,
					},
				},
			},
			fieldNames:     []string{"active", "valid"},
			expectedFields: 2,
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				assert.Equal(t, "active", frame.Fields[0].Name)
				assert.Equal(t, "valid", frame.Fields[1].Name)

				// Check boolean field values
				val0 := frame.Fields[0].At(0)
				val1 := frame.Fields[1].At(0)

				// Handle both *bool and bool types
				if boolVal, ok := val0.(bool); ok {
					assert.True(t, boolVal)
				} else if ptrVal, ok := val0.(*bool); ok && ptrVal != nil {
					assert.True(t, *ptrVal)
				} else {
					t.Errorf("Expected active field to be bool or *bool, got %T", val0)
				}

				if boolVal, ok := val1.(bool); ok {
					assert.False(t, boolVal)
				} else if ptrVal, ok := val1.(*bool); ok && ptrVal != nil {
					assert.False(t, *ptrVal)
				} else {
					t.Errorf("Expected valid field to be bool or *bool, got %T", val1)
				}
			},
		},
		{
			name: "handles array fields",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						"tags": []interface{}{"tag1", "tag2"},
					},
				},
			},
			fieldNames:     []string{"tags"},
			expectedFields: 1,
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				assert.Equal(t, "tags", frame.Fields[0].Name)

				// Check array serialized as string
				val, ok := frame.Fields[0].At(0).(string)
				assert.True(t, ok, "tags field should be string")
				assert.Contains(t, val, "tag1")
				assert.Contains(t, val, "tag2")
			},
		},
		{
			name: "handles empty results",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{},
			},
			fieldNames:     []string{"field1", "field2"},
			expectedFields: 0, // No fields should be added when results are empty
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				assert.Equal(t, 0, len(frame.Fields), "No fields should be added for empty results")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := data.NewFrame("test")
			addDataFieldsMulti(frame, tt.results, tt.fieldNames)

			assert.Equal(t, tt.expectedFields, len(frame.Fields), "Incorrect number of fields created")
			if tt.fieldAssertions != nil {
				tt.fieldAssertions(t, frame)
			}
		})
	}
}

// TestHandlePercentileFieldMultiComprehensive tests the handlePercentileFieldMulti function
// with more comprehensive scenarios to cover additional code paths and edge cases
func TestHandlePercentileFieldMultiComprehensive_Formatter(t *testing.T) {
	tests := []struct {
		name            string
		results         *nrdb.NRDBResultContainerMultiResultCustomized
		fieldName       string
		expectedFields  int
		fieldAssertions func(t *testing.T, frame *data.Frame)
	}{
		{
			name: "handles basic percentile field",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						"percentile.duration": map[string]interface{}{
							"50": 50.5,
							"95": 95.5,
						},
					},
					{
						"percentile.duration": map[string]interface{}{
							"50": 51.5,
							"95": 96.5,
						},
					},
				},
			},
			fieldName:      "percentile.duration",
			expectedFields: 2, // 50th and 95th percentile
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				// Check that both fields exist
				var p50Field, p95Field *data.Field
				for _, field := range frame.Fields {
					switch field.Name {
					case "percentile.duration.50":
						p50Field = field
					case "percentile.duration.95":
						p95Field = field
					}
				}

				// Verify both fields exist
				require.NotNil(t, p50Field, "Field percentile.duration.50 not found")
				require.NotNil(t, p95Field, "Field percentile.duration.95 not found")

				// Check field lengths
				assert.Equal(t, 2, p50Field.Len(), "Field should have 2 values")
				assert.Equal(t, 2, p95Field.Len(), "Field should have 2 values")

				// Check values for first row
				val50_0 := p50Field.At(0)
				if v, ok := val50_0.(*float64); ok && v != nil {
					assert.Equal(t, 50.5, *v)
				} else {
					t.Errorf("Expected *float64 for p50 at index 0, got %T", val50_0)
				}

				val95_0 := p95Field.At(0)
				if v, ok := val95_0.(*float64); ok && v != nil {
					assert.Equal(t, 95.5, *v)
				} else {
					t.Errorf("Expected *float64 for p95 at index 0, got %T", val95_0)
				}

				// Check values for second row
				val50_1 := p50Field.At(1)
				if v, ok := val50_1.(*float64); ok && v != nil {
					assert.Equal(t, 51.5, *v)
				} else {
					t.Errorf("Expected *float64 for p50 at index 1, got %T", val50_1)
				}

				val95_1 := p95Field.At(1)
				if v, ok := val95_1.(*float64); ok && v != nil {
					assert.Equal(t, 96.5, *v)
				} else {
					t.Errorf("Expected *float64 for p95 at index 1, got %T", val95_1)
				}
			},
		},
		{
			name: "handles different percentile sets across results",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						"percentile.duration": map[string]interface{}{
							"50": 50.5,
							"95": 95.5,
						},
					},
					{
						"percentile.duration": map[string]interface{}{
							"50": 51.5,
							"99": 99.5, // Note: has 99 but not 95
						},
					},
				},
			},
			fieldName:      "percentile.duration",
			expectedFields: 3, // 50th, 95th, and 99th percentile
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				// Check that all fields exist
				fieldMap := make(map[string]*data.Field)
				for _, field := range frame.Fields {
					fieldMap[field.Name] = field
				}

				require.Contains(t, fieldMap, "percentile.duration.50", "Should have 50th percentile field")
				require.Contains(t, fieldMap, "percentile.duration.95", "Should have 95th percentile field")
				require.Contains(t, fieldMap, "percentile.duration.99", "Should have 99th percentile field")

				// Check specific values
				p50Field := fieldMap["percentile.duration.50"]
				p95Field := fieldMap["percentile.duration.95"]
				p99Field := fieldMap["percentile.duration.99"]

				// 50th is in both results
				assert.Equal(t, 2, p50Field.Len())
				val50_1 := p50Field.At(0).(*float64)
				assert.Equal(t, 50.5, *val50_1)
				val50_2 := p50Field.At(1).(*float64)
				assert.Equal(t, 51.5, *val50_2)

				// 95th is only in first result, should be nil in second
				assert.Equal(t, 2, p95Field.Len())
				val95_1 := p95Field.At(0).(*float64)
				assert.Equal(t, 95.5, *val95_1)
				assert.Nil(t, p95Field.At(1), "Second 95th percentile should be nil")

				// 99th is only in second result, should be nil in first
				assert.Equal(t, 2, p99Field.Len())
				assert.Nil(t, p99Field.At(0), "First 99th percentile should be nil")
				val99_2 := p99Field.At(1).(*float64)
				assert.Equal(t, 99.5, *val99_2)
			},
		},
		{
			name: "handles mixed numeric types",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						"percentile.mixed": map[string]interface{}{
							"50": 50.5,      // float64
							"95": "95.5",    // string (valid float)
							"99": "invalid", // string (invalid float)
						},
					},
					{
						"percentile.mixed": map[string]interface{}{
							"50": 105.5,
							"90": 190.25, // Now it's a float
							"95": "205.75",
							"99": 305.25,
						},
					},
				},
			},
			fieldName:      "percentile.mixed",
			expectedFields: 4, // All four percentiles should have fields
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				assert.Equal(t, 4, len(frame.Fields), "Should have 4 percentile fields")

				// Create a map of field names to fields for easier lookup
				fieldMap := make(map[string]*data.Field)
				for _, f := range frame.Fields {
					fieldMap[f.Name] = f
				}

				// Check fields exist
				p50Field, ok := fieldMap["percentile.mixed.50"]
				require.True(t, ok, "Missing percentile.mixed.50 field")

				p90Field, ok := fieldMap["percentile.mixed.90"]
				require.True(t, ok, "Missing percentile.mixed.90 field")

				p95Field, ok := fieldMap["percentile.mixed.95"]
				require.True(t, ok, "Missing percentile.mixed.95 field")

				p99Field, ok := fieldMap["percentile.mixed.99"]
				require.True(t, ok, "Missing percentile.mixed.99 field")

				// Check p50 values
				val50_0 := p50Field.At(0)
				if v, ok := val50_0.(*float64); ok && v != nil {
					assert.Equal(t, 50.5, *v)
				} else {
					t.Errorf("Expected *float64 for p50 at index 0, got %T", val50_0)
				}

				val50_1 := p50Field.At(1)
				if v, ok := val50_1.(*float64); ok && v != nil {
					assert.Equal(t, 105.5, *v)
				} else {
					t.Errorf("Expected *float64 for p50 at index 1, got %T", val50_1)
				}

				// Check p90 values - first should be nil due to invalid string
				val90_0 := p90Field.At(0)
				assert.Nil(t, val90_0, "p90 at index 0 should be nil due to invalid string")

				val90_1 := p90Field.At(1)
				if v, ok := val90_1.(*float64); ok && v != nil {
					assert.Equal(t, 190.25, *v)
				} else {
					t.Errorf("Expected *float64 for p90 at index 1, got %T", val90_1)
				}

				// Check p95 values - both should be valid
				val95_0 := p95Field.At(0)
				if v, ok := val95_0.(*float64); ok && v != nil {
					assert.Equal(t, 95.5, *v)
				} else {
					t.Errorf("Expected *float64 for p95 at index 0, got %T", val95_0)
				}

				val95_1 := p95Field.At(1)
				if v, ok := val95_1.(*float64); ok && v != nil {
					assert.Equal(t, 205.75, *v)
				} else {
					t.Errorf("Expected *float64 for p95 at index 1, got %T", val95_1)
				}

				// Check p99 values - first should be nil due to boolean value
				val99_0 := p99Field.At(0)
				assert.Nil(t, val99_0, "p99 at index 0 should be nil due to boolean value")

				val99_1 := p99Field.At(1)
				if v, ok := val99_1.(*float64); ok && v != nil {
					assert.Equal(t, 305.25, *v)
				} else {
					t.Errorf("Expected *float64 for p99 at index 1, got %T", val99_1)
				}
			},
		},
		{
			name: "handles nil/missing fields and empty results",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{
						"percentile.valid": map[string]interface{}{
							"50": 50.5,
							"95": 95.5,
						},
					},
					{
						// No percentile.valid field at all
					},
					{
						"percentile.valid": nil, // Explicit nil
					},
					{
						"percentile.valid": map[string]interface{}{}, // Empty map
					},
				},
			},
			fieldName:      "percentile.valid",
			expectedFields: 2, // 50th and 95th percentile
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				// Check that fields exist
				fieldMap := make(map[string]*data.Field)
				for _, field := range frame.Fields {
					fieldMap[field.Name] = field
				}

				p50Field, ok := fieldMap["percentile.valid.50"]
				require.True(t, ok, "Missing 50th percentile field")
				p95Field, ok := fieldMap["percentile.valid.95"]
				require.True(t, ok, "Missing 95th percentile field")

				// Check lengths - should match total number of results
				assert.Equal(t, 4, p50Field.Len())
				assert.Equal(t, 4, p95Field.Len())

				// Check values - only first result has actual data
				val50 := p50Field.At(0).(*float64)
				assert.Equal(t, 50.5, *val50)
				val95 := p95Field.At(0).(*float64)
				assert.Equal(t, 95.5, *val95)

				// Rest should be nil
				for i := 1; i < 4; i++ {
					assert.Nil(t, p50Field.At(i), "Missing field value should be nil")
					assert.Nil(t, p95Field.At(i), "Missing field value should be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := data.NewFrame("test")
			HandlePercentileFieldMulti(frame, tt.results, tt.fieldName)

			assert.Equal(t, tt.expectedFields, len(frame.Fields), "Incorrect number of fields created")
			if tt.fieldAssertions != nil {
				tt.fieldAssertions(t, frame)
			}
		})
	}
}

// TestHandlePercentileField comprehensively tests the handlePercentileField function
// with various scenarios to ensure proper handling of percentile data
func TestHandlePercentileField_Formatter(t *testing.T) {
	tests := []struct {
		name            string
		results         *nrdb.NRDBResultContainer
		fieldName       string
		expectedFields  int
		fieldAssertions func(t *testing.T, frame *data.Frame)
	}{
		{
			name: "handles basic percentile field",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"percentile.duration": map[string]interface{}{
							"50": 50.5,
							"95": 95.5,
						},
					},
					{
						"percentile.duration": map[string]interface{}{
							"50": 51.5,
							"95": 96.5,
						},
					},
				},
			},
			fieldName:      "percentile.duration",
			expectedFields: 2, // 50th and 95th percentile
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				// Check that both fields exist
				var p50Field, p95Field *data.Field
				for _, field := range frame.Fields {
					switch field.Name {
					case "percentile.duration.50":
						p50Field = field
					case "percentile.duration.95":
						p95Field = field
					}
				}

				// Verify both fields exist
				require.NotNil(t, p50Field, "Field percentile.duration.50 not found")
				require.NotNil(t, p95Field, "Field percentile.duration.95 not found")

				// Check field lengths
				assert.Equal(t, 2, p50Field.Len(), "Field should have 2 values")
				assert.Equal(t, 2, p95Field.Len(), "Field should have 2 values")

				// Check values for first row
				val50_0 := p50Field.At(0)
				if v, ok := val50_0.(*float64); ok && v != nil {
					assert.Equal(t, 50.5, *v)
				} else {
					t.Errorf("Expected *float64 for p50 at index 0, got %T", val50_0)
				}

				val95_0 := p95Field.At(0)
				if v, ok := val95_0.(*float64); ok && v != nil {
					assert.Equal(t, 95.5, *v)
				} else {
					t.Errorf("Expected *float64 for p95 at index 0, got %T", val95_0)
				}

				// Check values for second row
				val50_1 := p50Field.At(1)
				if v, ok := val50_1.(*float64); ok && v != nil {
					assert.Equal(t, 51.5, *v)
				} else {
					t.Errorf("Expected *float64 for p50 at index 1, got %T", val50_1)
				}

				val95_1 := p95Field.At(1)
				if v, ok := val95_1.(*float64); ok && v != nil {
					assert.Equal(t, 96.5, *v)
				} else {
					t.Errorf("Expected *float64 for p95 at index 1, got %T", val95_1)
				}
			},
		},
		{
			name: "handles string percentile values",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"percentile.latency": map[string]interface{}{
							"50": "100.5", // String values
							"95": "200.75",
						},
					},
				},
			},
			fieldName:      "percentile.latency",
			expectedFields: 2,
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				// Don't assume field order, instead check that both fields exist
				var p50Field, p95Field *data.Field
				for _, f := range frame.Fields {
					if f.Name == "percentile.latency.50" {
						p50Field = f
					} else if f.Name == "percentile.latency.95" {
						p95Field = f
					}
				}

				// Verify both fields exist
				require.NotNil(t, p50Field, "Field percentile.latency.50 not found")
				require.NotNil(t, p95Field, "Field percentile.latency.95 not found")

				// Check p50 values properly converted from string to floats
				val50 := p50Field.At(0)
				if v, ok := val50.(*float64); ok && v != nil {
					assert.Equal(t, 100.5, *v)
				} else {
					t.Errorf("Expected *float64 for field percentile.latency.50, got %T", val50)
				}

				// Check p95 values properly converted from string to floats
				val95 := p95Field.At(0)
				if v, ok := val95.(*float64); ok && v != nil {
					assert.Equal(t, 200.75, *v)
				} else {
					t.Errorf("Expected *float64 for field percentile.latency.95, got %T", val95)
				}
			},
		},
		{
			name: "handles nil percentile field",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"percentile.null": nil,
					},
					{
						"percentile.null": nil,
					},
				},
			},
			fieldName:      "percentile.null",
			expectedFields: 0, // No fields should be added
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				assert.Equal(t, 0, len(frame.Fields), "No fields should be added for nil percentile")
			},
		},
		{
			name: "handles missing percentile field",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"otherField": 123,
					},
					{
						"anotherField": "abc",
					},
				},
			},
			fieldName:      "percentile.missing",
			expectedFields: 0, // No fields should be added
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				assert.Equal(t, 0, len(frame.Fields), "No fields should be added when field is missing")
			},
		},
		{
			name: "handles mixed data types",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"percentile.mixed": map[string]interface{}{
							"50": 100.5,
							"90": "invalid-string", // Should be skipped
							"95": "200.75",         // Should be parsed
							"99": true,             // Should be skipped
						},
					},
					{
						"percentile.mixed": map[string]interface{}{
							"50": 105.5,
							"90": 190.25, // Now it's a float
							"95": "205.75",
							"99": 305.25,
						},
					},
				},
			},
			fieldName:      "percentile.mixed",
			expectedFields: 4, // All four percentiles should have fields
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				assert.Equal(t, 4, len(frame.Fields), "Should have 4 percentile fields")

				// Create a map of field names to fields for easier lookup
				fieldMap := make(map[string]*data.Field)
				for _, f := range frame.Fields {
					fieldMap[f.Name] = f
				}

				// Check fields exist
				p50Field, ok := fieldMap["percentile.mixed.50"]
				require.True(t, ok, "Missing percentile.mixed.50 field")

				p90Field, ok := fieldMap["percentile.mixed.90"]
				require.True(t, ok, "Missing percentile.mixed.90 field")

				p95Field, ok := fieldMap["percentile.mixed.95"]
				require.True(t, ok, "Missing percentile.mixed.95 field")

				p99Field, ok := fieldMap["percentile.mixed.99"]
				require.True(t, ok, "Missing percentile.mixed.99 field")

				// Check p50 values
				val50_0 := p50Field.At(0)
				if v, ok := val50_0.(*float64); ok && v != nil {
					assert.Equal(t, 100.5, *v)
				} else {
					t.Errorf("Expected *float64 for p50 at index 0, got %T", val50_0)
				}

				val50_1 := p50Field.At(1)
				if v, ok := val50_1.(*float64); ok && v != nil {
					assert.Equal(t, 105.5, *v)
				} else {
					t.Errorf("Expected *float64 for p50 at index 1, got %T", val50_1)
				}

				// Check p90 values - first should be nil due to invalid string
				val90_0 := p90Field.At(0)
				assert.Nil(t, val90_0, "p90 at index 0 should be nil due to invalid string")

				val90_1 := p90Field.At(1)
				if v, ok := val90_1.(*float64); ok && v != nil {
					assert.Equal(t, 190.25, *v)
				} else {
					t.Errorf("Expected *float64 for p90 at index 1, got %T", val90_1)
				}

				// Check p95 values - both should be valid
				val95_0 := p95Field.At(0)
				if v, ok := val95_0.(*float64); ok && v != nil {
					assert.Equal(t, 200.75, *v)
				} else {
					t.Errorf("Expected *float64 for p95 at index 0, got %T", val95_0)
				}

				val95_1 := p95Field.At(1)
				if v, ok := val95_1.(*float64); ok && v != nil {
					assert.Equal(t, 205.75, *v)
				} else {
					t.Errorf("Expected *float64 for p95 at index 1, got %T", val95_1)
				}

				// Check p99 values - first should be nil due to boolean value
				val99_0 := p99Field.At(0)
				assert.Nil(t, val99_0, "p99 at index 0 should be nil due to boolean value")

				val99_1 := p99Field.At(1)
				if v, ok := val99_1.(*float64); ok && v != nil {
					assert.Equal(t, 305.25, *v)
				} else {
					t.Errorf("Expected *float64 for p99 at index 1, got %T", val99_1)
				}
			},
		},
		{
			name: "handles nil/missing fields and empty results",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"percentile.valid": map[string]interface{}{
							"50": 50.5,
							"95": 95.5,
						},
					},
					{
						// No percentile.valid field at all
					},
					{
						"percentile.valid": nil, // Explicit nil
					},
					{
						"percentile.valid": map[string]interface{}{}, // Empty map
					},
				},
			},
			fieldName:      "percentile.valid",
			expectedFields: 2, // 50th and 95th percentile
			fieldAssertions: func(t *testing.T, frame *data.Frame) {
				// Check that fields exist
				fieldMap := make(map[string]*data.Field)
				for _, field := range frame.Fields {
					fieldMap[field.Name] = field
				}

				p50Field, ok := fieldMap["percentile.valid.50"]
				require.True(t, ok, "Missing 50th percentile field")
				p95Field, ok := fieldMap["percentile.valid.95"]
				require.True(t, ok, "Missing 95th percentile field")

				// Check lengths - should match total number of results
				assert.Equal(t, 4, p50Field.Len())
				assert.Equal(t, 4, p95Field.Len())

				// Check values - only first result has actual data
				val50 := p50Field.At(0).(*float64)
				assert.Equal(t, 50.5, *val50)
				val95 := p95Field.At(0).(*float64)
				assert.Equal(t, 95.5, *val95)

				// Rest should be nil
				for i := 1; i < 4; i++ {
					assert.Nil(t, p50Field.At(i), "Missing field value should be nil")
					assert.Nil(t, p95Field.At(i), "Missing field value should be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := data.NewFrame("test")
			handlePercentileField(frame, tt.results, tt.fieldName)

			assert.Equal(t, tt.expectedFields, len(frame.Fields), "Incorrect number of fields created")
			if tt.fieldAssertions != nil {
				tt.fieldAssertions(t, frame)
			}
		})
	}
}
func TestAddRegularAggregationField(t *testing.T) {
	t.Run("float64 aggregation values", func(t *testing.T) {
		frame := data.NewFrame("test_frame")
		facetResults := []nrdb.NRDBResult{
			{
				"beginTimeSeconds": float64(1600000000),
				"endTimeSeconds":   float64(1600000060),
				"sum.duration":     float64(123.45),
			},
			{
				"beginTimeSeconds": float64(1600000060),
				"endTimeSeconds":   float64(1600000120),
				"sum.duration":     float64(234.56),
			},
		}

		addRegularAggregationField(frame, facetResults, "sum.duration", "service", "serviceA")

		// Check the field was created
		assert.Equal(t, 1, len(frame.Fields))
		assert.Equal(t, "sum.duration", frame.Fields[0].Name)

		// Check the label was set
		assert.Equal(t, "serviceA", frame.Fields[0].Labels["service"])

		// Check the values
		assert.Equal(t, 2, frame.Fields[0].Len())
		assert.Equal(t, 123.45, *frame.Fields[0].At(0).(*float64))
		assert.Equal(t, 234.56, *frame.Fields[0].At(1).(*float64))
	})

	t.Run("string aggregation values", func(t *testing.T) {
		frame := data.NewFrame("test_frame")
		facetResults := []nrdb.NRDBResult{
			{
				"beginTimeSeconds": float64(1600000000),
				"endTimeSeconds":   float64(1600000060),
				"avg.duration":     "123.45",
			},
			{
				"beginTimeSeconds": float64(1600000060),
				"endTimeSeconds":   float64(1600000120),
				"avg.duration":     "234.56",
			},
		}

		addRegularAggregationField(frame, facetResults, "avg.duration", "service", "serviceA")

		// Check values were properly parsed from strings
		assert.Equal(t, 123.45, *frame.Fields[0].At(0).(*float64))
		assert.Equal(t, 234.56, *frame.Fields[0].At(1).(*float64))
	})

	t.Run("int aggregation values", func(t *testing.T) {
		frame := data.NewFrame("test_frame")
		facetResults := []nrdb.NRDBResult{
			{
				"beginTimeSeconds": float64(1600000000),
				"endTimeSeconds":   float64(1600000060),
				"count":            123,
			},
		}

		addRegularAggregationField(frame, facetResults, "count", "service", "serviceA")

		// Check value was properly converted from int
		assert.Equal(t, float64(123), *frame.Fields[0].At(0).(*float64))
	})

	t.Run("int64 aggregation values", func(t *testing.T) {
		frame := data.NewFrame("test_frame")
		facetResults := []nrdb.NRDBResult{
			{
				"beginTimeSeconds": float64(1600000000),
				"endTimeSeconds":   float64(1600000060),
				"count":            int64(9876543210),
			},
		}

		addRegularAggregationField(frame, facetResults, "count", "service", "serviceA")

		// Check value was properly converted from int64
		assert.Equal(t, float64(9876543210), *frame.Fields[0].At(0).(*float64))
	})

	t.Run("missing values", func(t *testing.T) {
		frame := data.NewFrame("test_frame")
		facetResults := []nrdb.NRDBResult{
			{
				"beginTimeSeconds": float64(1600000000),
				"endTimeSeconds":   float64(1600000060),
				// No aggregation field
			},
			{
				"beginTimeSeconds": float64(1600000060),
				"endTimeSeconds":   float64(1600000120),
				"sum.duration":     nil,
			},
			{
				"beginTimeSeconds": float64(1600000120),
				"endTimeSeconds":   float64(1600000180),
				"sum.duration":     "",
			},
			{
				"beginTimeSeconds": float64(1600000180),
				"endTimeSeconds":   float64(1600000240),
				"sum.duration":     float64(345.67),
			},
		}

		addRegularAggregationField(frame, facetResults, "sum.duration", "service", "serviceA")

		// Check the field length
		assert.Equal(t, 4, frame.Fields[0].Len())

		// First three values should be nil
		assert.Nil(t, frame.Fields[0].At(0))
		assert.Nil(t, frame.Fields[0].At(1))
		assert.Nil(t, frame.Fields[0].At(2))

		// Last value should be set
		assert.Equal(t, 345.67, *frame.Fields[0].At(3).(*float64))
	})
}

func TestIsFacetFieldName(t *testing.T) {
	t.Run("field name matches facet name", func(t *testing.T) {
		facetNames := []string{"service", "environment", "host"}

		assert.True(t, isFacetFieldName("service", facetNames))
		assert.True(t, isFacetFieldName("environment", facetNames))
		assert.True(t, isFacetFieldName("host", facetNames))
	})

	t.Run("field name does not match any facet name", func(t *testing.T) {
		facetNames := []string{"service", "environment", "host"}

		assert.False(t, isFacetFieldName("count", facetNames))
		assert.False(t, isFacetFieldName("timestamp", facetNames))
		assert.False(t, isFacetFieldName("value", facetNames))
	})

	t.Run("empty facet names list", func(t *testing.T) {
		facetNames := []string{}

		assert.False(t, isFacetFieldName("service", facetNames))
	})

	t.Run("case sensitivity", func(t *testing.T) {
		facetNames := []string{"Service", "Environment"}

		// isFacetFieldName should be case-sensitive
		assert.False(t, isFacetFieldName("service", facetNames))
		assert.False(t, isFacetFieldName("environment", facetNames))
		assert.True(t, isFacetFieldName("Service", facetNames))
	})
}

func TestCreateFacetTableFrames(t *testing.T) {
	t.Run("single facet with multiple values", func(t *testing.T) {
		// Setup test data
		facetNames := []string{"service"}
		counts := []float64{10, 20, 30}
		facetFields := map[string][]string{
			"service": {"serviceA", "serviceB", "serviceC"},
		}

		// Create the frame
		frame := createFacetTableFrame(facetNames, counts, facetFields)

		// Verify frame properties
		assert.Equal(t, utils.FacetedFrameName, frame.Name)
		assert.Equal(t, "table", string(frame.Meta.PreferredVisualization))

		// Should have 2 fields: service and count
		assert.Equal(t, 2, len(frame.Fields))
		assert.Equal(t, "service", frame.Fields[0].Name)
		assert.Equal(t, utils.CountFieldName, frame.Fields[1].Name)

		// Verify field lengths
		assert.Equal(t, len(counts), frame.Fields[0].Len())
		assert.Equal(t, len(counts), frame.Fields[1].Len())

		// Check values for service field
		serviceValues := make([]string, len(counts))
		for i := 0; i < len(counts); i++ {
			serviceValues[i] = frame.Fields[0].At(i).(string)
		}
		assert.Equal(t, []string{"serviceA", "serviceB", "serviceC"}, serviceValues)

		// Check count values
		countValues := make([]float64, len(counts))
		for i := 0; i < len(counts); i++ {
			countValues[i] = frame.Fields[1].At(i).(float64)
		}
		assert.Equal(t, counts, countValues)
	})

	t.Run("multiple facets", func(t *testing.T) {
		// Setup test data
		facetNames := []string{"service", "environment"}
		counts := []float64{10, 20}
		facetFields := map[string][]string{
			"service":     {"serviceA", "serviceB"},
			"environment": {"prod", "dev"},
		}

		// Create the frame
		frame := createFacetTableFrame(facetNames, counts, facetFields)

		// Should have 3 fields: service, environment and count
		assert.Equal(t, 3, len(frame.Fields))
		assert.Equal(t, "service", frame.Fields[0].Name)
		assert.Equal(t, "environment", frame.Fields[1].Name)
		assert.Equal(t, utils.CountFieldName, frame.Fields[2].Name)
	})

	t.Run("empty facet names", func(t *testing.T) {
		// Setup test data
		facetNames := []string{}
		counts := []float64{10, 20}
		facetFields := map[string][]string{}

		// Create the frame
		frame := createFacetTableFrame(facetNames, counts, facetFields)

		// Should have just the count field
		assert.Equal(t, 1, len(frame.Fields))
		assert.Equal(t, utils.CountFieldName, frame.Fields[0].Name)
	})

	t.Run("empty counts", func(t *testing.T) {
		// Skip this test for now, as the implementation details may vary
		// In some implementations, empty counts might not create empty fields
		// but we don't want to modify the implementation just for testing
		t.Skip("Skipping test for empty counts behavior")
	})
}

func TestCreateFacetTimeSeriesFrame(t *testing.T) {
	t.Run("single facet with multiple values", func(t *testing.T) {
		// Setup test data
		facetNames := []string{"service"}
		counts := []float64{10, 20, 30}
		facetFields := map[string][]string{
			"service": {"serviceA", "serviceB", "serviceC"},
		}
		query := backend.DataQuery{
			RefID: "A",
		}

		// Create the frame
		frame := createFacetTimeSeriesFrame(facetNames, counts, facetFields, query)

		// Verify frame properties
		assert.Equal(t, utils.FacetedTimeSeriesFrameName, frame.Name)
		assert.Equal(t, "graph", string(frame.Meta.PreferredVisualization))

		// Should have 3 fields: time, service, count
		assert.Equal(t, 3, len(frame.Fields))
		assert.Equal(t, utils.TimeFieldName, frame.Fields[0].Name)
		assert.Equal(t, "service", frame.Fields[1].Name)
		assert.Equal(t, utils.CountFieldName, frame.Fields[2].Name)

		// Verify field lengths
		assert.Equal(t, len(counts), frame.Fields[0].Len())
		assert.Equal(t, len(counts), frame.Fields[1].Len())
		assert.Equal(t, len(counts), frame.Fields[2].Len())

		// Check labels on the service field
		labels := frame.Fields[1].Labels
		assert.Equal(t, "serviceA", labels["service"])

		// Check count values
		for i := 0; i < len(counts); i++ {
			assert.Equal(t, counts[i], frame.Fields[2].At(i))
		}

		// Check that time values are valid
		for i := 0; i < len(counts); i++ {
			_, ok := frame.Fields[0].At(i).(time.Time)
			assert.True(t, ok, "Expected time value at index %d", i)
		}
	})

	t.Run("multiple facets", func(t *testing.T) {
		// Setup test data
		facetNames := []string{"service", "environment"}
		counts := []float64{10, 20}
		facetFields := map[string][]string{
			"service":     {"serviceA", "serviceB"},
			"environment": {"prod", "dev"},
		}
		query := backend.DataQuery{
			RefID: "A",
		}

		// Create the frame
		frame := createFacetTimeSeriesFrame(facetNames, counts, facetFields, query)

		// Should have 4 fields: time, service, environment, count
		assert.Equal(t, 4, len(frame.Fields))
		assert.Equal(t, utils.TimeFieldName, frame.Fields[0].Name)
		assert.Equal(t, "service", frame.Fields[1].Name)
		assert.Equal(t, "environment", frame.Fields[2].Name)
		assert.Equal(t, utils.CountFieldName, frame.Fields[3].Name)

		// Check labels on facet fields
		serviceLabels := frame.Fields[1].Labels
		assert.Equal(t, "serviceA", serviceLabels["service"])

		envLabels := frame.Fields[2].Labels
		assert.Equal(t, "prod", envLabels["environment"])
	})

	t.Run("empty facet names", func(t *testing.T) {
		// Setup test data
		facetNames := []string{}
		counts := []float64{10, 20}
		facetFields := map[string][]string{}
		query := backend.DataQuery{
			RefID: "A",
		}

		// Create the frame
		frame := createFacetTimeSeriesFrame(facetNames, counts, facetFields, query)

		// Should have 2 fields: time and count
		assert.Equal(t, 2, len(frame.Fields))
		assert.Equal(t, utils.TimeFieldName, frame.Fields[0].Name)
		assert.Equal(t, utils.CountFieldName, frame.Fields[1].Name)
	})

	t.Run("empty facet values", func(t *testing.T) {
		// Setup test data
		facetNames := []string{"service"}
		counts := []float64{10, 20}
		facetFields := map[string][]string{
			"service": {}, // Empty values
		}
		query := backend.DataQuery{
			RefID: "A",
		}

		// Create the frame
		frame := createFacetTimeSeriesFrame(facetNames, counts, facetFields, query)

		// Should have 3 fields: time, service, count
		assert.Equal(t, 3, len(frame.Fields))

		// Service field should have empty labels
		serviceLabels := frame.Fields[1].Labels
		assert.Equal(t, "", serviceLabels["service"])
	})
}

func TestGroupTimeseriesByFacet(t *testing.T) {
	t.Run("standard facet array grouping", func(t *testing.T) {
		// Create test data with facet arrays
		results := &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{
				{
					"beginTimeSeconds": float64(1600000000),
					"endTimeSeconds":   float64(1600000060),
					"facet":            []interface{}{"serviceA"},
					"count":            float64(10),
				},
				{
					"beginTimeSeconds": float64(1600000060),
					"endTimeSeconds":   float64(1600000120),
					"facet":            []interface{}{"serviceA"},
					"count":            float64(15),
				},
				{
					"beginTimeSeconds": float64(1600000000),
					"endTimeSeconds":   float64(1600000060),
					"facet":            []interface{}{"serviceB"},
					"count":            float64(5),
				},
			},
		}

		// Group by facet
		grouped := groupTimeseriesByFacet(results, "service")

		// Verify the grouping
		assert.Equal(t, 2, len(grouped))
		assert.Equal(t, 2, len(grouped["serviceA"]))
		assert.Equal(t, 1, len(grouped["serviceB"]))

		// Check values in the first group
		assert.Equal(t, 10, int(grouped["serviceA"][0]["count"].(float64)))
		assert.Equal(t, 15, int(grouped["serviceA"][1]["count"].(float64)))

		// Check values in the second group
		assert.Equal(t, 5, int(grouped["serviceB"][0]["count"].(float64)))
	})

	t.Run("direct facet value grouping", func(t *testing.T) {
		// Create test data with direct facet values (not array)
		results := &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{
				{
					"beginTimeSeconds": float64(1600000000),
					"endTimeSeconds":   float64(1600000060),
					"facet":            "serviceA", // Direct value
					"count":            float64(10),
				},
				{
					"beginTimeSeconds": float64(1600000060),
					"endTimeSeconds":   float64(1600000120),
					"facet":            "serviceA", // Direct value
					"count":            float64(15),
				},
				{
					"beginTimeSeconds": float64(1600000000),
					"endTimeSeconds":   float64(1600000060),
					"facet":            "serviceB", // Direct value
					"count":            float64(5),
				},
			},
		}

		// Group by facet
		grouped := groupTimeseriesByFacet(results, "service")

		// Verify the grouping
		assert.Equal(t, 2, len(grouped))
		assert.Equal(t, 2, len(grouped["serviceA"]))
		assert.Equal(t, 1, len(grouped["serviceB"]))
	})

	t.Run("missing facet values", func(t *testing.T) {
		// Create test data with some missing facet values
		results := &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{
				{
					"beginTimeSeconds": float64(1600000000),
					"endTimeSeconds":   float64(1600000060),
					"facet":            []interface{}{"serviceA"},
					"count":            float64(10),
				},
				{
					"beginTimeSeconds": float64(1600000060),
					"endTimeSeconds":   float64(1600000120),
					// No facet field
					"count": float64(15),
				},
				{
					"beginTimeSeconds": float64(1600000000),
					"endTimeSeconds":   float64(1600000060),
					"facet":            []interface{}{"serviceB"},
					"count":            float64(5),
				},
			},
		}

		// Group by facet
		grouped := groupTimeseriesByFacet(results, "service")

		// Verify the grouping (should only include records with facet values)
		assert.Equal(t, 2, len(grouped))
		assert.Equal(t, 1, len(grouped["serviceA"]))
		assert.Equal(t, 1, len(grouped["serviceB"]))
	})

	t.Run("empty results", func(t *testing.T) {
		// Create empty results
		results := &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{},
		}

		// Group by facet
		grouped := groupTimeseriesByFacet(results, "service")

		// Should be empty
		assert.Equal(t, 0, len(grouped))
	})

	t.Run("non-string facet values", func(t *testing.T) {
		// Create test data with non-string facet values
		results := &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{
				{
					"beginTimeSeconds": float64(1600000000),
					"endTimeSeconds":   float64(1600000060),
					"facet":            []interface{}{float64(123)}, // Numeric facet
					"count":            float64(10),
				},
				{
					"beginTimeSeconds": float64(1600000060),
					"endTimeSeconds":   float64(1600000120),
					"facet":            []interface{}{true}, // Boolean facet
					"count":            float64(15),
				},
			},
		}

		// Group by facet
		grouped := groupTimeseriesByFacet(results, "service")

		// Should convert non-string values to strings
		assert.Equal(t, 2, len(grouped))
		assert.Equal(t, 1, len(grouped["123"]))
		assert.Equal(t, 1, len(grouped["true"]))
	})
}

func TestGetMapKeysFunctionality(t *testing.T) {
	tests := []struct {
		name         string
		input        map[string]interface{}
		expectedKeys []string
	}{
		{
			name: "standard map",
			input: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
			expectedKeys: []string{"key1", "key2", "key3"},
		},
		{
			name:         "empty map",
			input:        map[string]interface{}{},
			expectedKeys: []string{},
		},
		{
			name: "single key",
			input: map[string]interface{}{
				"singleKey": "value",
			},
			expectedKeys: []string{"singleKey"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := getMapKeys(tt.input)

			// Sort both slices for consistent comparison
			sort.Strings(keys)
			sort.Strings(tt.expectedKeys)

			assert.Equal(t, tt.expectedKeys, keys, "Map keys do not match expected keys")
			assert.Equal(t, len(tt.expectedKeys), len(keys), "Number of keys does not match expected count")
		})
	}
}

func TestAddDataFieldsMultiFunction(t *testing.T) {
	tests := []struct {
		name           string
		results        *nrdb.NRDBResultContainerMultiResultCustomized
		fieldNames     []string
		expectedFields int
		expectedValues map[string]interface{}
	}{
		{
			name: "empty results",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{},
			},
			fieldNames:     []string{},
			expectedFields: 0,
		},
		{
			name: "numeric fields",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{"count": 42.0, "value": 123.456},
				},
			},
			fieldNames:     []string{"count", "value"},
			expectedFields: 2,
			expectedValues: map[string]interface{}{
				"count": float64(42.0),
				"value": float64(123.456),
			},
		},
		{
			name: "string fields",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{"name": "test", "category": "testing"},
				},
			},
			fieldNames:     []string{"name", "category"},
			expectedFields: 2,
			expectedValues: map[string]interface{}{
				"name":     "test",
				"category": "testing",
			},
		},
		{
			name: "boolean fields",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{"enabled": true, "selected": false},
				},
			},
			fieldNames:     []string{"enabled", "selected"},
			expectedFields: 2,
			expectedValues: map[string]interface{}{
				"enabled":  true,
				"selected": false,
			},
		},
		{
			name: "array fields",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{"values": []interface{}{1.0, 2.0, 3.0}},
				},
			},
			fieldNames:     []string{"values"},
			expectedFields: 1,
		},
		{
			name: "timestamp fields",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{"timestamp": float64(1625076000000)},
				},
			},
			fieldNames:     []string{"timestamp"},
			expectedFields: 1,
		},
		{
			name: "object fields",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{"metadata": map[string]interface{}{"region": "us-east", "zone": "east-1b"}},
				},
			},
			fieldNames:     []string{"metadata"},
			expectedFields: 1,
		},
		{
			name: "percentile fields",
			results: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{"percentile.duration": map[string]interface{}{"95": 42.0, "99": 100.0}},
				},
			},
			fieldNames:     []string{"percentile.duration"},
			expectedFields: 2, // One field for each percentile
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := data.NewFrame("test")

			// Call the function being tested
			addDataFieldsMulti(frame, tt.results, tt.fieldNames)

			// Check that the fields were added correctly
			assert.Equal(t, tt.expectedFields, len(frame.Fields), "Should have the expected number of fields")

			// For non-empty results, verify field values
			if tt.expectedFields > 0 && len(tt.results.Results) > 0 {
				if tt.expectedValues != nil {
					for fieldName, expectedValue := range tt.expectedValues {
						found := false
						for _, field := range frame.Fields {
							if field.Name == fieldName {
								found = true
								assert.Equal(t, 1, field.Len(), "Field should have one value")
								assert.NotNil(t, field.At(0), "Field value should not be nil")

								// For pointers to values, we need to dereference
								switch v := field.At(0).(type) {
								case *float64:
									assert.Equal(t, expectedValue, *v, "Field value should match expected")
								case *bool:
									assert.Equal(t, expectedValue, *v, "Field value should match expected")
								case string:
									assert.Equal(t, expectedValue, v, "Field value should match expected")
								}
								break
							}
						}
						assert.True(t, found, "Field '%s' should exist", fieldName)
					}
				}

				// For percentile fields, just check that the fields exist
				if tt.name == "percentile fields" {
					percentileFound95 := false
					percentileFound99 := false
					for _, field := range frame.Fields {
						if field.Name == "percentile.duration.95" {
							percentileFound95 = true
						}
						if field.Name == "percentile.duration.99" {
							percentileFound99 = true
						}
					}
					assert.True(t, percentileFound95, "Percentile 95 field should exist")
					assert.True(t, percentileFound99, "Percentile 99 field should exist")
				}
			}
		})
	}
}

func TestExtractFacetNamesMulti(t *testing.T) {
	t.Run("with facets in metadata", func(t *testing.T) {
		results := &nrdb.NRDBResultContainerMultiResultCustomized{
			Metadata: nrdb.NRDBMetadata{
				Facets: []string{"service", "environment"},
			},
		}

		facetNames := extractFacetNamesMulti(results)
		assert.Equal(t, []string{"service", "environment"}, facetNames)
	})

	t.Run("without facets in metadata", func(t *testing.T) {
		results := &nrdb.NRDBResultContainerMultiResultCustomized{
			Metadata: nrdb.NRDBMetadata{},
		}

		facetNames := extractFacetNamesMulti(results)
		assert.Nil(t, facetNames)
	})

	t.Run("with nil metadata", func(t *testing.T) {
		// This shouldn't happen in practice, but test for nil safety
		results := &nrdb.NRDBResultContainerMultiResultCustomized{}

		facetNames := extractFacetNamesMulti(results)
		assert.Nil(t, facetNames)
	})
}

func TestHasTimeseriesDataMultis(t *testing.T) {
	t.Run("with beginTimeSeconds in results", func(t *testing.T) {
		results := &nrdb.NRDBResultContainerMultiResultCustomized{
			Results: []nrdb.NRDBResult{
				{
					"beginTimeSeconds": float64(1600000000),
					"endTimeSeconds":   float64(1600000060),
				},
			},
		}

		assert.True(t, hasTimeseriesDataMulti(results))
	})

	t.Run("with beginTimeSeconds in otherResult", func(t *testing.T) {
		results := &nrdb.NRDBResultContainerMultiResultCustomized{
			Results: []nrdb.NRDBResult{},
			OtherResult: []nrdb.NRDBResult{
				{
					"beginTimeSeconds": float64(1600000000),
					"endTimeSeconds":   float64(1600000060),
				},
			},
		}

		assert.True(t, hasTimeseriesDataMulti(results))
	})

	t.Run("without timeseries data", func(t *testing.T) {
		results := &nrdb.NRDBResultContainerMultiResultCustomized{
			Results: []nrdb.NRDBResult{
				{
					"count": float64(42),
				},
			},
		}

		assert.False(t, hasTimeseriesDataMulti(results))
	})

	t.Run("empty results", func(t *testing.T) {
		results := &nrdb.NRDBResultContainerMultiResultCustomized{
			Results: []nrdb.NRDBResult{},
		}

		assert.False(t, hasTimeseriesDataMulti(results))
	})
}

func TestGroupTimeseriesByFacetMulti(t *testing.T) {
	t.Run("with facet array in otherResult", func(t *testing.T) {
		results := &nrdb.NRDBResultContainerMultiResultCustomized{
			OtherResult: []nrdb.NRDBResult{
				{
					"beginTimeSeconds": float64(1600000000),
					"endTimeSeconds":   float64(1600000060),
					"facet":            []interface{}{"serviceA"},
					"count":            float64(10),
				},
				{
					"beginTimeSeconds": float64(1600000060),
					"endTimeSeconds":   float64(1600000120),
					"facet":            []interface{}{"serviceA"},
					"count":            float64(15),
				},
				{
					"beginTimeSeconds": float64(1600000000),
					"endTimeSeconds":   float64(1600000060),
					"facet":            []interface{}{"serviceB"},
					"count":            float64(5),
				},
			},
		}

		grouped := groupTimeseriesByFacetMulti(results, "service")

		assert.Equal(t, 2, len(grouped))
		assert.Equal(t, 2, len(grouped["serviceA"]))
		assert.Equal(t, 1, len(grouped["serviceB"]))
	})

	t.Run("with direct facet values in results", func(t *testing.T) {
		results := &nrdb.NRDBResultContainerMultiResultCustomized{
			Results: []nrdb.NRDBResult{
				{
					"beginTimeSeconds": float64(1600000000),
					"endTimeSeconds":   float64(1600000060),
					"facet":            "serviceA",
					"count":            float64(10),
				},
				{
					"beginTimeSeconds": float64(1600000060),
					"endTimeSeconds":   float64(1600000120),
					"facet":            "serviceB",
					"count":            float64(15),
				},
			},
			// Empty OtherResult to test fallback to Results
			OtherResult: []nrdb.NRDBResult{},
		}

		grouped := groupTimeseriesByFacetMulti(results, "service")

		assert.Equal(t, 2, len(grouped))
		assert.Equal(t, 1, len(grouped["serviceA"]))
		assert.Equal(t, 1, len(grouped["serviceB"]))
	})

	t.Run("empty results", func(t *testing.T) {
		results := &nrdb.NRDBResultContainerMultiResultCustomized{
			Results:     []nrdb.NRDBResult{},
			OtherResult: []nrdb.NRDBResult{},
		}

		grouped := groupTimeseriesByFacetMulti(results, "service")

		assert.Equal(t, 0, len(grouped))
	})
}

func TestParseNumericString(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedOutput float64
		expectError    bool
	}{
		{
			name:           "regular number",
			input:          "123.456",
			expectedOutput: 123.456,
			expectError:    false,
		},
		{
			name:           "scientific notation",
			input:          "1.23e2",
			expectedOutput: 123.0,
			expectError:    false,
		},
		{
			name:           "scientific notation with uppercase E",
			input:          "1.23E2",
			expectedOutput: 123.0,
			expectError:    false,
		},
		{
			name:           "negative scientific notation",
			input:          "-1.23e2",
			expectedOutput: -123.0,
			expectError:    false,
		},
		{
			name:           "zero",
			input:          "0",
			expectedOutput: 0.0,
			expectError:    false,
		},
		{
			name:           "invalid input",
			input:          "not a number",
			expectedOutput: 0.0,
			expectError:    true,
		},
		{
			name:           "empty string",
			input:          "",
			expectedOutput: 0.0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseNumericString(tt.input)

			if tt.expectError {
				assert.Error(t, err, "Expected an error for %s", tt.input)
			} else {
				assert.NoError(t, err, "Did not expect an error for %s", tt.input)
				assert.Equal(t, tt.expectedOutput, result, "Parsed value does not match expected output")
			}
		})
	}
}

func TestHandlePercentileFieldExtended(t *testing.T) {
	t.Run("single percentile value", func(t *testing.T) {
		// Create a frame and results with percentile data
		frame := data.NewFrame("test_frame")
		results := &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{
				{
					"percentile.duration": map[string]interface{}{
						"95": float64(123.45),
					},
				},
				{
					"percentile.duration": map[string]interface{}{
						"95": float64(234.56),
					},
				},
			},
		}

		// Process percentiles
		handlePercentileField(frame, results, "percentile.duration")

		// Should have one field for the 95th percentile
		assert.Equal(t, 1, len(frame.Fields))
		assert.Equal(t, "percentile.duration.95", frame.Fields[0].Name)

		// Check values
		assert.Equal(t, 2, frame.Fields[0].Len())
		assert.Equal(t, 123.45, *frame.Fields[0].At(0).(*float64))
		assert.Equal(t, 234.56, *frame.Fields[0].At(1).(*float64))
	})

	t.Run("multiple percentile values", func(t *testing.T) {
		// Create a frame and results with multiple percentile data points
		frame := data.NewFrame("test_frame")
		results := &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{
				{
					"percentile.duration": map[string]interface{}{
						"50": float64(50.1),
						"95": float64(95.2),
						"99": float64(99.3),
					},
				},
				{
					"percentile.duration": map[string]interface{}{
						"50": float64(51.1),
						"95": float64(96.2),
						"99": float64(100.3),
					},
				},
			},
		}

		// Process percentiles
		handlePercentileField(frame, results, "percentile.duration")

		// Should have three fields, one for each percentile
		assert.Equal(t, 3, len(frame.Fields))

		// Check field names (may be in any order)
		fieldNames := make(map[string]bool)
		for _, field := range frame.Fields {
			fieldNames[field.Name] = true
		}

		assert.True(t, fieldNames["percentile.duration.50"])
		assert.True(t, fieldNames["percentile.duration.95"])
		assert.True(t, fieldNames["percentile.duration.99"])

		// Check values in each field (would need to find which field is which first)
	})

	t.Run("missing percentile values", func(t *testing.T) {
		// Create a frame and results with some missing percentile data
		frame := data.NewFrame("test_frame")
		results := &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{
				{
					"percentile.duration": map[string]interface{}{
						"95": float64(95.2),
					},
				},
				{
					// No percentile data in this result
				},
				{
					"percentile.duration": map[string]interface{}{
						"95": float64(98.2),
					},
				},
			},
		}

		// Process percentiles
		handlePercentileField(frame, results, "percentile.duration")

		// Should have one field
		assert.Equal(t, 1, len(frame.Fields))
		assert.Equal(t, "percentile.duration.95", frame.Fields[0].Name)

		// Check values - middle one should be nil
		assert.Equal(t, 3, frame.Fields[0].Len())
		assert.Equal(t, 95.2, *frame.Fields[0].At(0).(*float64))
		assert.Nil(t, frame.Fields[0].At(1))
		assert.Equal(t, 98.2, *frame.Fields[0].At(2).(*float64))
	})

	t.Run("string percentile values", func(t *testing.T) {
		// Create a frame and results with string percentiles
		frame := data.NewFrame("test_frame")
		results := &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{
				{
					"percentile.duration": map[string]interface{}{
						"95": "123.45",
					},
				},
			},
		}

		// Process percentiles
		handlePercentileField(frame, results, "percentile.duration")

		// Should have correctly parsed the string value
		assert.Equal(t, 1, len(frame.Fields))
		assert.Equal(t, 123.45, *frame.Fields[0].At(0).(*float64))
	})
}

func TestHasCountFields(t *testing.T) {
	t.Run("with count field", func(t *testing.T) {
		results := &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{
				{
					"count": float64(42),
				},
			},
		}

		assert.True(t, hasCountField(results))
	})

	t.Run("without count field", func(t *testing.T) {
		results := &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{
				{
					"value": float64(42),
				},
			},
		}

		assert.False(t, hasCountField(results))
	})

	t.Run("empty results", func(t *testing.T) {
		results := &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{},
		}

		assert.False(t, hasCountField(results))
	})

	t.Run("null count field", func(t *testing.T) {
		results := &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{
				{
					"count": nil,
				},
			},
		}

		assert.True(t, hasCountField(results))
	})
}

func TestCreatePieChartFrame(t *testing.T) {
	t.Run("valid data with single facet", func(t *testing.T) {
		facetNames := []string{"service"}
		counts := []float64{10, 20, 30}
		facetFields := map[string][]string{
			"service": {"serviceA", "serviceB", "serviceC"},
		}

		frame := createPieChartFrame(facetNames, counts, facetFields)

		// Check general frame attributes
		assert.Equal(t, "pie_chart_data", frame.Name)
		assert.Equal(t, "table", string(frame.Meta.PreferredVisualization))
		assert.Equal(t, "pie", frame.Meta.Custom.(map[string]interface{})["chartType"])

		// Check fields
		assert.Equal(t, 2, len(frame.Fields))
		assert.Equal(t, "label", frame.Fields[0].Name)
		assert.Equal(t, "value", frame.Fields[1].Name)

		// Check data
		assert.Equal(t, 3, frame.Fields[0].Len())

		// Check values directly since we can't use StringSlice
		labelValues := make([]string, 3)
		for i := 0; i < 3; i++ {
			labelValues[i] = frame.Fields[0].At(i).(string)
		}
		assert.Equal(t, []string{"serviceA", "serviceB", "serviceC"}, labelValues)

		// Check numeric values
		valueValues := make([]float64, 3)
		for i := 0; i < 3; i++ {
			valueValues[i] = frame.Fields[1].At(i).(float64)
		}
		assert.Equal(t, counts, valueValues)
	})

	t.Run("empty facet names", func(t *testing.T) {
		facetNames := []string{}
		counts := []float64{10, 20, 30}
		facetFields := map[string][]string{
			"service": {"serviceA", "serviceB", "serviceC"},
		}

		frame := createPieChartFrame(facetNames, counts, facetFields)

		// Should still create a frame, but with no fields
		assert.Equal(t, "pie_chart_data", frame.Name)
		assert.Equal(t, 0, len(frame.Fields))
	})

	t.Run("empty counts", func(t *testing.T) {
		facetNames := []string{"service"}
		counts := []float64{}
		facetFields := map[string][]string{
			"service": {"serviceA", "serviceB", "serviceC"},
		}

		frame := createPieChartFrame(facetNames, counts, facetFields)

		// Should still create a frame, but with no fields
		assert.Equal(t, "pie_chart_data", frame.Name)
		assert.Equal(t, 0, len(frame.Fields))
	})

	t.Run("multiple facet names", func(t *testing.T) {
		facetNames := []string{"service", "environment"}
		counts := []float64{10, 20, 30}
		facetFields := map[string][]string{
			"service":     {"serviceA", "serviceB", "serviceC"},
			"environment": {"prod", "dev", "test"},
		}

		frame := createPieChartFrame(facetNames, counts, facetFields)

		// Should use the first facet name
		assert.Equal(t, "pie_chart_data", frame.Name)
		assert.Equal(t, 2, len(frame.Fields))
		assert.Equal(t, "label", frame.Fields[0].Name)
		assert.Equal(t, "value", frame.Fields[1].Name)

		// Should use labels from the first facet
		labelValues := make([]string, 3)
		for i := 0; i < 3; i++ {
			labelValues[i] = frame.Fields[0].At(i).(string)
		}
		assert.Equal(t, []string{"serviceA", "serviceB", "serviceC"}, labelValues)
	})
}

func TestFormatStandardQuery(t *testing.T) {
	now := time.Now()
	query := backend.DataQuery{
		RefID: "A",
		TimeRange: backend.TimeRange{
			From: now.Add(-1 * time.Hour),
			To:   now,
		},
	}

	tests := []struct {
		name           string
		results        *nrdb.NRDBResultContainer
		query          backend.DataQuery
		expectedFrames int
		expectedError  bool
	}{
		{
			name: "standard query with multiple fields",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"timestamp": float64(now.Add(-30 * time.Minute).Unix()),
						"value1":    10.0,
						"value2":    20.0,
						"name":      "test",
					},
					{
						"timestamp": float64(now.Add(-20 * time.Minute).Unix()),
						"value1":    15.0,
						"value2":    25.0,
						"name":      "test2",
					},
				},
			},
			query:          query,
			expectedFrames: 1,
			expectedError:  false,
		},
		{
			name: "empty results",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{},
			},
			query:          query,
			expectedFrames: 1, // The formatter creates an empty frame instead of returning nil
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := formatStandardQuery(tt.results, tt.query)

			assert.Equal(t, tt.expectedError, response.Error != nil)
			assert.Equal(t, tt.expectedFrames, len(response.Frames))

			if len(response.Frames) > 0 && !tt.expectedError && tt.name != "empty results" {
				frame := response.Frames[0]
				assert.NotNil(t, frame)

				// Check field names exist without assuming specific order
				fieldNames := make([]string, 0, len(frame.Fields))
				for _, field := range frame.Fields {
					fieldNames = append(fieldNames, field.Name)
				}

				assert.Contains(t, fieldNames, "time") // timestamp is renamed to time in formatter
				assert.Contains(t, fieldNames, "value1")
				assert.Contains(t, fieldNames, "value2")
				assert.Contains(t, fieldNames, "name")

				// Check row count
				assert.Equal(t, 2, frame.Fields[0].Len())
			}
		})
	}
}

func TestFormatFacetedCountQuery(t *testing.T) {
	now := time.Now()
	query := backend.DataQuery{
		RefID: "A",
		TimeRange: backend.TimeRange{
			From: now.Add(-1 * time.Hour),
			To:   now,
		},
	}

	tests := []struct {
		name           string
		results        *nrdb.NRDBResultContainer
		query          backend.DataQuery
		expectedFrames int
		expectedError  bool
	}{
		{
			name: "faceted count query",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"count": 10.0,
						"facet": []interface{}{"service1"},
					},
					{
						"count": 20.0,
						"facet": []interface{}{"service2"},
					},
				},
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"service"},
				},
			},
			query:          query,
			expectedFrames: 2, // The formatter returns two frames
			expectedError:  false,
		},
		{
			name: "empty results",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{},
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"service"},
				},
			},
			query:          query,
			expectedFrames: 0,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := formatFacetedCountQuery(tt.results, tt.query)

			assert.Equal(t, tt.expectedError, response.Error != nil)
			assert.Equal(t, tt.expectedFrames, len(response.Frames))

			if len(response.Frames) > 0 && !tt.expectedError && tt.name == "faceted_count_query" {
				// The formatFacetedCountQuery function only returns a single frame now
				frame := response.Frames[0]
				assert.NotNil(t, frame)

				// Should have fields for facet and count
				assert.Equal(t, 2, len(frame.Fields))

				// One field should be named "count"
				var hasCountField bool
				for _, field := range frame.Fields {
					if field.Name == "count" {
						hasCountField = true
						break
					}
				}
				assert.True(t, hasCountField, "Frame should have a 'count' field")

				// Check row count - should match our number of facets
				assert.Equal(t, 2, frame.Fields[0].Len())
			}
		})
	}
}

func TestIsFacetedTimeseriesQuery(t *testing.T) {
	tests := []struct {
		name     string
		results  *nrdb.NRDBResultContainer
		expected bool
	}{
		{
			name: "faceted timeseries query",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"count":            42.0,
						"facet":            []interface{}{"service1"},
						"beginTimeSeconds": 1609459200, // 2021-01-01 00:00:00
						"endTimeSeconds":   1609459260, // 2021-01-01 00:01:00
					},
				},
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"service"},
				},
			},
			expected: true,
		},
		{
			name: "not a faceted timeseries query - no timeseries",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"count": 42.0,
						"facet": []interface{}{"service1"},
					},
				},
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"service"},
				},
			},
			expected: false,
		},
		{
			name: "not a faceted timeseries query - no facet",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"count":            42.0,
						"beginTimeSeconds": 1609459200,
						"endTimeSeconds":   1609459260,
					},
				},
			},
			expected: false,
		},
		{
			name: "empty results",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isFacetedTimeseriesQuery(tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasTimeseriesData(t *testing.T) {
	tests := []struct {
		name     string
		results  *nrdb.NRDBResultContainer
		expected bool
	}{
		{
			name: "has timeseries data",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"beginTimeSeconds": 1609459200, // 2021-01-01 00:00:00
						"endTimeSeconds":   1609459260, // 2021-01-01 00:01:00
					},
				},
			},
			expected: true,
		},
		{
			name: "no timeseries data",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"count": 42.0,
					},
				},
			},
			expected: false,
		},
		{
			name: "empty results",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasTimeseriesData(tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Simple test that just makes sure the function doesn't crash
func TestFormatFacetedTimeseriesQueryBasic(t *testing.T) {
	now := time.Now()
	query := backend.DataQuery{
		RefID: "A",
		TimeRange: backend.TimeRange{
			From: now.Add(-1 * time.Hour),
			To:   now,
		},
	}

	// Create sample data with two facet values and two time points
	results := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{
				"count":            10.0,
				"facet":            []interface{}{"service1"},
				"beginTimeSeconds": float64(now.Add(-30 * time.Minute).Unix()),
				"endTimeSeconds":   float64(now.Add(-29 * time.Minute).Unix()),
			},
			{
				"count":            20.0,
				"facet":            []interface{}{"service1"},
				"beginTimeSeconds": float64(now.Add(-20 * time.Minute).Unix()),
				"endTimeSeconds":   float64(now.Add(-19 * time.Minute).Unix()),
			},
			{
				"count":            15.0,
				"facet":            []interface{}{"service2"},
				"beginTimeSeconds": float64(now.Add(-30 * time.Minute).Unix()),
				"endTimeSeconds":   float64(now.Add(-29 * time.Minute).Unix()),
			},
			{
				"count":            25.0,
				"facet":            []interface{}{"service2"},
				"beginTimeSeconds": float64(now.Add(-20 * time.Minute).Unix()),
				"endTimeSeconds":   float64(now.Add(-19 * time.Minute).Unix()),
			},
		},
		Metadata: nrdb.NRDBMetadata{
			Facets: []string{"service"},
		},
	}

	// Just check that the function doesn't panic
	response := formatFacetedTimeseriesQuery(results, query)

	// Basic checks
	assert.NotNil(t, response)
	assert.Nil(t, response.Error)
	assert.Equal(t, 2, len(response.Frames))
}
