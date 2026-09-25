// Package nrdbiface provides interfaces for New Relic Database (NRDB) query execution.
// This package enables dependency injection and testing by abstracting the concrete
// New Relic client implementation behind interfaces.
package nrdbiface

import (
	"context"
	"time"

	"github.com/newrelic/newrelic-client-go/v2/pkg/nerdgraph"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
)

// NRDBQueryExecutor defines the interface for executing NRQL queries against New Relic.
// This abstraction allows for easier testing and dependency injection.
//
// timeoutSeconds is an optional per-query NRQL timeout override (5-120s); 0 means
// unset, in which case New Relic's own default timeout applies, unchanged.
type NRDBQueryExecutor interface {
	QueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL, timeoutSeconds int) (*nrdb.NRDBResultContainer, error)
	PerformNRQLQueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL, timeoutSeconds int) (*nrdb.NRDBResultContainerMultiResultCustomized, error)
}

// RealNRDBExecutor is a wrapper around the real New Relic client that implements
// NRDBQueryExecutor. This allows us to use dependency injection in production code.
type RealNRDBExecutor struct {
	NRDB      nrdb.Nrdb
	NerdGraph nerdgraph.NerdGraph
	// Optional client for timeout-override queries; falls back to NerdGraph when nil.
	OverrideNerdGraph *nerdgraph.NerdGraph
}

func (r *RealNRDBExecutor) overrideNerdGraph() *nerdgraph.NerdGraph {
	if r.OverrideNerdGraph != nil {
		return r.OverrideNerdGraph
	}
	return &r.NerdGraph
}

// contextWithTimeoutBuffer sets a generous deadline alongside the GraphQL timeout
// argument, which is what actually lets NerdGraph run the query longer — this ctx
// deadline is a secondary, best-effort safeguard, not a guaranteed circuit breaker:
// the vendored newrelic-client-go HTTP layer has not been observed to abort an
// in-flight request on context deadline/cancellation. When timeoutSeconds is 0, ctx
// is returned unchanged.
func contextWithTimeoutBuffer(ctx context.Context, timeoutSeconds int) (context.Context, context.CancelFunc) {
	if timeoutSeconds == 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, time.Duration(timeoutSeconds+10)*time.Second)
}

// QueryWithContext executes an NRQL query using the real New Relic client. When
// timeoutSeconds is set, it sends gqlNrqlQueryWithTimeout via NerdGraph rather than
// QueryWithAdditionalOptions, whose document also requests suggestedFacets: that field
// has its own short server-side timeout, and its TIMEOUT error makes the client retry
// the whole (slow) query.
func (r *RealNRDBExecutor) QueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL, timeoutSeconds int) (*nrdb.NRDBResultContainer, error) {
	ctx, cancel := contextWithTimeoutBuffer(ctx, timeoutSeconds)
	defer cancel()

	if timeoutSeconds == 0 {
		return r.NRDB.QueryWithContext(ctx, accountID, query)
	}

	respBody := gqlNRQLStandardQueryWithTimeoutResponse{}
	if err := r.overrideNerdGraph().QueryWithResponseAndContext(ctx, gqlNrqlQueryWithTimeout, timeoutQueryVars(accountID, query, timeoutSeconds), &respBody); err != nil {
		return nil, err
	}
	return &respBody.Actor.Account.NRQL, nil
}

func timeoutQueryVars(accountID int, query nrdb.NRQL, timeoutSeconds int) map[string]interface{} {
	return map[string]interface{}{
		"accountId": accountID,
		"query":     query,
		"timeout":   nrdb.Seconds(timeoutSeconds),
	}
}

// gqlNrqlQueryWithTimeout mirrors gqlNrqlQuery (the client library's document behind
// both QueryWithContext and PerformNRQLQueryWithContext) field-for-field, adding only
// the $timeout variable, so override queries request exactly what unset queries do.
// Deliberately does NOT request rawResponse: query_handler.go branches on RawResponse
// being non-nil for FACET+TIMESERIES results, and requesting it would silently switch
// their rendering from FormatFacetedTimeseriesResults to FormatUniversal.
const gqlNrqlQueryWithTimeout = `query (
	$query: Nrql!,
	$accountId: Int!,
	$timeout: Seconds
)
{
  actor {
    account(id: $accountId) {
      nrql(query: $query, timeout: $timeout) {
        currentResults
        otherResult
        previousResults
        results
        totalResult
        metadata {
          eventTypes
          facets
          messages
          timeWindow {
            begin
            compareWith
            end
            since
            until
          }
        }
      }
    }
  }
}
`

// gqlNRQLQueryWithTimeoutResponse mirrors the client library's unexported
// gqlNRQLQueryResponseCustomized so PerformNRQLQueryWithContext's hand-built request
// unmarshals into the same, exported NRDBResultContainerMultiResultCustomized type.
type gqlNRQLQueryWithTimeoutResponse struct {
	Actor struct {
		Account struct {
			NRQL nrdb.NRDBResultContainerMultiResultCustomized
		}
	}
}

// gqlNRQLStandardQueryWithTimeoutResponse is the same shape for QueryWithContext.
type gqlNRQLStandardQueryWithTimeoutResponse struct {
	Actor struct {
		Account struct {
			NRQL nrdb.NRDBResultContainer
		}
	}
}

// PerformNRQLQueryWithContext executes an NRQL query using the enhanced New Relic
// client (used for FACET+TIMESERIES queries). When timeoutSeconds is set, no client
// method supports a timeout for this response shape, so it sends a hand-built GraphQL
// document (gqlNrqlQueryWithTimeout) via NerdGraph directly.
func (r *RealNRDBExecutor) PerformNRQLQueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL, timeoutSeconds int) (*nrdb.NRDBResultContainerMultiResultCustomized, error) {
	ctx, cancel := contextWithTimeoutBuffer(ctx, timeoutSeconds)
	defer cancel()

	if timeoutSeconds == 0 {
		return r.NRDB.PerformNRQLQueryWithContext(ctx, accountID, query)
	}

	respBody := gqlNRQLQueryWithTimeoutResponse{}
	if err := r.overrideNerdGraph().QueryWithResponseAndContext(ctx, gqlNrqlQueryWithTimeout, timeoutQueryVars(accountID, query, timeoutSeconds), &respBody); err != nil {
		return nil, err
	}
	return &respBody.Actor.Account.NRQL, nil
}
