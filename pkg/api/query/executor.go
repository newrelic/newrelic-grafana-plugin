// Package query provides functionality for executing New Relic NRQL queries.
// It handles query validation, execution, and error handling.
package query

import (
	"context"
	"fmt"

	"newrelic-grafana-plugin/pkg/nrdbiface"

	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
)

// ExecutionError represents an error during NRQL query execution.
type ExecutionError struct {
	Query string
	Msg   string
	Err   error // Wrapped error
}

func (e *ExecutionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("NRQL query execution error for '%s': %s: %v", e.Query, e.Msg, e.Err)
	}
	return fmt.Sprintf("NRQL query execution error for '%s': %s", e.Query, e.Msg)
}

func (e *ExecutionError) Unwrap() error {
	return e.Err
}

// Executor handles the execution of NRQL queries against New Relic.
type Executor struct {
	executor nrdbiface.NRDBQueryExecutor
}

// NewExecutor creates a new query executor with the given NRDB query executor.
func NewExecutor(executor nrdbiface.NRDBQueryExecutor) *Executor {
	return &Executor{executor: executor}
}

// Execute runs an NRQL query against New Relic and returns the results.
// It validates the input parameters and handles any errors that occur during execution.
func (e *Executor) Execute(ctx context.Context, accountID int, query string) (*nrdb.NRDBResultContainer, error) {
	if e.executor == nil {
		return nil, &ExecutionError{Query: query, Msg: "NRDB query executor is nil, cannot execute query"}
	}
	if query == "" {
		return nil, &ExecutionError{Query: query, Msg: "NRQL query text cannot be empty"}
	}
	if accountID == 0 {
		return nil, &ExecutionError{Query: query, Msg: "New Relic account ID cannot be 0"}
	}

	nrql := nrdb.NRQL(query)
	results, err := e.executor.QueryWithContext(ctx, accountID, nrql)
	if err != nil {
		return nil, &ExecutionError{Query: query, Msg: "error from New Relic API", Err: err}
	}
	return results, nil
}
