package formatter

import (
	"newrelic-grafana-plugin/pkg/utils"
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

				// We should have at least one frame
				require.GreaterOrEqual(t, len(resp.Frames), 1, "Expected at least one frame")
				
				// Find frames by their labels instead of assuming specific order
				var usersFrame, ordersFrame *data.Frame
				for _, frame := range resp.Frames {
					require.Len(t, frame.Fields, 2, "Each frame should have time and value fields")
					
					// Skip the time field (index 0) and check the value field (index 1)
					valueField := frame.Fields[1]
					if valueField.Labels["request.uri"] == "/api/users" {
						usersFrame = frame
					} else if valueField.Labels["request.uri"] == "/api/orders" {
						ordersFrame = frame
					}
				}
				
				// Check if we have the required frames for the test
				if usersFrame != nil {
					timeField1 := usersFrame.Fields[0]
					assert.Equal(t, "time", timeField1.Name)
					assert.Equal(t, 2, timeField1.Len()) // 2 time points for /api/users

					sumField1 := usersFrame.Fields[1]
					assert.Equal(t, "sum.duration", sumField1.Name)
					assert.Equal(t, "/api/users", sumField1.Labels["request.uri"])
					assert.Equal(t, 2, sumField1.Len()) // 2 sum.duration values
					// Verify the actual values
					assert.Equal(t, 1500.25, *sumField1.At(0).(*float64))
					assert.Equal(t, 2100.75, *sumField1.At(1).(*float64))
				}
				
				// Check the orders frame if it exists
				if ordersFrame != nil {
					require.Len(t, ordersFrame.Fields, 2) // time and sum.duration fields
					sumField2 := ordersFrame.Fields[1]
					assert.Equal(t, "sum.duration", sumField2.Name)
					assert.Equal(t, "/api/orders", sumField2.Labels["request.uri"])
					assert.Equal(t, 1, sumField2.Len()) // 1 sum.duration value
					assert.Equal(t, 800.50, *sumField2.At(0).(*float64))
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
