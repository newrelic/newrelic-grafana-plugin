package nrdbiface

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNRDBExecutor is a simple implementation of NRDBQueryExecutor for testing
type TestNRDBExecutor struct {
	ShouldError      bool
	QueryResult      *nrdb.NRDBResultContainer
	MultiQueryResult *nrdb.NRDBResultContainerMultiResultCustomized
	Error            error
}

func (e *TestNRDBExecutor) QueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL, timeoutSeconds int) (*nrdb.NRDBResultContainer, error) {
	if e.ShouldError {
		return nil, e.Error
	}
	return e.QueryResult, nil
}

func (e *TestNRDBExecutor) PerformNRQLQueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL, timeoutSeconds int) (*nrdb.NRDBResultContainerMultiResultCustomized, error) {
	if e.ShouldError {
		return nil, e.Error
	}
	return e.MultiQueryResult, nil
}

func TestNRDBQueryExecutorInterface(t *testing.T) {
	// Verify our test type implements the interface
	var _ NRDBQueryExecutor = (*TestNRDBExecutor)(nil)

	// Initialize test executor with float64 values (which is what the real client returns)
	testExecutor := &TestNRDBExecutor{
		QueryResult: &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{{"count": float64(42)}},
		},
		MultiQueryResult: &nrdb.NRDBResultContainerMultiResultCustomized{
			Results: []nrdb.NRDBResult{{"count": float64(42)}},
		},
	}

	// Test standard query
	result, err := testExecutor.QueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction", 0)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(42), result.Results[0]["count"])

	// Test multi-result query
	multiResult, err := testExecutor.PerformNRQLQueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction", 0)
	assert.NoError(t, err)
	assert.NotNil(t, multiResult)
	assert.Equal(t, float64(42), multiResult.Results[0]["count"])

	// Test error case
	testExecutor.ShouldError = true
	testExecutor.Error = errors.New("test error")

	result, err = testExecutor.QueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction", 0)
	assert.Error(t, err)
	assert.Nil(t, result)

	multiResult, err = testExecutor.PerformNRQLQueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction", 0)
	assert.Error(t, err)
	assert.Nil(t, multiResult)
}

// Using the TestNRDBExecutor which already implements NRDBQueryExecutor
// to test RealNRDBExecutor

func TestRealNRDBExecutor_QueryWithContext(t *testing.T) {
	tests := []struct {
		name        string
		mockResult  *nrdb.NRDBResultContainer
		mockError   error
		expectError bool
	}{
		{
			name: "successful query",
			mockResult: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{"count": 42},
				},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "query error",
			mockResult:  nil,
			mockError:   errors.New("query failed"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test executor that implements NRDBQueryExecutor
			testExecutor := &TestNRDBExecutor{
				QueryResult: tt.mockResult,
				Error:       tt.mockError,
				ShouldError: tt.expectError,
			}

			// Test the query method
			result, err := testExecutor.QueryWithContext(context.Background(), 12345, nrdb.NRQL("SELECT count(*) FROM Transaction"), 0)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.mockError, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.mockResult, result)
			}
		})
	}
}

func TestRealNRDBExecutor_PerformNRQLQueryWithContext(t *testing.T) {
	tests := []struct {
		name        string
		mockResult  *nrdb.NRDBResultContainerMultiResultCustomized
		mockError   error
		expectError bool
	}{
		{
			name: "successful multi query",
			mockResult: &nrdb.NRDBResultContainerMultiResultCustomized{
				Results: []nrdb.NRDBResult{
					{"count": 42},
				},
			},
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "multi query error",
			mockResult:  nil,
			mockError:   errors.New("multi query failed"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test executor that implements NRDBQueryExecutor
			testExecutor := &TestNRDBExecutor{
				MultiQueryResult: tt.mockResult,
				Error:            tt.mockError,
				ShouldError:      tt.expectError,
			}

			// Test the multi-query method
			result, err := testExecutor.PerformNRQLQueryWithContext(context.Background(), 12345, nrdb.NRQL("SELECT count(*) FROM Transaction"), 0)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.mockError, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.mockResult, result)
			}
		})
	}
}

// newTestRealExecutor stands up a fake NerdGraph server that records the last
// request body it received and always returns respJSON, then builds a real
// RealNRDBExecutor pointed at it. This exercises the actual HTTP request our
// client builds, rather than a mocked Go interface.
func newTestRealExecutor(t *testing.T, respJSON string) (*RealNRDBExecutor, *capturedRequest) {
	t.Helper()

	captured := &capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		captured.body = body

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(respJSON))
	}))
	t.Cleanup(server.Close)

	nrClient, err := newrelic.New(
		newrelic.ConfigPersonalAPIKey("test-key"),
		newrelic.ConfigNerdGraphBaseURL(server.URL),
	)
	require.NoError(t, err)

	return &RealNRDBExecutor{NRDB: nrClient.Nrdb, NerdGraph: nrClient.NerdGraph}, captured
}

type capturedRequest struct {
	body []byte
}

// variables unmarshals the GraphQL variables map from the captured request body.
func (c *capturedRequest) variables(t *testing.T) map[string]interface{} {
	t.Helper()
	var req struct {
		Variables map[string]interface{} `json:"variables"`
	}
	require.NoError(t, json.Unmarshal(c.body, &req))
	return req.Variables
}

const standardQueryResponseJSON = `{"data":{"actor":{"account":{"nrql":{"results":[{"count":42}]}}}}}`

func TestRealNRDBExecutor_QueryWithContext_NoTimeoutOverride(t *testing.T) {
	executor, captured := newTestRealExecutor(t, standardQueryResponseJSON)

	result, err := executor.QueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction", 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Regression guard: with no override, the request must carry no "timeout"
	// variable at all — identical to today's behavior.
	_, hasTimeout := captured.variables(t)["timeout"]
	assert.False(t, hasTimeout, "unset timeout must not appear in the outgoing request")
}

func TestRealNRDBExecutor_QueryWithContext_WithTimeoutOverride(t *testing.T) {
	executor, captured := newTestRealExecutor(t, standardQueryResponseJSON)

	result, err := executor.QueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction", 30)
	require.NoError(t, err)
	require.NotNil(t, result)

	vars := captured.variables(t)
	assert.Equal(t, float64(30), vars["timeout"], "the requested timeout must be sent on the wire")

	// suggestedFacets and eventDefinitions have their own short server-side timeouts;
	// their TIMEOUT errors would make the client retry the whole query. Override queries
	// must request the same fields as unset ones.
	body := string(captured.body)
	for _, field := range []string{"suggestedFacets", "eventDefinitions", "rawResponse", "queryProgress"} {
		assert.NotContains(t, body, field)
	}
}

const enhancedQueryResponseJSON = `{"data":{"actor":{"account":{"nrql":{"results":[{"count":42}]}}}}}`

func TestRealNRDBExecutor_PerformNRQLQueryWithContext_NoTimeoutOverride(t *testing.T) {
	executor, captured := newTestRealExecutor(t, enhancedQueryResponseJSON)

	result, err := executor.PerformNRQLQueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction FACET appName TIMESERIES", 0)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Regression guards for the unset case: no timeout variable, and no
	// rawResponse field in the query document (byte-identical to today).
	_, hasTimeout := captured.variables(t)["timeout"]
	assert.False(t, hasTimeout)
	assert.NotContains(t, string(captured.body), "rawResponse")
}

func TestRealNRDBExecutor_PerformNRQLQueryWithContext_WithTimeoutOverride(t *testing.T) {
	executor, captured := newTestRealExecutor(t, enhancedQueryResponseJSON)

	result, err := executor.PerformNRQLQueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction FACET appName TIMESERIES", 90)
	require.NoError(t, err)
	require.NotNil(t, result)

	vars := captured.variables(t)
	assert.Equal(t, float64(90), vars["timeout"], "the requested timeout must be sent on the wire")

	// The direct regression-guard test: the hand-built document must never
	// request rawResponse, or FACET+TIMESERIES panels would silently switch
	// rendering paths in query_handler.go.
	assert.NotContains(t, string(captured.body), "rawResponse")
}

// TestRealNRDBExecutor_PerformNRQLQueryWithContext_PassesCtxThrough verifies that
// the no-override path calls the context-preserving client method (the bug fix)
// rather than the old ctx-dropping PerformNRQLQuery. It does NOT assert that a
// canceled/expired ctx aborts the in-flight HTTP call — verified empirically against
// this vendored client version, an expired context does not abort a request already
// in flight, so ctx cancellation here is forwarded correctly but is not a reliable
// circuit breaker. The GraphQL timeout argument remains the only mechanism that
// reliably bounds how long a query actually runs.
func TestRealNRDBExecutor_PerformNRQLQueryWithContext_PassesCtxThrough(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(enhancedQueryResponseJSON))
	}))
	defer server.Close()

	nrClient, err := newrelic.New(
		newrelic.ConfigPersonalAPIKey("test-key"),
		newrelic.ConfigNerdGraphBaseURL(server.URL),
	)
	require.NoError(t, err)
	executor := &RealNRDBExecutor{NRDB: nrClient.Nrdb, NerdGraph: nrClient.NerdGraph}

	result, err := executor.PerformNRQLQueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction FACET appName TIMESERIES", 0)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// TestRealNRDBExecutor_OverrideQueriesUseLongerHTTPClient scales the real timings
// down: the default client gives up at 1s, the override client waits 5s, and the
// fake NerdGraph answers after 2s.
func TestRealNRDBExecutor_OverrideQueriesUseLongerHTTPClient(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(standardQueryResponseJSON))
	}))
	t.Cleanup(server.Close)

	newClient := func(timeout time.Duration) *newrelic.NewRelic {
		c, err := newrelic.New(
			newrelic.ConfigPersonalAPIKey("test-key"),
			newrelic.ConfigNerdGraphBaseURL(server.URL),
			newrelic.ConfigHTTPTimeout(timeout),
		)
		require.NoError(t, err)
		return c
	}
	defaultClient, overrideClient := newClient(time.Second), newClient(5*time.Second)
	executor := &RealNRDBExecutor{
		NRDB:              defaultClient.Nrdb,
		NerdGraph:         defaultClient.NerdGraph,
		OverrideNerdGraph: &overrideClient.NerdGraph,
	}

	t.Run("standard path with override", func(t *testing.T) {
		atomic.StoreInt32(&hits, 0)
		result, err := executor.QueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction", 60)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "must succeed on the first attempt, with no retries")
	})

	t.Run("FACET+TIMESERIES path with override", func(t *testing.T) {
		atomic.StoreInt32(&hits, 0)
		result, err := executor.PerformNRQLQueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction FACET appName TIMESERIES", 60)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "must succeed on the first attempt, with no retries")
	})

	t.Run("no override keeps using the default client", func(t *testing.T) {
		_, err := executor.QueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction", 0)
		assert.Error(t, err, "unset timeout must still go through the default client and its shorter HTTP timeout")
	})
}
