// Package plugin implements the New Relic Grafana datasource plugin.
// It provides functionality to query New Relic data and integrate it with Grafana.
package plugin

import (
	"context"
	"fmt"

	"newrelic-grafana-plugin/pkg/client"
	"newrelic-grafana-plugin/pkg/handler"
	"newrelic-grafana-plugin/pkg/models"
	"newrelic-grafana-plugin/pkg/nrdbiface"
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
func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
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
	logger := log.DefaultLogger.FromContext(ctx)
	response := backend.NewQueryDataResponse()

	config, err := models.LoadPluginSettings(*req.PluginContext.DataSourceInstanceSettings)
	if err != nil {
		logger.Error("Failed to load plugin settings", "error", err, "datasourceID", req.PluginContext.DataSourceInstanceSettings.ID)
		return nil, fmt.Errorf("failed to load plugin settings: %w", err)
	}

	if err := validator.ValidatePluginSettings(config); err != nil {
		logger.Error("Invalid plugin configuration", "error", err, "datasourceID", req.PluginContext.DataSourceInstanceSettings.ID)
		return nil, fmt.Errorf("invalid plugin configuration: %w", err)
	}

	nrClient, err := client.GetClient(config.Secrets.ApiKey, &client.DefaultNewRelicClientFactory{})
	if err != nil {
		logger.Error("Failed to create New Relic client", "error", err, "datasourceID", req.PluginContext.DataSourceInstanceSettings.ID)
		return nil, fmt.Errorf("failed to create New Relic client: %w", err)
	}

	// Create the executor wrapper for the real client
	executor := &nrdbiface.RealNRDBExecutor{NRDB: nrClient.Nrdb}

	// Process queries concurrently using a worker pool
	queryResults := make(chan struct {
		refID string
		res   backend.DataResponse
	}, len(req.Queries))

	for _, q := range req.Queries {
		go func(query backend.DataQuery) {
			res := handler.HandleQuery(ctx, executor, config, query)
			queryResults <- struct {
				refID string
				res   backend.DataResponse
			}{query.RefID, *res}
		}(q)
	}

	// Collect results
	for i := 0; i < len(req.Queries); i++ {
		result := <-queryResults
		response.Responses[result.refID] = result.res
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
	logger := log.DefaultLogger.FromContext(ctx)

	config, err := models.LoadPluginSettings(*req.PluginContext.DataSourceInstanceSettings)
	if err != nil {
		logger.Error("Failed to load plugin settings during health check", "error", err, "datasourceID", req.PluginContext.DataSourceInstanceSettings.ID)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Failed to load plugin configuration",
		}, nil
	}

	if err := validator.ValidatePluginSettings(config); err != nil {
		logger.Error("Invalid plugin configuration during health check", "error", err, "datasourceID", req.PluginContext.DataSourceInstanceSettings.ID)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Invalid plugin configuration: " + err.Error(),
		}, nil
	}

	nrClient, err := client.GetClient(config.Secrets.ApiKey, &client.DefaultNewRelicClientFactory{})
	if err != nil {
		logger.Error("Failed to create New Relic client during health check", "error", err, "datasourceID", req.PluginContext.DataSourceInstanceSettings.ID)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Failed to initialize New Relic client: " + err.Error(),
		}, nil
	}

	// Create the executor wrapper for the real client
	executor := &nrdbiface.RealNRDBExecutor{NRDB: nrClient.Nrdb}

	return validator.CheckHealth(ctx, config, executor)
}
