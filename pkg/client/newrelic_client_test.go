package client

import (
	"errors"
	"testing"

	"github.com/newrelic/newrelic-client-go/v2/newrelic"
)

// MockNewRelicClientFactory implements NewRelicClientFactory for testing.
type MockNewRelicClientFactory struct {
	Client *newrelic.NewRelic
	Err    error
}

// CreateClient simulates the newrelic.New function for testing.
func (m *MockNewRelicClientFactory) CreateClient(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Client, nil
}

func TestGetClient_Success(t *testing.T) {
	apiKey := "valid_api_key"
	mockClient := &newrelic.NewRelic{} // A dummy client instance

	// Create a mock factory that returns the dummy client
	mockFactory := &MockNewRelicClientFactory{
		Client: mockClient,
		Err:    nil,
	}

	client, err := GetClient(apiKey, mockFactory)
	if err != nil {
		t.Fatalf("GetClient failed with error: %v", err)
	}
	if client == nil {
		t.Fatal("GetClient returned nil client")
	}
	if client != mockClient {
		t.Errorf("GetClient returned wrong client instance")
	}
}

func TestGetClient_EmptyAPIKey(t *testing.T) {
	apiKey := ""                                // Empty API Key
	mockFactory := &MockNewRelicClientFactory{} // Factory won't even be called if API key is empty

	_, err := GetClient(apiKey, mockFactory)
	if err == nil {
		t.Fatal("GetClient did not return error for empty API key")
	}

	nrErr, ok := err.(*NewRelicClientError)
	if !ok {
		t.Fatalf("Expected NewRelicClientError, got %T", err)
	}
	if nrErr.Msg != "New Relic API key cannot be empty" {
		t.Errorf("Expected error message 'New Relic API key cannot be empty', got '%s'", nrErr.Msg)
	}
	if nrErr.Unwrap() != nil {
		t.Error("Expected no wrapped error, but got one")
	}
}

func TestGetClient_NewRelicClientInitFailure(t *testing.T) {
	apiKey := "valid_api_key"
	expectedErr := errors.New("simulated New Relic init error")

	// Create a mock factory that simulates init failure
	mockFactory := &MockNewRelicClientFactory{
		Client: nil,
		Err:    expectedErr,
	}

	_, err := GetClient(apiKey, mockFactory)
	if err == nil {
		t.Fatal("GetClient did not return error when New Relic client init failed")
	}

	nrErr, ok := err.(*NewRelicClientError)
	if !ok {
		t.Fatalf("Expected NewRelicClientError, got %T", err)
	}
	if nrErr.Msg != "failed to initialize New Relic client" {
		t.Errorf("Expected error message 'failed to initialize New Relic client', got '%s'", nrErr.Msg)
	}
	if !errors.Is(nrErr, expectedErr) { // Use errors.Is to check for wrapped error
		t.Errorf("Expected wrapped error %v, got %v", expectedErr, nrErr.Unwrap())
	}
}
