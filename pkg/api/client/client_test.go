package client

import (
	"errors"
	"testing"

	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	"github.com/stretchr/testify/assert"
)

// MockClientFactory implements the ClientFactory interface for testing
type MockClientFactory struct {
	shouldFail bool
	client     *newrelic.NewRelic
	err        error
}

func (m *MockClientFactory) CreateClient(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error) {
	if m.shouldFail {
		return nil, m.err
	}
	return m.client, nil
}

func TestNewClient(t *testing.T) {
	mockClient := &newrelic.NewRelic{}

	tests := []struct {
		name      string
		apiKey    string
		factory   ClientFactory
		expectErr bool
		errMsg    string
	}{
		{
			name:      "success",
			apiKey:    "valid-key",
			factory:   &MockClientFactory{shouldFail: false, client: mockClient},
			expectErr: false,
		},
		{
			name:      "empty API key",
			apiKey:    "",
			factory:   &MockClientFactory{},
			expectErr: true,
			errMsg:    "New Relic API key cannot be empty",
		},
		{
			name:      "client creation error",
			apiKey:    "valid-key",
			factory:   &MockClientFactory{shouldFail: true, err: errors.New("factory error")},
			expectErr: true,
			errMsg:    "failed to initialize New Relic client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.apiKey, tt.factory)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, mockClient, client)
			}
		})
	}
}

func TestClientError(t *testing.T) {
	tests := []struct {
		name     string
		err      *ClientError
		expected string
	}{
		{
			name: "with wrapped error",
			err: &ClientError{
				Msg: "test message",
				Err: errors.New("wrapped error"),
			},
			expected: "new relic client error: test message: wrapped error",
		},
		{
			name: "without wrapped error",
			err: &ClientError{
				Msg: "test message",
			},
			expected: "new relic client error: test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
			assert.Equal(t, tt.err.Err, tt.err.Unwrap())
		})
	}
}

func TestDefaultClientFactory(t *testing.T) {
	// Skip this test since it requires actual connection to New Relic
	t.Skip("Skipping test that requires network connection")

	factory := &DefaultClientFactory{}

	opts := []newrelic.ConfigOption{
		newrelic.ConfigPersonalAPIKey("invalid-key"),
	}

	client, err := factory.CreateClient(opts...)
	assert.Error(t, err)
	assert.Nil(t, client)
}
