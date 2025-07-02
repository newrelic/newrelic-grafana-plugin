package client

import (
	"errors"
	"testing"

	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultNewRelicClientFactory_CreateClient tests the DefaultNewRelicClientFactory
func TestDefaultNewRelicClientFactory_CreateClient(t *testing.T) {
	// Save the original newrelic.New function and restore it after the test
	originalNewFunc := newrelicNewFunc
	defer func() { newrelicNewFunc = originalNewFunc }()

	tests := []struct {
		name    string
		mockFn  func(...newrelic.ConfigOption) (*newrelic.NewRelic, error)
		wantErr bool
	}{
		{
			name: "successful client creation",
			mockFn: func(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error) {
				// Mock successful creation
				return &newrelic.NewRelic{}, nil
			},
			wantErr: false,
		},
		{
			name: "error during client creation",
			mockFn: func(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error) {
				// Mock an error
				return nil, errors.New("failed to create New Relic client")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override the newrelic.New function with our mock for this test
			newrelicNewFunc = tt.mockFn

			// Create the factory
			factory := &DefaultNewRelicClientFactory{}

			// Call the CreateClient method with dummy config options
			client, err := factory.CreateClient(
				newrelic.ConfigPersonalAPIKey("dummy-key"),
				newrelic.ConfigRegion("US"),
			)

			// Assert the results
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
				assert.Contains(t, err.Error(), "failed to create New Relic client")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

// TestNewRelicClientFactoryInterface verifies that DefaultNewRelicClientFactory
// correctly implements the NewRelicClientFactory interface
func TestNewRelicClientFactoryInterface(t *testing.T) {
	// Static type assertion
	var _ NewRelicClientFactory = (*DefaultNewRelicClientFactory)(nil)

	// Runtime verification
	factory := &DefaultNewRelicClientFactory{}
	require.Implements(t, (*NewRelicClientFactory)(nil), factory)
}
