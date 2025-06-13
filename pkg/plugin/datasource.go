package plugin

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/client"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/handler"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/models"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/validator"
)

var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct{}

// NewDatasource creates a new datasource instance.
func NewDatasource(_ context.Context, _ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &Datasource{}, nil
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
	log.DefaultLogger.Debug("New Relic Datasource instance disposed.")
}

// QueryData handles incoming data queries from Grafana.
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

// CheckHealth handles health checks sent from Grafana to the plugin.
// This is used for the "Test" button on the datasource configuration page.
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
