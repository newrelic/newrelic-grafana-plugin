// Package client provides the New Relic API client implementation.
// It handles authentication, connection management, and API interactions.
package client

import (
	"fmt"

	"github.com/newrelic/newrelic-client-go/v2/newrelic"
)

// ClientError represents an error specifically related to New Relic client operations.
type ClientError struct {
	Msg string
	Err error // Wrapped error
}

func (e *ClientError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("new relic client error: %s: %v", e.Msg, e.Err)
	}
	return fmt.Sprintf("new relic client error: %s", e.Msg)
}

func (e *ClientError) Unwrap() error {
	return e.Err
}

// ClientFactory defines an interface for creating New Relic clients.
// This allows for dependency injection and better testing.
type ClientFactory interface {
	CreateClient(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error)
}

// DefaultClientFactory is the concrete implementation that uses the actual newrelic.New function.
type DefaultClientFactory struct{}

// CreateClient implements the ClientFactory interface using the actual newrelic.New function.
func (f *DefaultClientFactory) CreateClient(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error) {
	return newrelic.New(opts...)
}

// NewClient initializes and returns a New Relic client using the provided API key
// and a ClientFactory.
func NewClient(apiKey string, factory ClientFactory) (*newrelic.NewRelic, error) {
	if apiKey == "" {
		return nil, &ClientError{Msg: "New Relic API key cannot be empty"}
	}

	client, err := factory.CreateClient(newrelic.ConfigPersonalAPIKey(apiKey))
	if err != nil {
		return nil, &ClientError{Msg: "failed to initialize New Relic client", Err: err}
	}
	return client, nil
}
