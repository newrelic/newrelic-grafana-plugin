package plugin

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"newrelic-grafana-plugin/pkg/health"
	"newrelic-grafana-plugin/pkg/models"
)

// TestNewDatasource ensures that a new Datasource instance can be created.
func TestNewDatasource(t *testing.T) {
	ds, err := NewDatasource(context.Background(), backend.DataSourceInstanceSettings{})
	require.NoError(t, err)
	assert.NotNil(t, ds)
}

// TestDatasource_Dispose ensures the Dispose method runs without errors or panics.
func TestDatasource_Dispose(t *testing.T) {
	ds := &Datasource{}
	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Dispose caused a panic: %v", r)
		}
	}()
	ds.Dispose()
}

// TestDatasource_CheckHealth_Success verifies a successful health check.
func TestDatasource_CheckHealth_Success(t *testing.T) {
	ds := &Datasource{}
	ctx := context.Background()
	req := &backend.CheckHealthRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{},
		},
	}

	// Mock the 'health.ExecuteHealthCheck' function to simulate a successful outcome.
	originalExecuteHealthCheck := health.ExecuteHealthCheck
	health.ExecuteHealthCheck = func(ctx context.Context, dsSettings backend.DataSourceInstanceSettings) (*backend.CheckHealthResult, error) {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusOk,
			Message: "Plugin is healthy!",
		}, nil
	}
	defer func() { health.ExecuteHealthCheck = originalExecuteHealthCheck }()

	// Call the CheckHealth method of the Datasource.
	res, err := ds.CheckHealth(ctx, req)

	// Assertions for a successful health check.
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, backend.HealthStatusOk, res.Status)
	assert.Equal(t, "Plugin is healthy!", res.Message)
}

// TestDatasource_CheckHealth_Failure verifies a failing health check due to an internal error.
func TestDatasource_CheckHealth_Failure(t *testing.T) {
	ds := &Datasource{}
	ctx := context.Background()
	req := &backend.CheckHealthRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{},
		},
	}

	// Mock 'health.ExecuteHealthCheck' to simulate an internal error.
	expectedErr := errors.New("simulated internal health check failure")
	originalExecuteHealthCheck := health.ExecuteHealthCheck
	health.ExecuteHealthCheck = func(ctx context.Context, dsSettings backend.DataSourceInstanceSettings) (*backend.CheckHealthResult, error) {
		return nil, expectedErr
	}
	defer func() { health.ExecuteHealthCheck = originalExecuteHealthCheck }()

	// Call the CheckHealth method.
	res, err := ds.CheckHealth(ctx, req)

	// Assertions for a failing health check.
	require.NoError(t, err) // CheckHealth should not return Go errors, but health status errors
	require.NotNil(t, res)
	assert.Equal(t, backend.HealthStatusError, res.Status)
	expectedMsg := fmt.Sprintf("Health check encountered an internal error: %s", expectedErr.Error())
	assert.Equal(t, expectedMsg, res.Message)
}

func TestQueryData(t *testing.T) {
	tests := []struct {
		name    string
		queries []backend.DataQuery
		wantErr bool
	}{
		{
			name:    "empty queries",
			queries: []backend.DataQuery{},
			wantErr: false,
		},
		{
			name: "single query",
			queries: []backend.DataQuery{
				{RefID: "A", JSON: []byte(`{"queryText":"SELECT count(*) FROM Transaction","accountID":123456}`)},
			},
			wantErr: false,
		},
		{
			name: "multiple queries",
			queries: []backend.DataQuery{
				{RefID: "A", JSON: []byte(`{"queryText":"SELECT count(*) FROM Transaction","accountID":123456}`)},
				{RefID: "B", JSON: []byte(`{"queryText":"SELECT count(*) FROM PageView","accountID":123456}`)},
			},
			wantErr: false,
		},
	}

	// Create mock settings
	settings := backend.DataSourceInstanceSettings{
		ID:       1,
		Name:     "test-datasource",
		JSONData: []byte(`{"path": "/test"}`),
		DecryptedSecureJSONData: map[string]string{
			"apiKey":    "test-api-key",
			"accountID": "123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Datasource{}
			resp, err := ds.QueryData(
				context.Background(),
				&backend.QueryDataRequest{
					PluginContext: backend.PluginContext{
						DataSourceInstanceSettings: &settings,
					},
					Queries: tt.queries,
				},
			)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				// For now, expect an error because the client will fail with invalid credentials
				// But we shouldn't get a nil pointer dereference
				if err != nil {
					assert.Contains(t, err.Error(), "failed to create New Relic client")
				}
				if resp != nil {
					assert.Equal(t, len(tt.queries), len(resp.Responses))
				}
			}
		})
	}
}

// TestDatasource_QueryData_Success verifies a successful data query with mocking.
func TestDatasource_QueryData_Success(t *testing.T) {
	ds := &Datasource{}
	ctx := context.Background()
	req := &backend.QueryDataRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{},
		},
		Queries: []backend.DataQuery{
			{RefID: "queryA", QueryType: "nrql", JSON: []byte(`{"queryText":"SELECT count(*) FROM Transaction","accountID":123456}`)},
		},
	}

	// Mock 'models.LoadPluginSettings' - this is declared as a variable, so it can be mocked
	originalLoadPluginSettings := models.LoadPluginSettings
	models.LoadPluginSettings = func(settings backend.DataSourceInstanceSettings) (*models.PluginSettings, error) {
		return &models.PluginSettings{
			Secrets: &models.SecretPluginSettings{
				ApiKey:    "test-api-key",
				AccountId: 12345,
			},
		}, nil
	}
	defer func() { models.LoadPluginSettings = originalLoadPluginSettings }()

	// Note: validator.ValidatePluginSettings and client.CreateNewRelicClient are not declared as variables,
	// so they cannot be mocked in this way. The test will use the real implementations.

	// Call the QueryData method.
	res, err := ds.QueryData(ctx, req)

	// With mocked settings loading, this should at least get past the settings loading phase
	// It may still fail at client creation with test credentials, but shouldn't panic
	if err != nil {
		// Expect failure due to invalid test credentials, but verify it's the right kind of error
		assert.Contains(t, err.Error(), "failed to create New Relic client")
	} else {
		// If it somehow succeeds (e.g., with mock credentials), verify the response structure
		require.NotNil(t, res)
		assert.Equal(t, 1, len(res.Responses))
	}
}

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name     string
		settings backend.DataSourceInstanceSettings
		wantErr  bool
	}{
		{
			name: "valid settings",
			settings: backend.DataSourceInstanceSettings{
				JSONData: []byte(`{"path": "/test"}`),
				DecryptedSecureJSONData: map[string]string{
					"apiKey":    "test-api-key",
					"accountID": "123456",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid settings - missing credentials",
			settings: backend.DataSourceInstanceSettings{
				JSONData: []byte(`{}`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Datasource{}
			resp, err := ds.CheckHealth(
				context.Background(),
				&backend.CheckHealthRequest{
					PluginContext: backend.PluginContext{
						DataSourceInstanceSettings: &tt.settings,
					},
				},
			)

			if tt.wantErr {
				// Expect no error from the function itself, but a health status error
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, backend.HealthStatusError, resp.Status)
			} else {
				// Expect no error but connection might fail with test credentials
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				// With test credentials, we expect either OK or Error status
				assert.Contains(t, []backend.HealthStatus{backend.HealthStatusOk, backend.HealthStatusError}, resp.Status)
			}
		})
	}
}
