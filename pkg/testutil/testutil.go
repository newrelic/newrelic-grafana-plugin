package testutil

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

// MockTimeNow returns a fixed time for testing
func MockTimeNow() time.Time {
	return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
}

// CreateTestQuery creates a test query with the given refID and query string
func CreateTestQuery(t *testing.T, refID string, query string) backend.DataQuery {
	t.Helper()

	// Create proper JSON structure for the query
	queryJSON := map[string]interface{}{
		"queryText": query,
	}

	jsonBytes, err := json.Marshal(queryJSON)
	require.NoError(t, err)

	return backend.DataQuery{
		RefID:     refID,
		QueryType: "nrql",
		JSON:      jsonBytes,
		TimeRange: backend.TimeRange{
			From: time.Now().Add(-1 * time.Hour),
			To:   time.Now(),
		},
	}
}

// CreateTestSettings creates test datasource settings
func CreateTestSettings(t *testing.T, region string) *backend.DataSourceInstanceSettings {
	t.Helper()
	return &backend.DataSourceInstanceSettings{
		JSONData: []byte(`{
			"region": "` + region + `",
			"secureJsonData": {
				"apiKey": "test-api-key",
				"accountId": "123456"
			}
		}`),
	}
}

// AssertFrameFields checks if a data frame has the expected fields
func AssertFrameFields(t *testing.T, frame *data.Frame, expectedFields []string) {
	t.Helper()

	require.Equal(t, len(expectedFields), len(frame.Fields), "number of fields")
	for i, field := range frame.Fields {
		require.Equal(t, expectedFields[i], field.Name, "field name")
	}
}

// CreateTestPluginContext creates a test plugin context
func CreateTestPluginContext(t *testing.T, settings *backend.DataSourceInstanceSettings) backend.PluginContext {
	t.Helper()
	return backend.PluginContext{
		DataSourceInstanceSettings: settings,
	}
}
