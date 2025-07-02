package nrdbiface

import (
	"context"
	"errors"
	"testing"

	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	"github.com/stretchr/testify/assert"
)

// TestNRDBExecutor is a simple implementation of NRDBQueryExecutor for testing
type TestNRDBExecutor struct {
	ShouldError      bool
	QueryResult      *nrdb.NRDBResultContainer
	MultiQueryResult *nrdb.NRDBResultContainerMultiResultCustomized
	Error            error
}

func (e *TestNRDBExecutor) QueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL) (*nrdb.NRDBResultContainer, error) {
	if e.ShouldError {
		return nil, e.Error
	}
	return e.QueryResult, nil
}

func (e *TestNRDBExecutor) PerformNRQLQueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL) (*nrdb.NRDBResultContainerMultiResultCustomized, error) {
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
	result, err := testExecutor.QueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(42), result.Results[0]["count"])

	// Test multi-result query
	multiResult, err := testExecutor.PerformNRQLQueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction")
	assert.NoError(t, err)
	assert.NotNil(t, multiResult)
	assert.Equal(t, float64(42), multiResult.Results[0]["count"])

	// Test error case
	testExecutor.ShouldError = true
	testExecutor.Error = errors.New("test error")

	result, err = testExecutor.QueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction")
	assert.Error(t, err)
	assert.Nil(t, result)

	multiResult, err = testExecutor.PerformNRQLQueryWithContext(context.Background(), 12345, "SELECT count(*) FROM Transaction")
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
			result, err := testExecutor.QueryWithContext(context.Background(), 12345, nrdb.NRQL("SELECT count(*) FROM Transaction"))

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
			result, err := testExecutor.PerformNRQLQueryWithContext(context.Background(), 12345, nrdb.NRQL("SELECT count(*) FROM Transaction"))

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
