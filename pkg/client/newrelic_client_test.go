package client

import (
	"errors"
	"testing"
	"time"

	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	"github.com/stretchr/testify/assert"
)

// MockNewRelicClientFactory is a mock implementation of NewRelicClientFactory
type MockNewRelicClientFactory struct {
	createClientFunc func(config ClientConfig) (*newrelic.NewRelic, error)
}

func (m *MockNewRelicClientFactory) CreateClient(config ClientConfig) (*newrelic.NewRelic, error) {
	return m.createClientFunc(config)
}

func TestGetClient(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		factory NewRelicClientFactory
		wantErr bool
	}{
		{
			name:    "valid api key",
			apiKey:  "valid-api-key",
			factory: &DefaultNewRelicClientFactory{},
			wantErr: false,
		},
		{
			name:    "empty api key",
			apiKey:  "",
			factory: &DefaultNewRelicClientFactory{},
			wantErr: true,
		},
		{
			name:   "factory error",
			apiKey: "valid-api-key",
			factory: &MockNewRelicClientFactory{
				createClientFunc: func(config ClientConfig) (*newrelic.NewRelic, error) {
					return nil, errors.New("factory error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := GetClient(tt.apiKey, tt.factory)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestNewRelicClientError(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		err     error
		wantMsg string
	}{
		{
			name:    "with wrapped error",
			msg:     "test error",
			err:     errors.New("wrapped error"),
			wantMsg: "new relic client error: test error: wrapped error",
		},
		{
			name:    "without wrapped error",
			msg:     "test error",
			err:     nil,
			wantMsg: "new relic client error: test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &NewRelicClientError{
				Msg: tt.msg,
				Err: tt.err,
			}
			assert.Equal(t, tt.wantMsg, err.Error())
			assert.Equal(t, tt.err, err.Unwrap())
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, "US", config.Region)
	assert.Equal(t, "newrelic-grafana-plugin", config.UserAgent)
	assert.IsType(t, time.Duration(0), config.Timeout)
	assert.Greater(t, config.Timeout, time.Duration(0))
}

func TestCreateClient(t *testing.T) {
	tests := []struct {
		name    string
		config  ClientConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ClientConfig{
				APIKey:    "valid-api-key",
				Region:    "US",
				Timeout:   30 * time.Second,
				UserAgent: "test-user-agent",
			},
			wantErr: false,
		},
		{
			name: "empty api key",
			config: ClientConfig{
				APIKey:    "",
				Region:    "US",
				Timeout:   30 * time.Second,
				UserAgent: "test-user-agent",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := &DefaultNewRelicClientFactory{}
			client, err := factory.CreateClient(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}
