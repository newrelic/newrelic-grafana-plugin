// Package plugin implements the New Relic Grafana datasource plugin.
// It provides functionality to query New Relic data and integrate it with Grafana.
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"newrelic-grafana-plugin/pkg/client"
	"newrelic-grafana-plugin/pkg/handler"
	"newrelic-grafana-plugin/pkg/health"
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
	_ backend.CallResourceHandler   = (*Datasource)(nil)
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

	// Get datasource UID for service naming
	datasourceUID := req.PluginContext.DataSourceInstanceSettings.UID

	config, err := models.LoadPluginSettings(*req.PluginContext.DataSourceInstanceSettings)
	if err != nil {
		logger.Error("Failed to load plugin settings", "error", err, "datasourceID", req.PluginContext.DataSourceInstanceSettings.ID)
		return nil, fmt.Errorf("failed to load plugin settings: %w", err)
	}

	if err := validator.ValidatePluginSettings(config); err != nil {
		logger.Error("Invalid plugin configuration", "error", err, "datasourceID", req.PluginContext.DataSourceInstanceSettings.ID)
		return nil, fmt.Errorf("invalid plugin configuration: %w", err)
	}

	// Create a client config
	clientConfig := client.DefaultConfig()
	clientConfig.APIKey = config.Secrets.ApiKey
	clientConfig.DatasourceUID = datasourceUID // Set the datasource UID for unique service name

	// Create New Relic client using the new method
	nrClient, err := client.NewClient(clientConfig)
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
	log.DefaultLogger.Debug("Datasource.CheckHealth: Initiating health check routing")

	// Delegate the comprehensive health check to the 'health' package.
	// We pass the context and the raw DataSourceInstanceSettings.
	healthResult, err := health.ExecuteHealthCheck(ctx, *req.PluginContext.DataSourceInstanceSettings)
	if err != nil {
		// Log the unexpected error from the health package for debugging purposes.
		log.DefaultLogger.Error("Datasource.CheckHealth: Health check failed internally", "error", err)
		// Return a generic error to Grafana UI if the health.ExecuteHealthCheck
		// returns a Go error instead of a *backend.CheckHealthResult with an error status.
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Health check encountered an internal error: %s", err.Error()),
		}, nil
	}

	// Return the result received from the health package directly to Grafana.
	return healthResult, nil
}

// CallResource handles incoming resource requests from Grafana.
// It processes the request and returns the appropriate response.
//
// Parameters:
//   - ctx: The context for the operation
//   - req: The resource request
//   - sender: The response sender
//
// Returns:
//   - error: Any error that occurred during resource processing
func (d *Datasource) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	log.DefaultLogger.Debug("Datasource.CallResource: Handling resource request", "path", req.Path)

	switch req.Path {
	case "health":
		return d.handleHealthResource(ctx, req, sender)
	default:
		return sender.Send(&backend.CallResourceResponse{
			Status: http.StatusNotFound,
			Body:   []byte(`{"error": "Resource not found"}`),
		})
	}
}

// handleHealthResource handles the /health resource endpoint
func (d *Datasource) handleHealthResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	// Call the same health check logic used by CheckHealth
	healthResult, err := health.ExecuteHealthCheck(ctx, *req.PluginContext.DataSourceInstanceSettings)
	if err != nil {
		log.DefaultLogger.Error("Resource health check failed internally", "error", err)
		// Return 200 with error details instead of 500 to avoid browser popups
		response := map[string]interface{}{
			"status":  "ERROR",
			"message": "Internal health check error: " + err.Error(),
		}
		responseBody, _ := json.Marshal(response)
		return sender.Send(&backend.CallResourceResponse{
			Status: http.StatusOK,
			Body:   responseBody,
			Headers: map[string][]string{
				"Content-Type": {"application/json"},
			},
		})
	}

	// Convert health result to JSON response
	response := map[string]interface{}{
		"status":  healthResult.Status.String(),
		"message": healthResult.Message,
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		log.DefaultLogger.Error("Failed to marshal health response", "error", err)
		// Return 200 with error details instead of 500 to avoid browser popups
		fallbackResponse := map[string]interface{}{
			"status":  "ERROR",
			"message": "Failed to process health check response",
		}
		fallbackBody, _ := json.Marshal(fallbackResponse)
		return sender.Send(&backend.CallResourceResponse{
			Status: http.StatusOK,
			Body:   fallbackBody,
			Headers: map[string][]string{
				"Content-Type": {"application/json"},
			},
		})
	}

	return sender.Send(&backend.CallResourceResponse{
		Status: http.StatusOK,
		Body:   responseBody,
		Headers: map[string][]string{
			"Content-Type": {"application/json"},
		},
	})
}
