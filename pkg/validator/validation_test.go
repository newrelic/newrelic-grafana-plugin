package validator

import (
	"context"
	"errors"
	"testing"

	"newrelic-grafana-plugin/pkg/models"
	"newrelic-grafana-plugin/pkg/nrdbiface"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	nrErrors "github.com/newrelic/newrelic-client-go/v2/pkg/errors"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePluginSettings(t *testing.T) {
	tests := []struct {
		name    string
		config  *models.PluginSettings
		wantErr bool
	}{
		{
			name: "valid settings",
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					ApiKey:    "test-key",
					AccountId: 123456,
				},
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "nil secrets",
			config: &models.PluginSettings{
				Secrets: nil,
			},
			wantErr: true,
		},
		{
			name: "empty API key",
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					ApiKey:    "",
					AccountId: 123456,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid account ID",
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					ApiKey:    "test-key",
					AccountId: 0,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePluginSettings(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

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

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name     string
		config   *models.PluginSettings
		executor nrdbiface.NRDBQueryExecutor
		want     *backend.CheckHealthResult
		wantErr  bool
	}{
		{
			name: "successful health check",
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					ApiKey:    "test-key",
					AccountId: 123456,
				},
			},
			executor: &mockNRDBExecutor{},
			want: &backend.CheckHealthResult{
				Status:  backend.HealthStatusOk,
				Message: "✅ New Relic connection successful (Account ID: 123456)",
			},
			wantErr: false,
		},
		{
			name: "unauthorized error",
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					ApiKey:    "test-key",
					AccountId: 123456,
				},
			},
			executor: &mockNRDBExecutor{
				queryErr: &nrErrors.UnauthorizedError{},
			},
			want: &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: "Authentication failed for account ID 123456. Please verify your API key is correct and has access to this account.",
			},
			wantErr: false,
		},
		{
			name: "other error",
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					ApiKey:    "test-key",
					AccountId: 123456,
				},
			},
			executor: &mockNRDBExecutor{
				queryErr: errors.New("connection error"),
			},
			want: &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: "Failed to connect to New Relic API (Account ID: 123456). Error: connection error",
			},
			wantErr: false,
		},
		{
			name: "empty results",
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					ApiKey:    "test-key",
					AccountId: 123456,
				},
			},
			executor: &mockNRDBExecutor{
				results: &nrdb.NRDBResultContainer{Results: []nrdb.NRDBResult{}},
			},
			want: &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: "New Relic API returned an empty response for account ID 123456. Please verify the account has data or the account ID is correct.",
			},
			wantErr: false,
		},
		{
			name: "nil executor",
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					ApiKey:    "test-key",
					AccountId: 123456,
				},
			},
			executor: nil,
			want: &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: "NRDB query executor is not initialized for health check.",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckHealth(context.Background(), tt.config, tt.executor)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.Status, got.Status)
			assert.Equal(t, tt.want.Message, got.Message)
		})
	}
}
