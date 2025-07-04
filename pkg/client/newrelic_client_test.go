package client

import (
	"errors"
	"testing"
	"time"

	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	"github.com/stretchr/testify/assert"
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

// TestNewClient tests the direct client creation functionality
func TestNewClient(t *testing.T) {
	// Save the original newrelic.New function to restore after tests
	originalNewFunc := NewrelicNewFunc
	defer func() { NewrelicNewFunc = originalNewFunc }()

	tests := []struct {
		name    string
		config  ClientConfig
		mockFn  func(...newrelic.ConfigOption) (*newrelic.NewRelic, error)
		wantErr bool
	}{
		{
			name: "valid config",
			config: ClientConfig{
				APIKey:    "valid-api-key",
				Region:    "US",
				Timeout:   30 * time.Second,
				UserAgent: "test-user-agent",
			},
			mockFn: func(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error) {
				return &newrelic.NewRelic{}, nil
			},
			wantErr: false,
		},
		{
			name: "empty api key",
			config: ClientConfig{
				APIKey:    "",
				Region:    "US",
				Timeout:   30 * time.Second,
				UserAgent: "test-user-agent",
			},
			mockFn:  nil, // won't be called with empty API key
			wantErr: true,
		},
		{
			name: "newrelic.New error",
			config: ClientConfig{
				APIKey:    "valid-api-key",
				Region:    "US",
				Timeout:   30 * time.Second,
				UserAgent: "test-user-agent",
			},
			mockFn: func(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error) {
				return nil, errors.New("failed to initialize client")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override newrelic.New for this test case if mockFn is provided
			if tt.mockFn != nil {
				NewrelicNewFunc = tt.mockFn
			}

			client, err := NewClient(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
				// For empty API key, verify specific error message
				if tt.config.APIKey == "" {
					assert.Contains(t, err.Error(), "API key cannot be empty")
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}

	// Test cases for UID-specific service names
	t.Run("with datasource UID", func(t *testing.T) {
		// Mock the newrelic.New function to capture configuration options
		var capturedOpts []newrelic.ConfigOption
		NewrelicNewFunc = func(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error) {
			capturedOpts = opts
			return &newrelic.NewRelic{}, nil
		}

		config := ClientConfig{
			APIKey:        "valid-api-key",
			Region:        "US",
			Timeout:       30 * time.Second,
			UserAgent:     "test-user-agent",
			DatasourceUID: "test-uid-123",
		}

		client, err := NewClient(config)
		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Verify that configuration options were captured
		assert.NotEmpty(t, capturedOpts)

		// Note: We can't easily test the exact service name without more complex mocking
		// but we can verify that the function completes successfully with a UID
	})

	t.Run("without datasource UID", func(t *testing.T) {
		// Mock the newrelic.New function
		NewrelicNewFunc = func(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error) {
			return &newrelic.NewRelic{}, nil
		}

		config := ClientConfig{
			APIKey:        "valid-api-key",
			Region:        "US",
			Timeout:       30 * time.Second,
			UserAgent:     "test-user-agent",
			DatasourceUID: "", // Empty UID
		}

		client, err := NewClient(config)
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})
}

// TestCreateNewRelicClient_Success tests the successful creation of a New Relic client.
func TestCreateNewRelicClient_Success(t *testing.T) {
	// Define a valid API key for the test.
	apiKey := "valid_api_key"
	// Create a dummy New Relic client instance.
	mockClient := &newrelic.NewRelic{}

	// Create a mock factory configured to return the dummy client and no error.
	mockFactory := &MockNewRelicClientFactory{
		Client: mockClient,
		Err:    nil,
	}

	// Call the function under test.
	client, err := CreateNewRelicClient(apiKey, mockFactory)

	// Assertions:
	if err != nil {
		t.Fatalf("CreateNewRelicClient failed with unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("CreateNewRelicClient returned nil client")
	}
	if client != mockClient {
		t.Errorf("CreateNewRelicClient returned wrong client instance; expected %p, got %p", mockClient, client)
	}
}

// TestCreateNewRelicClient_EmptyAPIKey tests the scenario where an empty API key is provided.
func TestCreateNewRelicClient_EmptyAPIKey(t *testing.T) {
	// Define an empty API key for the test.
	apiKey := ""
	mockFactory := &MockNewRelicClientFactory{}

	// Call the function under test.
	_, err := CreateNewRelicClient(apiKey, mockFactory)

	// Assertions:
	if err == nil {
		t.Fatal("CreateNewRelicClient did not return error for empty API key")
	}

	nrErr, ok := err.(*NewRelicClientError)
	if !ok {
		t.Fatalf("Expected error of type *NewRelicClientError, got %T", err)
	}
	expectedErrMsg := "New Relic API key cannot be empty"
	if nrErr.Msg != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, nrErr.Msg)
	}
	if nrErr.Unwrap() != nil {
		t.Errorf("Expected no wrapped error for empty API key, but got: %v", nrErr.Unwrap())
	}
}

// TestCreateNewRelicClient_NewRelicClientInitFailure tests when the underlying New Relic client
// initialization fails.
func TestCreateNewRelicClient_NewRelicClientInitFailure(t *testing.T) {
	apiKey := "valid_api_key"
	expectedErr := errors.New("simulated New Relic client initialization error from SDK")

	mockFactory := &MockNewRelicClientFactory{
		Client: nil,
		Err:    expectedErr,
	}

	_, err := CreateNewRelicClient(apiKey, mockFactory)

	if err == nil {
		t.Fatal("CreateNewRelicClient did not return error when New Relic client init failed")
	}

	nrErr, ok := err.(*NewRelicClientError)
	if !ok {
		t.Fatalf("Expected error of type *NewRelicClientError, got %T", err)
	}
	expectedErrMsg := "failed to initialize New Relic client"
	if nrErr.Msg != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, nrErr.Msg)
	}
	if !errors.Is(nrErr, expectedErr) {
		t.Errorf("Expected wrapped error to be '%v', but got '%v'", expectedErr, nrErr.Unwrap())
	}
}

func TestGetClient(t *testing.T) {
	tests := []struct {
		name    string
		config  ClientConfig
		factory NewRelicClientFactory
		wantErr bool
	}{
		{
			name: "valid config",
			config: ClientConfig{
				APIKey:    "valid-api-key",
				Region:    "US",
				Timeout:   30 * time.Second,
				UserAgent: "test-user-agent",
			},
			factory: &MockNewRelicClientFactory{Client: &newrelic.NewRelic{}},
			wantErr: false,
		},
		{
			name: "empty api key",
			config: ClientConfig{
				APIKey:    "",
				Region:    "US",
				Timeout:   30 * time.Second,
				UserAgent: "test-user-agent",
			},
			factory: &DefaultNewRelicClientFactory{},
			wantErr: true,
		},
		{
			name: "factory error",
			config: ClientConfig{
				APIKey:    "valid-api-key",
				Region:    "US",
				Timeout:   30 * time.Second,
				UserAgent: "test-user-agent",
			},
			factory: &MockNewRelicClientFactory{
				Err: errors.New("factory error"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := GetClient(tt.config, tt.factory)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestNewRelicClientError(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		err     error
		wantMsg string
	}{
		{
			name:    "with wrapped error",
			msg:     "test error",
			err:     errors.New("wrapped error"),
			wantMsg: "new relic client error: test error: wrapped error",
		},
		{
			name:    "without wrapped error",
			msg:     "test error",
			err:     nil,
			wantMsg: "new relic client error: test error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &NewRelicClientError{
				Msg: tt.msg,
				Err: tt.err,
			}
			assert.Equal(t, tt.wantMsg, err.Error())
			assert.Equal(t, tt.err, err.Unwrap())
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, "US", config.Region)
	assert.Equal(t, "newrelic-grafana-plugin", config.UserAgent)
	assert.IsType(t, time.Duration(0), config.Timeout)
	assert.Greater(t, config.Timeout, time.Duration(0))
}
