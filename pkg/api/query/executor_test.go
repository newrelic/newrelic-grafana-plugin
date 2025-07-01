package query

import (
	"context"
	"errors"
	"testing"

	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	"github.com/stretchr/testify/assert"
)

// MockNRDBExecutor implements the nrdbiface.NRDBQueryExecutor interface for testing
type MockNRDBExecutor struct {
	shouldFail bool
	results    *nrdb.NRDBResultContainer
	err        error
}

func (m *MockNRDBExecutor) QueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL) (*nrdb.NRDBResultContainer, error) {
	if m.shouldFail {
		return nil, m.err
	}
	return m.results, nil
}

func (m *MockNRDBExecutor) PerformNRQLQueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL) (*nrdb.NRDBResultContainerMultiResultCustomized, error) {
	if m.shouldFail {
		return nil, m.err
	}

	// Convert standard result to multi-result
	multiResult := &nrdb.NRDBResultContainerMultiResultCustomized{}
	if m.results != nil {
		multiResult.Metadata = m.results.Metadata
		multiResult.Results = m.results.Results
	}

	return multiResult, nil
}

func TestExecutor_Execute(t *testing.T) {
	mockResults := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{"count": 42},
		},
	}

	tests := []struct {
		name       string
		executor   *Executor
		accountID  int
		query      string
		shouldFail bool
		errMsg     string
	}{
		{
			name:      "success",
			executor:  NewExecutor(&MockNRDBExecutor{results: mockResults}),
			accountID: 12345,
			query:     "SELECT count(*) FROM Transaction",
		},
		{
			name:       "nil executor",
			executor:   &Executor{executor: nil},
			accountID:  12345,
			query:      "SELECT count(*) FROM Transaction",
			shouldFail: true,
			errMsg:     "NRDB query executor is nil",
		},
		{
			name:       "empty query",
			executor:   NewExecutor(&MockNRDBExecutor{}),
			accountID:  12345,
			query:      "",
			shouldFail: true,
			errMsg:     "NRQL query text cannot be empty",
		},
		{
			name:       "query execution error",
			executor:   NewExecutor(&MockNRDBExecutor{shouldFail: true, err: errors.New("execution error")}),
			accountID:  12345,
			query:      "SELECT count(*) FROM Transaction",
			shouldFail: true,
			errMsg:     "execution error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := tt.executor.Execute(context.Background(), tt.accountID, tt.query)

			if tt.shouldFail {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, results)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, mockResults, results)
			}
		})
	}
}

func TestExecutionError(t *testing.T) {
	tests := []struct {
		name     string
		err      *ExecutionError
		expected string
	}{
		{
			name: "with wrapped error",
			err: &ExecutionError{
				Query: "SELECT count(*) FROM Transaction",
				Msg:   "test message",
				Err:   errors.New("wrapped error"),
			},
			expected: "NRQL query execution error for 'SELECT count(*) FROM Transaction': test message: wrapped error",
		},
		{
			name: "without wrapped error",
			err: &ExecutionError{
				Query: "SELECT count(*) FROM Transaction",
				Msg:   "test message",
			},
			expected: "NRQL query execution error for 'SELECT count(*) FROM Transaction': test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
			assert.Equal(t, tt.err.Err, tt.err.Unwrap())
		})
	}
}

// No multi-result method in the executor
