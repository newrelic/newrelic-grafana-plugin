package plugin

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/newrelic/newrelic-client-go/v2/newrelic"

	// Import your specific packages.
	// Ensure that functions within these packages that you intend to mock
	// are declared as variables (e.g., `var MyFunc = func(...) (...)`) in their source files.
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/client"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/handler"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/health"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/models"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/validator"
)

// MockNewRelicClientFactory is used to mock the client.CreateNewRelicClient's internal factory.
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

// --- Basic Datasource Tests ---

// TestNewDatasource ensures that a new Datasource instance can be created.
func TestNewDatasource(t *testing.T) {
	ds, err := NewDatasource(context.Background(), backend.DataSourceInstanceSettings{})
	if err != nil {
		t.Fatalf("NewDatasource failed unexpectedly: %v", err)
	}
	if ds == nil {
		t.Fatal("NewDatasource returned a nil Datasource instance")
	}
}

// TestDatasource_Dispose ensures the Dispose method runs without errors or panics.
func TestDatasource_Dispose(t *testing.T) {
	ds := &Datasource{}
	// Calling Dispose should ideally do cleanup and not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Dispose caused a panic: %v", r)
		}
	}()
	ds.Dispose()
}

// --- CheckHealth Route Tests ---

// TestDatasource_CheckHealth_Success verifies a successful health check.
func TestDatasource_CheckHealth_Success(t *testing.T) {
	ds := &Datasource{}
	ctx := context.Background()
	req := &backend.CheckHealthRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{},
		},
	}

	// 1. Mock the 'health.ExecuteHealthCheck' function to simulate a successful outcome.
	// Store the original function to restore it later, ensuring tests are isolated.
	originalExecuteHealthCheck := health.ExecuteHealthCheck
	health.ExecuteHealthCheck = func(ctx context.Context, dsSettings backend.DataSourceInstanceSettings) (*backend.CheckHealthResult, error) {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusOk,
			Message: "Plugin is healthy!",
		}, nil
	}
	defer func() { health.ExecuteHealthCheck = originalExecuteHealthCheck }() // Restore the original function after test

	// Call the CheckHealth method of the Datasource.
	res, err := ds.CheckHealth(ctx, req)

	// Assertions for a successful health check.
	if err != nil {
		t.Fatalf("CheckHealth failed unexpectedly: %v", err)
	}
	if res == nil {
		t.Fatal("CheckHealth returned nil result")
	}
	if res.Status != backend.HealthStatusOk {
		t.Errorf("Expected HealthStatusOk, got %v", res.Status)
	}
	if res.Message != "Plugin is healthy!" {
		t.Errorf("Expected message 'Plugin is healthy!', got '%s'", res.Message)
	}
}

// TestDatasource_CheckHealth_Failure verifies a failing health check due to an internal error.
func TestDatasource_CheckHealth_Failure(t *testing.T) {
	ds := &Datasource{}
	ctx := context.Background()
	req := &backend.CheckHealthRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{},
		},
	}

	// 1. Mock 'health.ExecuteHealthCheck' to simulate an internal error.
	expectedErr := errors.New("simulated internal health check failure")
	originalExecuteHealthCheck := health.ExecuteHealthCheck
	health.ExecuteHealthCheck = func(ctx context.Context, dsSettings backend.DataSourceInstanceSettings) (*backend.CheckHealthResult, error) {
		return nil, expectedErr // Return a Go error to simulate an internal failure
	}
	defer func() { health.ExecuteHealthCheck = originalExecuteHealthCheck }()

	// Call the CheckHealth method.
	res, err := ds.CheckHealth(ctx, req)

	// Assertions for a failing health check.
	if err != nil {
		// Note: The Datasource.CheckHealth handler itself converts internal Go errors
		// into a backend.CheckHealthResult with StatusError. So, err from ds.CheckHealth
		// should be nil, and the error status should be in `res`.
		t.Fatalf("CheckHealth returned an unexpected Go error: %v", err)
	}
	if res == nil {
		t.Fatal("CheckHealth returned nil result")
	}
	if res.Status != backend.HealthStatusError {
		t.Errorf("Expected HealthStatusError, got %v", res.Status)
	}
	expectedMsg := fmt.Sprintf("Health check encountered an internal error: %s", expectedErr.Error())
	if res.Message != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, res.Message)
	}
}

// --- QueryData Route Tests ---

// TestDatasource_QueryData_Success verifies a successful data query.
func TestDatasource_QueryData_Success(t *testing.T) {
	ds := &Datasource{}
	ctx := context.Background()
	req := &backend.QueryDataRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{},
		},
		Queries: []backend.DataQuery{
			{RefID: "queryA", QueryType: "nrql", JSON: []byte(`{"nrql":"SELECT count(*) FROM Transaction"}`)},
		},
	}

	// 1. Mock 'models.LoadPluginSettings'
	originalLoadPluginSettings := models.LoadPluginSettings
	models.LoadPluginSettings = func(settings backend.DataSourceInstanceSettings) (*models.PluginSettings, error) {
		return &models.PluginSettings{
			Secrets: &models.SecretPluginSettings{
				ApiKey:    "test-api-key",
				AccountId: 12345, // Add a valid account ID for testing
			},
		}, nil
	}
	defer func() { models.LoadPluginSettings = originalLoadPluginSettings }()

	// 2. Mock 'validator.ValidatePluginSettings'
	originalValidatePluginSettings := validator.ValidatePluginSettings
	validator.ValidatePluginSettings = func(config *models.PluginSettings) error {
		return nil // Simulate successful validation
	}
	defer func() { validator.ValidatePluginSettings = originalValidatePluginSettings }()

	// 3. Mock 'client.CreateNewRelicClient'
	mockNRClient := &newrelic.NewRelic{} // A dummy New Relic client
	originalCreateNewRelicClient := client.CreateNewRelicClient
	client.CreateNewRelicClient = func(apiKey string, factory client.NewRelicClientFactory) (*newrelic.NewRelic, error) {
		return mockNRClient, nil // Simulate successful client creation
	}
	defer func() { client.CreateNewRelicClient = originalCreateNewRelicClient }()

	// 4. Mock 'handler.HandleQuery'
	expectedDataResponse := &backend.DataResponse{
		Frames: data.Frames{
			data.NewFrame("mock_frame",
				data.NewField("Time", nil, []int64{1000, 2000}),
				data.NewField("Value", nil, []float64{10.5, 20.3}),
			),
		},
	}
	originalHandleQuery := handler.HandleQuery
	handler.HandleQuery = func(ctx context.Context, nrClient *newrelic.NewRelic, config *models.PluginSettings, query backend.DataQuery) *backend.DataResponse {
		return expectedDataResponse // Return our predefined mock response
	}
	defer func() { handler.HandleQuery = originalHandleQuery }()

	// Call the QueryData method.
	res, err := ds.QueryData(ctx, req)

	// Assertions for a successful query.
	if err != nil {
		t.Fatalf("QueryData failed unexpectedly: %v", err)
	}
	if res == nil {
		t.Fatal("QueryData returned nil response")
	}
	if len(res.Responses) != 1 {
		t.Errorf("Expected 1 response, got %d", len(res.Responses))
	}
	if res.Responses["queryA"].Frames[0].Name != "mock_frame" {
		t.Errorf("Expected frame name 'mock_frame', got '%s'", res.Responses["queryA"].Frames[0].Name)
	}
}

// TestDatasource_QueryData_Failure_LoadSettings tests a failing data query due to settings loading failure.
func TestDatasource_QueryData_Failure_LoadSettings(t *testing.T) {
	ds := &Datasource{}
	ctx := context.Background()
	req := &backend.QueryDataRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{},
		},
		Queries: []backend.DataQuery{
			{RefID: "queryA"}, // A query must be present for the loop to run
		},
	}

	// 1. Mock 'models.LoadPluginSettings' to simulate a failure.
	expectedLoadErr := errors.New("simulated failed to load plugin settings")
	originalLoadPluginSettings := models.LoadPluginSettings
	models.LoadPluginSettings = func(settings backend.DataSourceInstanceSettings) (*models.PluginSettings, error) {
		return nil, expectedLoadErr // Return an error
	}
	defer func() { models.LoadPluginSettings = originalLoadPluginSettings }()

	// Call the QueryData method.
	_, err := ds.QueryData(ctx, req)

	// Assertions for a failing query.
	if err == nil {
		t.Fatal("QueryData did not return an error when settings loading failed")
	}
	// Check if the returned error wraps our expected error.
	if !errors.Is(err, expectedLoadErr) {
		t.Errorf("Expected error to wrap '%v', but got '%v'", expectedLoadErr, err)
	}
	expectedErrMsg := fmt.Sprintf("failed to load plugin settings: %s", expectedLoadErr.Error())
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', but got '%s'", expectedErrMsg, err.Error())
	}
}
