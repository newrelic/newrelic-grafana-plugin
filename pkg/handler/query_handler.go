// Package handler processes incoming query requests from Grafana and executes
// NRQL queries against the New Relic API. It handles query parsing, validation,
// execution, and response formatting with proper error handling.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"newrelic-grafana-plugin/pkg/formatter"
	"newrelic-grafana-plugin/pkg/models"
	"newrelic-grafana-plugin/pkg/nrdbiface"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
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

// shouldUseEnhancedQuery determines if the enhanced query should be used (FACET + TIMESERIES)
func shouldUseEnhancedQuery(query string) bool {
	hasFacet := strings.Contains(strings.ToUpper(query), "FACET")
	hasTimeseries := strings.Contains(strings.ToUpper(query), "TIMESERIES")
	return hasFacet && hasTimeseries
}

// ExecuteNRQLQuery takes an NRDB query executor, account ID, and NRQL query string,
// executes the query, and returns the results.
func ExecuteNRQLQuery(ctx context.Context, executor nrdbiface.NRDBQueryExecutor, accountID int, nrqlQueryText string) (interface{}, error) {
	if executor == nil {
		return nil, &NRQLExecutionError{Query: nrqlQueryText, Msg: "NRDB query executor is nil, cannot execute query"}
	}
	if nrqlQueryText == "" {
		return nil, &NRQLExecutionError{Query: nrqlQueryText, Msg: "NRQL query text cannot be empty"}
	}
	if accountID == 0 { // Assuming 0 is an invalid account ID for execution
		return nil, &NRQLExecutionError{Query: nrqlQueryText, Msg: "New Relic account ID cannot be 0"}
	}

	nrql := nrdb.NRQL(nrqlQueryText)
	if shouldUseEnhancedQuery(nrqlQueryText) {
		return executor.PerformNRQLQueryWithContext(ctx, accountID, nrql)
	}
	return executor.QueryWithContext(ctx, accountID, nrql)
}

// checkFacetAndTimeseries logs if both FACET and TIMESERIES are present in the query
func checkFacetAndTimeseries(query string) {
	hasFacet := strings.Contains(strings.ToUpper(query), "FACET")
	hasTimeseries := strings.Contains(strings.ToUpper(query), "TIMESERIES")
	if hasFacet && hasTimeseries {
		log.DefaultLogger.Info("Query contains both FACET and TIMESERIES", "query", query)
	}
}

// HandleQuery processes a single Grafana data query using our interface-based approach.
func HandleQuery(ctx context.Context, executor nrdbiface.NRDBQueryExecutor, config *models.PluginSettings, query backend.DataQuery) *backend.DataResponse {
	resp := &backend.DataResponse{}

	// Parse the query JSON
	var qm models.QueryModel
	if err := json.Unmarshal(query.JSON, &qm); err != nil {
		resp.Error = fmt.Errorf("error parsing query JSON: %w", err)
		log.DefaultLogger.Error("Error parsing query JSON", "refId", query.RefID, "error", err)
		return resp
	}

	log.DefaultLogger.Debug("Processing query", "refId", query.RefID, "queryText", qm.QueryText, "configAccountID", config.Secrets.AccountId, "queryAccountID", qm.AccountID)

	nrqlQueryText := "SELECT count(*) FROM Transaction SINCE 1 hour ago"
	if qm.QueryText != "" {
		nrqlQueryText = qm.QueryText
	}

	accountID := config.Secrets.AccountId
	if qm.AccountID > 0 {
		accountID = qm.AccountID
	}

	results, err := ExecuteNRQLQuery(ctx, executor, accountID, nrqlQueryText)
	if err != nil {
		resp.Error = fmt.Errorf("NRQL query execution failed: %w", err)
		log.DefaultLogger.Error("NRQL query execution failed", "refId", query.RefID, "query", nrqlQueryText, "accountID", accountID, "error", err)
		return resp
	}

	switch r := results.(type) {
	case *nrdb.NRDBResultContainer:
		return formatter.FormatQueryResults(r, query)
	case *nrdb.NRDBResultContainerMultiResultCustomized:
		return formatter.FormatFacetedTimeseriesResults(r, query)
	default:
		resp.Error = fmt.Errorf("unexpected result type from NRQL query execution")
		return resp
	}
}
