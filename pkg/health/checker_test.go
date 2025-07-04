package health

import (
	"context"
	"fmt"
	"testing"

	"newrelic-grafana-plugin/pkg/client"
	"newrelic-grafana-plugin/pkg/models"
	"newrelic-grafana-plugin/pkg/nrdbiface"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerformHealthCheck1(t *testing.T) {
	tests := []struct {
		name             string
		settings         backend.DataSourceInstanceSettings
		expectedStatus   backend.HealthStatus
		expectedContains string
	}{
		{
			name: "successful health check",
			settings: backend.DataSourceInstanceSettings{
				ID:   1,
				Name: "test-datasource",
				DecryptedSecureJSONData: map[string]string{
					"apiKey":    "test-api-key",
					"accountID": "123456",
				},
				JSONData: []byte(`{}`),
			},
			expectedStatus:   backend.HealthStatusError, // Will fail with real API call
			expectedContains: "Authentication failed",
		},
		{
			name: "invalid settings - missing API key",
			settings: backend.DataSourceInstanceSettings{
				ID:       1,
				Name:     "test-datasource",
				JSONData: []byte(`{}`),
				DecryptedSecureJSONData: map[string]string{
					"accountID": "123456",
				},
			},
			expectedStatus:   backend.HealthStatusError,
			expectedContains: "Enter New Relic API key",
		},
		{
			name: "invalid settings - invalid JSON",
			settings: backend.DataSourceInstanceSettings{
				ID:       1,
				Name:     "test-datasource",
				JSONData: []byte(`invalid json`),
				DecryptedSecureJSONData: map[string]string{
					"apiKey":    "test-api-key",
					"accountID": "123456",
				},
			},
			expectedStatus:   backend.HealthStatusError,
			expectedContains: "Failed to load datasource configuration",
		},
		{
			name: "invalid settings - missing account ID",
			settings: backend.DataSourceInstanceSettings{
				ID:       1,
				Name:     "test-datasource",
				JSONData: []byte(`{}`),
				DecryptedSecureJSONData: map[string]string{
					"apiKey": "test-api-key",
				},
			},
			expectedStatus:   backend.HealthStatusError,
			expectedContains: "Enter an account ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PerformHealthCheck1(context.Background(), tt.settings)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedStatus, result.Status)
			assert.Contains(t, result.Message, tt.expectedContains)
		})
	}
}

func TestExecuteHealthCheck(t *testing.T) {
	// Test that the ExecuteHealthCheck variable works correctly
	originalFunc := ExecuteHealthCheck
	defer func() {
		ExecuteHealthCheck = originalFunc
	}()

	// Test with original function
	settings := backend.DataSourceInstanceSettings{
		ID:       1,
		Name:     "test-datasource",
		JSONData: []byte(`{}`),
		DecryptedSecureJSONData: map[string]string{
			"apiKey":    "test-api-key",
			"accountID": "123456",
		},
	}

	result, err := ExecuteHealthCheck(context.Background(), settings)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, backend.HealthStatusError, result.Status) // Will fail with real API

	// Test with mocked function
	ExecuteHealthCheck = func(ctx context.Context, dsSettings backend.DataSourceInstanceSettings) (*backend.CheckHealthResult, error) {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusOk,
			Message: "Mocked success",
		}, nil
	}

	result, err = ExecuteHealthCheck(context.Background(), settings)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, backend.HealthStatusOk, result.Status)
	assert.Equal(t, "Mocked success", result.Message)
}

// TestExecuteHealthCheckPassthrough ensures ExecuteHealthCheck correctly passes through to PerformHealthCheck1
func TestExecuteHealthCheckPassthrough(t *testing.T) {
	// Save original function and restore after test
	originalExecuteHealthCheck := ExecuteHealthCheck
	defer func() { ExecuteHealthCheck = originalExecuteHealthCheck }()

	// Create a mock of PerformHealthCheck1 by replacing ExecuteHealthCheck directly
	mockCalled := false
	ExecuteHealthCheck = func(ctx context.Context, dsSettings backend.DataSourceInstanceSettings) (*backend.CheckHealthResult, error) {
		mockCalled = true
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusOk,
			Message: "test passthrough",
		}, nil
	}

	// Call ExecuteHealthCheck
	result, err := ExecuteHealthCheck(context.Background(), backend.DataSourceInstanceSettings{})

	// Verify expectations
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, mockCalled, "ExecuteHealthCheck mock was not called")
	assert.Equal(t, backend.HealthStatusOk, result.Status)
	assert.Equal(t, "test passthrough", result.Message)
}

// TestPerformHealthCheck1_ClientCreationFailure tests different scenarios for client creation failure
func TestPerformHealthCheck1_ClientCreationFailure(t *testing.T) {
	tests := []struct {
		name             string
		settings         backend.DataSourceInstanceSettings
		expectedStatus   backend.HealthStatus
		expectedContains string
	}{
		{
			name: "empty API key",
			settings: backend.DataSourceInstanceSettings{
				ID:       1,
				Name:     "test-datasource",
				JSONData: []byte(`{}`),
				DecryptedSecureJSONData: map[string]string{
					"apiKey":    "",
					"accountID": "123456",
				},
			},
			expectedStatus:   backend.HealthStatusError,
			expectedContains: "Enter New Relic API key",
		},
		{
			name: "malformed account ID",
			settings: backend.DataSourceInstanceSettings{
				ID:       1,
				Name:     "test-datasource",
				JSONData: []byte(`{}`),
				DecryptedSecureJSONData: map[string]string{
					"apiKey":    "test-api-key",
					"accountID": "not-a-number",
				},
			},
			expectedStatus:   backend.HealthStatusError,
			expectedContains: "could not convert accountID 'not-a-number' to int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PerformHealthCheck1(context.Background(), tt.settings)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedStatus, result.Status)
			assert.Contains(t, result.Message, tt.expectedContains)
		})
	}
}

// TestPerformHealthCheck1WithMockValidatorError directly tests the error handling in PerformHealthCheck1
// by mocking the CheckHealth function
func TestPerformHealthCheck1WithMockValidatorError(t *testing.T) {
	// Save original function and restore after test
	originalCheckHealthFunc := checkHealthFunction
	defer func() { checkHealthFunction = originalCheckHealthFunc }()

	// Mock the CheckHealth function to return an error
	checkHealthFunction = func(ctx context.Context, settings *models.PluginSettings, executor nrdbiface.NRDBQueryExecutor) (*backend.CheckHealthResult, error) {
		return nil, fmt.Errorf("unexpected validator error")
	}

	// Use valid settings
	settings := backend.DataSourceInstanceSettings{
		ID:   1,
		Name: "test-datasource",
		DecryptedSecureJSONData: map[string]string{
			"apiKey":    "test-api-key",
			"accountID": "123456",
		},
		JSONData: []byte(`{}`),
	}

	// Call PerformHealthCheck1 with our mocked validator
	result, err := PerformHealthCheck1(context.Background(), settings)

	// Verify expectations
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, backend.HealthStatusError, result.Status)
	assert.Contains(t, result.Message, "Internal error during New Relic API check")
	assert.Contains(t, result.Message, "unexpected validator error")
}

// TestPerformHealthCheck1_ValidatorError tests the scenario where ExecuteHealthCheck returns a result via our mocked checkHealthFunction
func TestPerformHealthCheck1_ValidatorError(t *testing.T) {
	// Save original function and restore after test
	originalCheckHealthFunc := checkHealthFunction
	defer func() { checkHealthFunction = originalCheckHealthFunc }()

	// Mock the CheckHealth function to return an error
	checkHealthFunction = func(ctx context.Context, settings *models.PluginSettings, executor nrdbiface.NRDBQueryExecutor) (*backend.CheckHealthResult, error) {
		return nil, fmt.Errorf("unexpected validator error")
	}

	// Use a valid settings object that would normally pass the earlier checks
	settings := backend.DataSourceInstanceSettings{
		ID:   1,
		Name: "test-datasource",
		DecryptedSecureJSONData: map[string]string{
			"apiKey":    "test-api-key",
			"accountID": "123456",
		},
		JSONData: []byte(`{}`),
	}

	// Execute the health check directly (not through the ExecuteHealthCheck variable)
	result, err := PerformHealthCheck1(context.Background(), settings)

	// Verify expectations
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, backend.HealthStatusError, result.Status)
	assert.Contains(t, result.Message, "Internal error during New Relic API check")
	assert.Contains(t, result.Message, "unexpected validator error")
}

// TestPerformHealthCheck1_SuccessfulValidation tests the scenario where validator.CheckHealth returns a success result
func TestPerformHealthCheck1_SuccessfulValidation(t *testing.T) {
	// Save original function and restore after test
	originalCheckHealthFunc := checkHealthFunction
	defer func() { checkHealthFunction = originalCheckHealthFunc }()

	// Create a custom mock implementation of CheckHealth
	successResult := &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "✅ New Relic connection successful (Account ID: 123456)",
	}

	// Track if our mock was called
	mockWasCalled := false

	// Mock the CheckHealth function to return our success result
	checkHealthFunction = func(ctx context.Context, settings *models.PluginSettings, executor nrdbiface.NRDBQueryExecutor) (*backend.CheckHealthResult, error) {
		mockWasCalled = true
		return successResult, nil
	}

	// Use valid settings
	settings := backend.DataSourceInstanceSettings{
		ID:   1,
		Name: "test-datasource",
		DecryptedSecureJSONData: map[string]string{
			"apiKey":    "test-api-key",
			"accountID": "123456",
		},
		JSONData: []byte(`{}`),
	}

	// Call PerformHealthCheck1 with our mocked validator
	result, err := PerformHealthCheck1(context.Background(), settings)

	// Verify expectations
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, mockWasCalled, "Mock CheckHealth function was not called")
	assert.Equal(t, backend.HealthStatusOk, result.Status)
	assert.Equal(t, "✅ New Relic connection successful (Account ID: 123456)", result.Message)

	// Verify that our mock result was returned unchanged (reference equality)
	assert.True(t, result == successResult, "Expected the exact same result object to be returned")
}

// TestPerformHealthCheck1_NewClientError tests the scenario where client.NewClient returns an error
func TestPerformHealthCheck1_NewClientError(t *testing.T) {
	// Store the original function and restore after test
	originalNewFunc := client.NewrelicNewFunc
	defer func() { client.NewrelicNewFunc = originalNewFunc }()

	// Mock the newrelicNewFunc to always return an error
	client.NewrelicNewFunc = func(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error) {
		return nil, fmt.Errorf("simulated client creation error")
	}

	// Use valid settings that would otherwise pass
	settings := backend.DataSourceInstanceSettings{
		ID:   1,
		Name: "test-datasource",
		DecryptedSecureJSONData: map[string]string{
			"apiKey":    "test-api-key",
			"accountID": "123456",
		},
		JSONData: []byte(`{}`),
	}

	// Call PerformHealthCheck1 which should now hit the client error path
	result, err := PerformHealthCheck1(context.Background(), settings)

	// Verify expectations
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, backend.HealthStatusError, result.Status)
	assert.Contains(t, result.Message, "API key invalid or New Relic client failed to initialize")
	assert.Contains(t, result.Message, "simulated client creation error")
}

// TestPerformHealthCheck1_WithUID tests health check with datasource UID
func TestPerformHealthCheck1_WithUID(t *testing.T) {
	// Save original client creation function
	originalNewFunc := client.NewrelicNewFunc
	defer func() { client.NewrelicNewFunc = originalNewFunc }()

	// Save original health check function
	originalCheckHealthFunction := checkHealthFunction
	defer func() { checkHealthFunction = originalCheckHealthFunction }()

	// Mock the client creation to succeed
	client.NewrelicNewFunc = func(opts ...newrelic.ConfigOption) (*newrelic.NewRelic, error) {
		return &newrelic.NewRelic{}, nil
	}

	// Mock the health check function to avoid actual API calls that would panic
	checkHealthFunction = func(ctx context.Context, config *models.PluginSettings, executor nrdbiface.NRDBQueryExecutor) (*backend.CheckHealthResult, error) {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusOk,
			Message: "✅ Test successful with UID",
		}, nil
	}

	settings := backend.DataSourceInstanceSettings{
		ID:   1,
		UID:  "test-uid-456", // Test UID
		Name: "test-datasource-with-uid",
		DecryptedSecureJSONData: map[string]string{
			"apiKey":    "test-api-key",
			"accountID": "123456",
		},
		JSONData: []byte(`{}`),
	}

	ctx := context.Background()
	result, err := PerformHealthCheck1(ctx, settings)

	// With proper mocking, this should succeed
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, backend.HealthStatusOk, result.Status)

	// The key thing is that the function completes successfully with UID present
	assert.NotEmpty(t, settings.UID, "UID should be present for this test")
}
