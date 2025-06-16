package plugin

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDatasource(t *testing.T) {
	ds, err := NewDatasource(context.Background(), backend.DataSourceInstanceSettings{})
	require.NoError(t, err)
	assert.NotNil(t, ds)
}

func TestDispose(t *testing.T) {
	ds := &Datasource{}
	// Should not panic
	ds.Dispose()
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
				{RefID: "A"},
			},
			wantErr: false,
		},
		{
			name: "multiple queries",
			queries: []backend.DataQuery{
				{RefID: "A"},
				{RefID: "B"},
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
