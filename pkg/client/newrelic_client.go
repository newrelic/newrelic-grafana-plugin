package client

import (
	"fmt"
	"time"

	"github.com/newrelic/newrelic-client-go/v2/newrelic"
)

// ClientConfig holds configuration options for the New Relic client
type ClientConfig struct {
	APIKey     string
	Region     string
	Timeout    time.Duration
	RetryCount int
	RetryDelay time.Duration
	UserAgent  string
}

// DefaultConfig returns a ClientConfig with sensible defaults
func DefaultConfig() ClientConfig {
	return ClientConfig{
		Region:     "US",
		Timeout:    30 * time.Second,
		RetryCount: 3,
		RetryDelay: 1 * time.Second,
		UserAgent:  "newrelic-grafana-plugin",
	}
}

// NewRelicClientError represents an error specifically related to New Relic client operations.
type NewRelicClientError struct {
	Msg string
	Err error
}

// Error implements the error interface for NewRelicClientError.
func (e *NewRelicClientError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("new relic client error: %s: %v", e.Msg, e.Err)
	}
	return fmt.Sprintf("new relic client error: %s", e.Msg)
}

// Unwrap returns the wrapped error, if any, allowing for error chain inspection.
func (e *NewRelicClientError) Unwrap() error {
	return e.Err
}

// CreateNewRelicClient initializes and returns a New Relic client using the provided API key
// and a NewRelicClientFactory.
//
// The factory argument makes this function easily testable by allowing
// injection of mock client factories during unit tests.
func CreateNewRelicClient(apiKey string, factory NewRelicClientFactory) (*newrelic.NewRelic, error) {
	// Validate that the API key is not empty.
	if apiKey == "" {
		return nil, &NewRelicClientError{Msg: "New Relic API key cannot be empty"}
	}

	// Use the provided factory to create the New Relic client.
	// This delegates the actual client initialization logic, separating concerns.
	client, err := factory.CreateClient(newrelic.ConfigPersonalAPIKey(apiKey))
	if err != nil {
		// Wrap any error from the factory in a NewRelicClientError for consistent error handling.
		return nil, &NewRelicClientError{Msg: "failed to initialize New Relic client", Err: err}
	}
	return client, nil
}

// GetClient initializes and returns a New Relic client using the provided configuration
// and a NewRelicClientFactory. This provides more configuration options than CreateNewRelicClient.
func GetClient(config ClientConfig, factory NewRelicClientFactory) (*newrelic.NewRelic, error) {
	if config.APIKey == "" {
		return nil, &NewRelicClientError{Msg: "New Relic API key cannot be empty"}
	}

	opts := []newrelic.ConfigOption{
		newrelic.ConfigPersonalAPIKey(config.APIKey),
		newrelic.ConfigRegion(config.Region),
		newrelic.ConfigUserAgent(config.UserAgent),
	}

	client, err := factory.CreateClient(opts...)
	if err != nil {
		return nil, &NewRelicClientError{Msg: "failed to initialize New Relic client", Err: err}
	}
	return client, nil
}
