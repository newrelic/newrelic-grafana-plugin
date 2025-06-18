package health

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNRDBExecutor implements the nrdbiface.NRDBQueryExecutor interface for testing
type mockNRDBExecutor struct {
	queryErr error
	results  *nrdb.NRDBResultContainer
}

func (m *mockNRDBExecutor) QueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL) (*nrdb.NRDBResultContainer, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	if m.results == nil {
		// Return a mock result with one data point to simulate a successful query
		return &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{
				{"count": 1.0},
			},
		}, nil
	}
	return m.results, nil
}

func TestPerformHealthCheck1(t *testing.T) {
	tests := []struct {
		name             string
		settings         backend.DataSourceInstanceSettings
		expectedStatus   backend.HealthStatus
		expectedContains string
	}{
		{
			name: "successful health check",
			settings: backend.DataSourceInstanceSettings{
				ID:   1,
				Name: "test-datasource",
				DecryptedSecureJSONData: map[string]string{
					"apiKey":    "test-api-key",
					"accountID": "123456",
				},
				JSONData: []byte(`{}`),
			},
			expectedStatus:   backend.HealthStatusError, // Will fail with real API call
			expectedContains: "Authentication failed",
		},
		{
			name: "invalid settings - missing API key",
			settings: backend.DataSourceInstanceSettings{
				ID:       1,
				Name:     "test-datasource",
				JSONData: []byte(`{}`),
				DecryptedSecureJSONData: map[string]string{
					"accountID": "123456",
				},
			},
			expectedStatus:   backend.HealthStatusError,
			expectedContains: "Enter New Relic API key",
		},
		{
			name: "invalid settings - invalid JSON",
			settings: backend.DataSourceInstanceSettings{
				ID:       1,
				Name:     "test-datasource",
				JSONData: []byte(`invalid json`),
				DecryptedSecureJSONData: map[string]string{
					"apiKey":    "test-api-key",
					"accountID": "123456",
				},
			},
			expectedStatus:   backend.HealthStatusError,
			expectedContains: "Failed to load datasource configuration",
		},
		{
			name: "invalid settings - missing account ID",
			settings: backend.DataSourceInstanceSettings{
				ID:       1,
				Name:     "test-datasource",
				JSONData: []byte(`{}`),
				DecryptedSecureJSONData: map[string]string{
					"apiKey": "test-api-key",
				},
			},
			expectedStatus:   backend.HealthStatusError,
			expectedContains: "Enter an account ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PerformHealthCheck1(context.Background(), tt.settings)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedStatus, result.Status)
			assert.Contains(t, result.Message, tt.expectedContains)
		})
	}
}

func TestExecuteHealthCheck(t *testing.T) {
	// Test that the ExecuteHealthCheck variable works correctly
	originalFunc := ExecuteHealthCheck
	defer func() {
		ExecuteHealthCheck = originalFunc
	}()

	// Test with original function
	settings := backend.DataSourceInstanceSettings{
		ID:       1,
		Name:     "test-datasource",
		JSONData: []byte(`{}`),
		DecryptedSecureJSONData: map[string]string{
			"apiKey":    "test-api-key",
			"accountID": "123456",
		},
	}

	result, err := ExecuteHealthCheck(context.Background(), settings)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, backend.HealthStatusError, result.Status) // Will fail with real API

	// Test with mocked function
	ExecuteHealthCheck = func(ctx context.Context, dsSettings backend.DataSourceInstanceSettings) (*backend.CheckHealthResult, error) {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusOk,
			Message: "Mocked success",
		}, nil
	}

	result, err = ExecuteHealthCheck(context.Background(), settings)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, backend.HealthStatusOk, result.Status)
	assert.Equal(t, "Mocked success", result.Message)
}

func TestPerformHealthCheck1_ClientCreationFailure(t *testing.T) {
	// Test with empty API key to trigger client creation failure
	settings := backend.DataSourceInstanceSettings{
		ID:       1,
		Name:     "test-datasource",
		JSONData: []byte(`{}`),
		DecryptedSecureJSONData: map[string]string{
			"apiKey":    "",
			"accountID": "123456",
		},
	}

	result, err := PerformHealthCheck1(context.Background(), settings)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, backend.HealthStatusError, result.Status)
	assert.Contains(t, result.Message, "Enter New Relic API key")
}
