package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/newrelic/newrelic-client-go/newrelic"
	"github.com/org/newrelic-grafana-plugin/pkg/connection"
	"github.com/org/newrelic-grafana-plugin/pkg/dataformatter"
	"github.com/org/newrelic-grafana-plugin/pkg/models"
	"github.com/org/newrelic-grafana-plugin/pkg/nrql"
	"github.com/org/newrelic-grafana-plugin/pkg/util"
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

	if err := util.ValidatePluginSettings(config); err != nil {
		log.DefaultLogger.Error("Invalid plugin configuration", "error", err)
		return nil, fmt.Errorf("invalid plugin configuration: %w", err)
	}

	nrClient, err := connection.GetClient(config.Secrets.ApiKey)
	if err != nil {
		log.DefaultLogger.Error("Failed to create New Relic client", "error", err)
		return nil, fmt.Errorf("failed to create New Relic client: %w", err)
	}

	for _, q := range req.Queries {
		res := d.handleQuery(ctx, nrClient, config, q)
		response.Responses[q.RefID] = *res
	}

	return response, nil
}

// handleQuery processes a single Grafana data query.
func (d *Datasource) handleQuery(ctx context.Context, nrClient *newrelic.NewRelic, config *models.PluginSettings, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	// Parse the query JSON
	var qm models.QueryModel
	if err := json.Unmarshal(query.JSON, &qm); err != nil {
		resp.Error = fmt.Errorf("error parsing query JSON: %w", err)
		log.DefaultLogger.Error("Error parsing query JSON", "refId", query.RefID, "error", err)
		return resp
	}

	log.DefaultLogger.Debug("Processing query", "refId", query.RefID, "queryText", qm.QueryText, "configAccountID", config.Secrets.AccountId, "queryAccountID", qm.AccountID)

	nrqlQueryText := "SELECT count(*) FROM Transaction"
	if qm.QueryText != "" {
		nrqlQueryText = qm.QueryText
	}

	accountID := config.Secrets.AccountId
	if qm.AccountID > 0 {
		accountID = qm.AccountID
	}

	results, err := nrql.ExecuteNRQLQuery(ctx, nrClient, accountID, nrqlQueryText)
	if err != nil {
		resp.Error = fmt.Errorf("NRQL query execution failed: %w", err)
		log.DefaultLogger.Error("NRQL query execution failed", "refId", query.RefID, "query", nrqlQueryText, "accountID", accountID, "error", err)
		return resp
	}

	if dataformatter.IsCountQuery(results) {
		return dataformatter.FormatCountQueryResults(results)
	}
	return dataformatter.FormatRegularQueryResults(results, query)

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
			Message: fmt.Sprintf("Failed to load plugin settings for health check: %s", err.Error()),
		}, nil
	}

	// Step 2: Attempt to create a New Relic client using the API key from settings
	// This will catch cases where the API key is missing or invalid format.
	nrClient, err := connection.GetClient(config.Secrets.ApiKey)
	if err != nil {
		log.DefaultLogger.Error("Failed to create New Relic client during health check", "error", err)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("API key invalid or client failed to initialize: %s", err.Error()),
		}, nil
	}

	// Step 3: Delegate the comprehensive health check (including API call)
	// to the util package, passing the client and context.
	healthResult, checkErr := util.CheckHealth(ctx, config, nrClient)
	if checkErr != nil {
		// This `checkErr` should ideally be nil if util.CheckHealth only returns `*backend.CheckHealthResult`
		// and not a Go error, but kept for robustness.
		log.DefaultLogger.Error("Unexpected error during health check utility call", "error", checkErr)
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Unexpected error during health check: %s", checkErr.Error()),
		}, nil
	}
	log.DefaultLogger.Debug("Health check completed", "status", healthResult.Status.String(), "message", healthResult.Message)
	return healthResult, nil

}
