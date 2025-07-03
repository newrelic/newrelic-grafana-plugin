package client

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/newrelic/newrelic-client-go/v2/newrelic"
)

var (
	serviceName = "newrelic-grafana-plugin"
	// newrelicNewFunc is a variable that holds the function to create a New Relic client
	// This allows for mocking in tests
	newrelicNewFunc = newrelic.New

	// NewrelicNewFunc is an exported version of newrelicNewFunc for testing purposes
	// This allows other packages to mock the New Relic client creation for tests
	NewrelicNewFunc = newrelicNewFunc
)

// ClientConfig holds configuration options for the New Relic client
type ClientConfig struct {
	APIKey        string
	Region        string
	Timeout       time.Duration
	RetryCount    int
	RetryDelay    time.Duration
	UserAgent     string
	DatasourceUID string // New field for datasource UID
}

// DefaultConfig returns a ClientConfig with sensible defaults
func DefaultConfig() ClientConfig {
	return ClientConfig{
		Region:     "US",
		Timeout:    30 * time.Second,
		RetryCount: 3,
		RetryDelay: 1 * time.Second,
		UserAgent:  serviceName,
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

// NewClient creates a New Relic client with the specified configuration.
// This is the recommended way to create a client.
func NewClient(config ClientConfig) (*newrelic.NewRelic, error) {
	log.DefaultLogger.Debug("NewRelicClient: Initializing New Relic client with configuration")
	if config.APIKey == "" {
		return nil, &NewRelicClientError{Msg: "New Relic API key cannot be empty"}
	}

	// Create service name with UID if provided
	clientServiceName := serviceName
	if config.DatasourceUID != "" {
		clientServiceName = fmt.Sprintf("%s-%s", serviceName, config.DatasourceUID)
		log.DefaultLogger.Debug("NewRelicClient: Using UID-specific service name", "serviceName", clientServiceName, "uid", config.DatasourceUID)
	} else {
		log.DefaultLogger.Debug("NewRelicClient: Using default service name", "serviceName", clientServiceName)
	}

	// Setup configuration options
	cfgOpts := []newrelic.ConfigOption{
		newrelic.ConfigPersonalAPIKey(config.APIKey),
		newrelic.ConfigRegion(config.Region),
		newrelic.ConfigUserAgent(config.UserAgent),
		newrelic.ConfigServiceName(clientServiceName),
	}

	// Create the client directly using the variable function to allow for testing
	nrClient, err := NewrelicNewFunc(cfgOpts...)
	// print nrClient to debug
	if nrClient != nil {
		log.DefaultLogger.Debug("NewRelicClient: New Relic client initialized successfully", "client", nrClient)
	}
	if err != nil {
		return nil, &NewRelicClientError{Msg: "failed to initialize New Relic client", Err: err}
	}

	return nrClient, nil
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

	// Create service name with UID if provided
	clientServiceName := serviceName
	if config.DatasourceUID != "" {
		clientServiceName = fmt.Sprintf("%s-%s", serviceName, config.DatasourceUID)
		log.DefaultLogger.Debug("GetClient: Using UID-specific service name", "serviceName", clientServiceName, "uid", config.DatasourceUID)
	} else {
		log.DefaultLogger.Debug("GetClient: Using default service name", "serviceName", clientServiceName)
	}

	opts := []newrelic.ConfigOption{
		newrelic.ConfigPersonalAPIKey(config.APIKey),
		newrelic.ConfigRegion(config.Region),
		newrelic.ConfigUserAgent(config.UserAgent),
		newrelic.ConfigServiceName(clientServiceName),
	}

	client, err := factory.CreateClient(opts...)
	if err != nil {
		return nil, &NewRelicClientError{Msg: "failed to initialize New Relic client", Err: err}
	}
	return client, nil
}
