// Package plugin implements the New Relic Grafana datasource plugin.
// It provides functionality to query New Relic data and integrate it with Grafana.
package plugin

import (
	"context"
	"fmt"

	"newrelic-grafana-plugin/pkg/client"
	"newrelic-grafana-plugin/pkg/handler"
	"newrelic-grafana-plugin/pkg/models"
	"newrelic-grafana-plugin/pkg/validator"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

// Datasource implements the New Relic Grafana datasource plugin.
// It handles data queries, health checks, and resource management.
type Datasource struct{}

// NewDatasource creates a new instance of the New Relic datasource.
// It is called by the Grafana plugin SDK when a new datasource instance is needed.
//
// Parameters:
//   - ctx: The context for the operation
//   - settings: The datasource instance settings from Grafana
//
// Returns:
//   - instancemgmt.Instance: The new datasource instance
//   - error: Any error that occurred during creation
func NewDatasource(_ context.Context, _ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &Datasource{}, nil
}

// Dispose cleans up resources when a datasource instance is no longer needed.
// It is called by the Grafana plugin SDK when a datasource instance is being disposed.
func (d *Datasource) Dispose() {
	log.DefaultLogger.Debug("New Relic Datasource instance disposed")
}

// QueryData handles incoming data queries from Grafana.
// It processes multiple queries in parallel and returns the results.
//
// Parameters:
//   - ctx: The context for the operation
//   - req: The query data request containing multiple queries
//
// Returns:
//   - *backend.QueryDataResponse: The response containing results for all queries
//   - error: Any error that occurred during query processing
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// create response struct
	response := backend.NewQueryDataResponse()

	config, err := models.LoadPluginSettings(*req.PluginContext.DataSourceInstanceSettings)
	if err != nil {
		log.DefaultLogger.Error("Failed to load plugin settings", "error", err)
		return nil, fmt.Errorf("failed to load plugin settings: %w", err)
	}

	if err := validator.ValidatePluginSettings(config); err != nil {
		log.DefaultLogger.Error("Invalid plugin configuration", "error", err)
		return nil, fmt.Errorf("invalid plugin configuration: %w", err)
	}

	nrClient, err := client.GetClient(config.Secrets.ApiKey, &client.DefaultNewRelicClientFactory{})
	if err != nil {
		log.DefaultLogger.Error("Failed to create New Relic client", "error", err)
		return nil, fmt.Errorf("failed to create New Relic client: %w", err)
	}

	for _, q := range req.Queries {
		res := handler.HandleQuery(ctx, nrClient, config, q)
		response.Responses[q.RefID] = *res
	}

	return response, nil
}

// CheckHealth performs a health check of the datasource.
// It validates the configuration and tests the connection to New Relic.
//
// Parameters:
//   - ctx: The context for the operation
//   - req: The health check request
//
// Returns:
//   - *backend.CheckHealthResult: The result of the health check
//   - error: Any error that occurred during the health check
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// Step 1: Load plugin settings from Grafana's request
	config, err := models.LoadPluginSettings(*req.PluginContext.DataSourceInstanceSettings)
	if err != nil {
		log.DefaultLogger.Error("Failed to load plugin settings for health check", "error", err)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf(" %s", err.Error()),
		}, nil
	}

	// Step 2: Attempt to create a New Relic client using the API key from settings
	// This will catch cases where the API key is missing or invalid format.
	nrClient, err := client.GetClient(config.Secrets.ApiKey, &client.DefaultNewRelicClientFactory{})
	if err != nil {
		log.DefaultLogger.Error("Failed to create New Relic client during health check", "error", err)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("API key invalid or client failed to initialize: %s", err.Error()),
		}, nil
	}

	// Step 3: Delegate the comprehensive health check (including API call)
	// to the validator package, passing the client and context.
	healthResult, checkErr := validator.CheckHealth(ctx, config, nrClient)
	if checkErr != nil {
		// This `checkErr` should ideally be nil if validator.CheckHealth only returns `*backend.CheckHealthResult`
		// and not a Go error, but kept for robustness.
		log.DefaultLogger.Error("Unexpected error during health check validator call", "error", checkErr)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Unexpected error during health check: %s", checkErr.Error()),
		}, nil
	}
	log.DefaultLogger.Debug("Health check completed", "status", healthResult.Status.String(), "message", healthResult.Message)
	return healthResult, nil
}
