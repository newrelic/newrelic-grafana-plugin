package config

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadSettings(t *testing.T) {
	tests := []struct {
		name      string
		source    backend.DataSourceInstanceSettings
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid settings",
			source: backend.DataSourceInstanceSettings{
				JSONData: []byte(`{"path": "/test/path"}`),
				DecryptedSecureJSONData: map[string]string{
					"apiKey":    "test-api-key",
					"accountID": "123456",
				},
			},
			expectErr: false,
		},
		{
			name: "invalid JSON",
			source: backend.DataSourceInstanceSettings{
				JSONData: []byte(`invalid json`),
				DecryptedSecureJSONData: map[string]string{
					"apiKey":    "test-api-key",
					"accountID": "123456",
				},
			},
			expectErr: true,
			errMsg:    "could not unmarshal Settings JSON",
		},
		{
			name: "missing API key",
			source: backend.DataSourceInstanceSettings{
				JSONData: []byte(`{"path": "/test/path"}`),
				DecryptedSecureJSONData: map[string]string{
					"accountID": "123456",
				},
			},
			expectErr: true,
			errMsg:    "Enter New Relic API key",
		},
		{
			name: "missing account ID",
			source: backend.DataSourceInstanceSettings{
				JSONData: []byte(`{"path": "/test/path"}`),
				DecryptedSecureJSONData: map[string]string{
					"apiKey": "test-api-key",
				},
			},
			expectErr: true,
			errMsg:    "Enter an account ID",
		},
		{
			name: "invalid account ID",
			source: backend.DataSourceInstanceSettings{
				JSONData: []byte(`{"path": "/test/path"}`),
				DecryptedSecureJSONData: map[string]string{
					"apiKey":    "test-api-key",
					"accountID": "not-a-number",
				},
			},
			expectErr: true,
			errMsg:    "could not convert accountID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings, err := LoadSettings(tt.source)
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, settings)
			} else {
				require.NoError(t, err)
				require.NotNil(t, settings)
				assert.Equal(t, "/test/path", settings.Path)
				assert.Equal(t, "test-api-key", settings.Secrets.ApiKey)
				assert.Equal(t, 123456, settings.Secrets.AccountId)
			}
		})
	}
}

func TestSettingsError(t *testing.T) {
	tests := []struct {
		name     string
		err      *SettingsError
		expected string
	}{
		{
			name: "with message and wrapped error",
			err: &SettingsError{
				Msg: "test message",
				Err: assert.AnError,
			},
			expected: "test message: assert.AnError general error for testing",
		},
		{
			name: "with wrapped error only",
			err: &SettingsError{
				Err: assert.AnError,
			},
			expected: "assert.AnError general error for testing",
		},
		{
			name: "with message only",
			err: &SettingsError{
				Msg: "test message",
			},
			expected: " test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
			if tt.err.Err != nil {
				assert.Equal(t, tt.err.Err, tt.err.Unwrap())
			} else {
				assert.Nil(t, tt.err.Unwrap())
			}
		})
	}
}
