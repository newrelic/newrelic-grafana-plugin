package nrql

import (
	"context"
	"fmt"

	"github.com/newrelic/newrelic-client-go/v2/newrelic"
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
