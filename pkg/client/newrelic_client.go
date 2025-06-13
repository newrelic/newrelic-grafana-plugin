package client

import (
	"fmt"

	"github.com/newrelic/newrelic-client-go/v2/newrelic"
)

// NewRelicClientError represents an error specifically related to New Relic client operations.
type NewRelicClientError struct {
	Msg string
	Err error // Wrapped error
}

func (e *NewRelicClientError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("new relic client error: %s: %v", e.Msg, e.Err)
	}
	return fmt.Sprintf("new relic client error: %s", e.Msg)
}

func (e *NewRelicClientError) Unwrap() error {
	return e.Err
}

// NewRelicClientFactory defines an interface for creating New Relic clients.
// This allows us to inject a mock factory during testing.
type NewRelicClientFactory interface {
	CreateClient(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error)
}

// DefaultNewRelicClientFactory is the concrete implementation that uses the actual newrelic.New function.
type DefaultNewRelicClientFactory struct{}

// CreateClient implements the NewRelicClientFactory interface using the actual newrelic.New function.
func (f *DefaultNewRelicClientFactory) CreateClient(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error) {
	return newrelic.New(opts...)
}

// GetClient initializes and returns a New Relic client using the provided API key
// and a NewRelicClientFactory.
// The factory argument makes this function testable.
func GetClient(apiKey string, factory NewRelicClientFactory) (*newrelic.NewRelic, error) {
	if apiKey == "" {
		return nil, &NewRelicClientError{Msg: "New Relic API key cannot be empty"}
	}

	// Use the provided factory to create the client
	client, err := factory.CreateClient(newrelic.ConfigPersonalAPIKey(apiKey))
	if err != nil {
		return nil, &NewRelicClientError{Msg: "failed to initialize New Relic client", Err: err}
	}
	return client, nil
}
