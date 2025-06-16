package validator

import (
	"context"
	"errors"
	"testing"

	"newrelic-grafana-plugin/pkg/models"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	nrErrors "github.com/newrelic/newrelic-client-go/v2/pkg/errors"
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

type mockNRClient struct {
	queryErr error
}

func (m *mockNRClient) Nrdb() *newrelic.Nrdb {
	return &newrelic.Nrdb{}
}

func (m *mockNRClient) QueryWithContext(ctx context.Context, accountID int, query string) (*newrelic.NrdbResult, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return &newrelic.NrdbResult{}, nil
}

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name    string
		config  *models.PluginSettings
		client  *mockNRClient
		want    *backend.CheckHealthResult
		wantErr bool
	}{
		{
			name: "successful health check",
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					ApiKey:    "test-key",
					AccountId: 123456,
				},
			},
			client: &mockNRClient{},
			want: &backend.CheckHealthResult{
				Status:  backend.HealthStatusOk,
				Message: "Successfully connected to New Relic API.",
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
			client: &mockNRClient{
				queryErr: &nrErrors.UnauthorizedError{},
			},
			want: &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: "An error occurred with connecting to NewRelic.Could not connect to NewRelic. This usually happens when the API key is incorrect.",
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
			client: &mockNRClient{
				queryErr: errors.New("connection error"),
			},
			want: &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: "Failed to connect to New Relic API or authenticate. Error: connection error",
			},
			wantErr: false,
		},
		{
			name: "nil client",
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					ApiKey:    "test-key",
					AccountId: 123456,
				},
			},
			client: nil,
			want: &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: "New Relic client is not initialized for health check.",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckHealth(context.Background(), tt.config, tt.client)
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
