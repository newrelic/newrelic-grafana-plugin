// Package health provides comprehensive health check functionality for the New Relic Grafana plugin.
// It validates plugin settings, initializes clients, and performs connectivity tests
// to ensure the datasource is properly configured and operational.
package health

import (
	"context"
	"fmt"

	"newrelic-grafana-plugin/pkg/client"
	"newrelic-grafana-plugin/pkg/models"
	"newrelic-grafana-plugin/pkg/nrdbiface"
	"newrelic-grafana-plugin/pkg/validator"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// checkHealthFunction allows for mocking the validator.CheckHealth function in tests
var checkHealthFunction = validator.CheckHealth

// ExecuteHealthCheck performs a comprehensive health check for the New Relic datasource.
// It encapsulates the full logic for validating plugin settings, initializing the
// New Relic client, and performing a test API call to New Relic.
//
// Parameters:
//
//	ctx: The context for the operation, allowing for cancellation/timeouts.
//	dsSettings: The DataSourceInstanceSettings from Grafana's health check request,
//	            containing configuration like API keys.
//
// Returns:
//
//	A *backend.CheckHealthResult indicating the status and a message for Grafana.
//	An error if an unexpected internal Go error occurs during the process.
func PerformHealthCheck1(ctx context.Context, dsSettings backend.DataSourceInstanceSettings) (*backend.CheckHealthResult, error) {
	log.DefaultLogger.Debug("health.ExecuteHealthCheck: Starting comprehensive health check logic")

	// Step 1: Load plugin settings from Grafana's request.
	// This is the first point of failure if the configuration itself is invalid.
	config, err := models.LoadPluginSettings(dsSettings)
	if err != nil {
		log.DefaultLogger.Error("health.ExecuteHealthCheck: Failed to load plugin settings", "error", err)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Failed to load datasource configuration: %s", err.Error()),
		}, nil
	}

	// Step 2: Attempt to create a New Relic client using the API key from settings.
	// This verifies that the API key is present and allows for basic client initialization.
	clientConfig := client.DefaultConfig()
	clientConfig.APIKey = config.Secrets.ApiKey
	clientConfig.DatasourceUID = dsSettings.UID // Set the datasource UID for unique service name
	log.DefaultLogger.Debug("health.ExecuteHealthCheck: Creating client with UID", "uid", dsSettings.UID)
	nrClient, err := client.NewClient(clientConfig)
	if err != nil {
		log.DefaultLogger.Error("health.ExecuteHealthCheck: Failed to create New Relic client", "error", err)
		// Return a HealthStatusError to Grafana, indicating an issue with client setup.
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("API key invalid or New Relic client failed to initialize: %s", err.Error()),
		}, nil
	}

	// Step 3: Create the executor wrapper for the real client
	executor := &nrdbiface.RealNRDBExecutor{NRDB: nrClient.Nrdb}

	// Step 4: Delegate the actual New Relic API connectivity check to the 'validator' package.
	// This is where a real API call (e.g., a simple NRQL query) is performed to confirm
	// that New Relic is reachable and the credentials are valid.
	healthResult, checkErr := checkHealthFunction(ctx, config, executor)
	if checkErr != nil {
		// This condition catches any unexpected Go errors returned by the validator.
		// Ideally, validator.CheckHealth should always return a *backend.CheckHealthResult
		// with an appropriate Status, making this 'checkErr' nil on expected outcomes.
		log.DefaultLogger.Error("health.ExecuteHealthCheck: Unexpected error from validator.CheckHealth", "error", checkErr)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Internal error during New Relic API check: %s", checkErr.Error()),
		}, nil
	}

	// Step 5: Log the final health check status and message for internal debugging.
	log.DefaultLogger.Debug("health.ExecuteHealthCheck: Health check completed", "status", healthResult.Status.String(), "message", healthResult.Message)
	// Return the result directly from the validator to Grafana.
	return healthResult, nil
}

var ExecuteHealthCheck = func(ctx context.Context, dsSettings backend.DataSourceInstanceSettings) (*backend.CheckHealthResult, error) {
	// This variable is used to allow mocking in tests.
	return PerformHealthCheck1(ctx, dsSettings)
}
