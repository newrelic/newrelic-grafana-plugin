package client

import (
	"errors"
	"testing"

	"github.com/newrelic/newrelic-client-go/v2/newrelic"
)

// MockNewRelicClientFactory implements NewRelicClientFactory for testing.
// It allows simulating successful client creation or an error during creation.
type MockNewRelicClientFactory struct {
	Client *newrelic.NewRelic // The dummy client to return on success
	Err    error              // The error to return on failure
}

// CreateClient simulates the newrelic.New function for testing purposes.
// It returns the pre-configured Client or Err from the MockNewRelicClientFactory.
func (m *MockNewRelicClientFactory) CreateClient(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error) {
	// If an error is configured in the mock, return it immediately.
	if m.Err != nil {
		return nil, m.Err
	}
	// Otherwise, return the mock client.
	return m.Client, nil
}

// TestCreateNewRelicClient_Success tests the successful creation of a New Relic client.
func TestCreateNewRelicClient_Success(t *testing.T) {
	// Define a valid API key for the test.
	apiKey := "valid_api_key"
	// Create a dummy New Relic client instance. In a real scenario, this would be a mock object
	// or a bare struct matching the interface of newrelic.NewRelic methods you might use.
	mockClient := &newrelic.NewRelic{}

	// Create a mock factory configured to return the dummy client and no error.
	mockFactory := &MockNewRelicClientFactory{
		Client: mockClient,
		Err:    nil,
	}

	// Call the function under test.
	client, err := CreateNewRelicClient(apiKey, mockFactory)

	// Assertions:
	// 1. Check that no error was returned.
	if err != nil {
		t.Fatalf("CreateNewRelicClient failed with unexpected error: %v", err)
	}
	// 2. Check that a client instance was returned.
	if client == nil {
		t.Fatal("CreateNewRelicClient returned nil client")
	}
	// 3. Check that the returned client is the exact mock client we provided.
	if client != mockClient {
		t.Errorf("CreateNewRelicClient returned wrong client instance; expected %p, got %p", mockClient, client)
	}
}

// TestCreateNewRelicClient_EmptyAPIKey tests the scenario where an empty API key is provided.
func TestCreateNewRelicClient_EmptyAPIKey(t *testing.T) {
	// Define an empty API key for the test.
	apiKey := ""
	// Create a mock factory. Note that its CreateClient method won't even be called
	// because the API key validation happens before factory usage.
	mockFactory := &MockNewRelicClientFactory{}

	// Call the function under test.
	_, err := CreateNewRelicClient(apiKey, mockFactory)

	// Assertions:
	// 1. Check that an error was returned.
	if err == nil {
		t.Fatal("CreateNewRelicClient did not return error for empty API key")
	}

	// 2. Type assertion to ensure the error is of type *NewRelicClientError.
	nrErr, ok := err.(*NewRelicClientError)
	if !ok {
		t.Fatalf("Expected error of type *NewRelicClientError, got %T", err)
	}
	// 3. Check the specific error message.
	expectedErrMsg := "New Relic API key cannot be empty"
	if nrErr.Msg != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, nrErr.Msg)
	}
	// 4. Check that no underlying error is wrapped.
	if nrErr.Unwrap() != nil {
		t.Errorf("Expected no wrapped error for empty API key, but got: %v", nrErr.Unwrap())
	}
}

// TestCreateNewRelicClient_NewRelicClientInitFailure tests when the underlying New Relic client
// initialization (simulated by the factory) fails.
func TestCreateNewRelicClient_NewRelicClientInitFailure(t *testing.T) {
	// Define a valid API key (it won't be the cause of failure here).
	apiKey := "valid_api_key"
	// Define an expected error that the mock factory will return.
	expectedErr := errors.New("simulated New Relic client initialization error from SDK")

	// Create a mock factory configured to return an error when CreateClient is called.
	mockFactory := &MockNewRelicClientFactory{
		Client: nil, // No client is returned on error
		Err:    expectedErr,
	}

	// Call the function under test.
	_, err := CreateNewRelicClient(apiKey, mockFactory)

	// Assertions:
	// 1. Check that an error was returned.
	if err == nil {
		t.Fatal("CreateNewRelicClient did not return error when New Relic client init failed")
	}

	// 2. Type assertion to ensure the error is of type *NewRelicClientError.
	nrErr, ok := err.(*NewRelicClientError)
	if !ok {
		t.Fatalf("Expected error of type *NewRelicClientError, got %T", err)
	}
	// 3. Check the specific error message.
	expectedErrMsg := "failed to initialize New Relic client"
	if nrErr.Msg != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, nrErr.Msg)
	}
	// 4. Use errors.Is to check if the expected underlying error is wrapped.
	if !errors.Is(nrErr, expectedErr) {
		t.Errorf("Expected wrapped error to be '%v', but got '%v'", expectedErr, nrErr.Unwrap())
	}
}
