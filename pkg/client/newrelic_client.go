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
type NewRelicClientFactory interface {
	CreateClient(config ClientConfig) (*newrelic.NewRelic, error)
}

// DefaultNewRelicClientFactory is the concrete implementation that uses the actual newrelic.New function.
type DefaultNewRelicClientFactory struct{}

// CreateClient implements the NewRelicClientFactory interface using the actual newrelic.New function.
func (f *DefaultNewRelicClientFactory) CreateClient(config ClientConfig) (*newrelic.NewRelic, error) {
	if config.APIKey == "" {
		return nil, &NewRelicClientError{Msg: "New Relic API key cannot be empty"}
	}

	opts := []newrelic.ConfigOption{
		newrelic.ConfigPersonalAPIKey(config.APIKey),
		newrelic.ConfigRegion(config.Region),
		newrelic.ConfigUserAgent(config.UserAgent),
	}

	client, err := newrelic.New(opts...)
	if err != nil {
		return nil, &NewRelicClientError{Msg: "failed to initialize New Relic client", Err: err}
	}
	return client, nil
}

// GetClient initializes and returns a New Relic client using the provided configuration
// and a NewRelicClientFactory.
func GetClient(apiKey string, factory NewRelicClientFactory) (*newrelic.NewRelic, error) {
	config := DefaultConfig()
	config.APIKey = apiKey
	return factory.CreateClient(config)
}
