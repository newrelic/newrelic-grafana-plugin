package client

import (
	"github.com/newrelic/newrelic-client-go/v2/newrelic"
)

// NewRelicClientFactory defines an interface for creating New Relic clients.
// This interface is crucial for enabling dependency injection, especially for testing
// scenarios where you might want to mock the New Relic client creation process
// without making actual API calls.
type NewRelicClientFactory interface {
	// CreateClient takes New Relic configuration options and returns a pointer
	// to a NewRelic client instance or an error if initialization fails.
	CreateClient(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error)
}

// DefaultNewRelicClientFactory is the concrete implementation of the
// NewRelicClientFactory interface. It uses the actual newrelic.New function
// to create a live New Relic client.
type DefaultNewRelicClientFactory struct{}

// CreateClient implements the NewRelicClientFactory interface for
// DefaultNewRelicClientFactory. It directly calls the New Relic Go client's
// constructor to instantiate a new client with the given options.
func (f *DefaultNewRelicClientFactory) CreateClient(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error) {
	// Use the newrelicNewFunc variable to create a New Relic client.
	// This allows for easier testing by mocking the newrelicNewFunc.
	return newrelicNewFunc(opts...)
}
