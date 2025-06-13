package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/dataformatter"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/models"
)

// NRQLExecutionError represents an error during NRQL query execution.
type NRQLExecutionError struct {
	Query string
	Msg   string
	Err   error // Wrapped error
}

func (e *NRQLExecutionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("NRQL query execution error for '%s': %s: %v", e.Query, e.Msg, e.Err)
	}
	return fmt.Sprintf("NRQL query execution error for '%s': %s", e.Query, e.Msg)
}

func (e *NRQLExecutionError) Unwrap() error {
	return e.Err
}

// ExecuteNRQLQuery takes a New Relic client, account ID, and NRQL query string,
// executes the query, and returns the results.
func ExecuteNRQLQuery(ctx context.Context, client *newrelic.NewRelic, accountID int, nrqlQueryText string) (*nrdb.NRDBResultContainer, error) {
	if client == nil {
		return nil, &NRQLExecutionError{Query: nrqlQueryText, Msg: "New Relic client is nil, cannot execute query"}
	}
	if nrqlQueryText == "" {
		return nil, &NRQLExecutionError{Query: nrqlQueryText, Msg: "NRQL query text cannot be empty"}
	}
	if accountID == 0 { // Assuming 0 is an invalid account ID for execution
		return nil, &NRQLExecutionError{Query: nrqlQueryText, Msg: "New Relic account ID cannot be 0"}
	}

	nrql := nrdb.NRQL(nrqlQueryText)
	results, err := client.Nrdb.QueryWithContext(ctx, accountID, nrql)
	if err != nil {
		return nil, &NRQLExecutionError{Query: nrqlQueryText, Msg: "error from New Relic API", Err: err}
	}
	return results, nil
}

// handleQuery processes a single Grafana data query.
func HandleQuery(ctx context.Context, nrClient *newrelic.NewRelic, config *models.PluginSettings, query backend.DataQuery) *backend.DataResponse {
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

	results, err := ExecuteNRQLQuery(ctx, nrClient, accountID, nrqlQueryText)
	if err != nil {
		resp.Error = fmt.Errorf("NRQL query execution failed: %w", err)
		log.DefaultLogger.Error("NRQL query execution failed", "refId", query.RefID, "query", nrqlQueryText, "accountID", accountID, "error", err)
		return resp
	}

	return dataformatter.FormatQueryResults(results, query)

}
