// Package nrdbiface provides interfaces for New Relic Database (NRDB) query execution.
// This package enables dependency injection and testing by abstracting the concrete
// New Relic client implementation behind interfaces.
package nrdbiface

import (
	"context"

	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
)

// NRDBQueryExecutor defines the interface for executing NRQL queries against New Relic.
// This abstraction allows for easier testing and dependency injection.
type NRDBQueryExecutor interface {
	QueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL) (*nrdb.NRDBResultContainer, error)
}

// RealNRDBExecutor is a wrapper around the real nrdb.Nrdb that implements NRDBQueryExecutor.
// This allows us to use dependency injection in production code.
type RealNRDBExecutor struct {
	NRDB nrdb.Nrdb
}

// QueryWithContext executes an NRQL query using the real New Relic client.
func (r *RealNRDBExecutor) QueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL) (*nrdb.NRDBResultContainer, error) {
	return r.NRDB.QueryWithContext(ctx, accountID, query)
}
