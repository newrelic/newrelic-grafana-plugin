package plugin

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/client"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/handler"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/health"
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

	nrClient, err := client.CreateNewRelicClient(config.Secrets.ApiKey, &client.DefaultNewRelicClientFactory{})
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
