package handler

import (
	"context"
	"errors"
	"testing"

	"newrelic-grafana-plugin/pkg/models"
	"newrelic-grafana-plugin/pkg/nrdbiface"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	"github.com/stretchr/testify/assert"
)

// mockNRDBExecutor implements the nrdbiface.NRDBQueryExecutor interface for testing
type mockNRDBExecutor struct {
	queryErr error
	results  *nrdb.NRDBResultContainer
}

func (m *mockNRDBExecutor) QueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL) (*nrdb.NRDBResultContainer, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	if m.results == nil {
		// Return a default successful result
		return &nrdb.NRDBResultContainer{
			Results: []nrdb.NRDBResult{
				{
					"count": 42,
				},
			},
		}, nil
	}
	return m.results, nil
}

// TestNormalizeQuery tests the NormalizeQuery function with various inputs
func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple query",
			input:    "SELECT average(duration) FROM Transaction",
			expected: "SELECT average(duration) FROM Transaction",
		},
		{
			name:     "query with line breaks",
			input:    "SELECT average(duration) \nFROM Transaction",
			expected: "SELECT average(duration) FROM Transaction",
		},
		{
			name:     "query with carriage returns",
			input:    "SELECT average(duration) \r\nFROM Transaction",
			expected: "SELECT average(duration) FROM Transaction",
		},
		{
			name:     "query with multiple spaces",
			input:    "SELECT  average(duration)   FROM  Transaction",
			expected: "SELECT average(duration) FROM Transaction",
		},
		{
			name:     "complex query with line breaks and spaces",
			input:    "SELECT\n  average(duration)\n  FROM\n  Transaction\n  WHERE appName = 'MyApp'\n  LIMIT 100",
			expected: "SELECT average(duration) FROM Transaction WHERE appName = 'MyApp' LIMIT 100",
		},
		{
			name:     "query with leading and trailing spaces",
			input:    "  SELECT average(duration) FROM Transaction  ",
			expected: "SELECT average(duration) FROM Transaction",
		},
		{
			name:     "query with SQL comments",
			input:    "-- This is a comment\nSELECT count(*) FROM\nTransaction\n-- Another comment\nFACET request.uri\nsince 1 week ago",
			expected: "SELECT count(*) FROM Transaction FACET request.uri since 1 week ago",
		},
		{
			name:     "query with WITH clause comment",
			input:    "--WITH aparse(rawQuery, '%SELECT *(%)%') AS nrqlFunctions\nselect * from Transaction",
			expected: "select * from Transaction",
		},
		{
			name:     "query with comments and empty lines",
			input:    "\n\n-- Comment at the top\n\nSELECT count(*)\n-- Comment in the middle\nFROM Transaction\n\n-- Comment at the end\n",
			expected: "SELECT count(*) FROM Transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeQuery(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func (m *mockNRDBExecutor) PerformNRQLQueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL) (*nrdb.NRDBResultContainerMultiResultCustomized, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	// Return a default successful faceted timeseries result
	return &nrdb.NRDBResultContainerMultiResultCustomized{
		Results: []nrdb.NRDBResult{
			{"count": 42.0},
		},
	}, nil
}

func TestExecuteNRQLQuery(t *testing.T) {
	tests := []struct {
		name       string
		executor   nrdbiface.NRDBQueryExecutor
		accountID  int
		query      string
		wantErr    bool
		errMessage string
	}{
		{
			name:      "successful query",
			executor:  &mockNRDBExecutor{results: &nrdb.NRDBResultContainer{}},
			accountID: 123456,
			query:     "SELECT count(*) FROM Transaction",
			wantErr:   false,
		},
		{
			name:       "nil executor",
			executor:   nil,
			accountID:  123456,
			query:      "SELECT count(*) FROM Transaction",
			wantErr:    true,
			errMessage: "NRDB query executor is nil, cannot execute query",
		},
		{
			name:       "empty query",
			executor:   &mockNRDBExecutor{},
			accountID:  123456,
			query:      "",
			wantErr:    true,
			errMessage: "NRQL query text cannot be empty",
		},
		{
			name:       "invalid account ID",
			executor:   &mockNRDBExecutor{},
			accountID:  0,
			query:      "SELECT count(*) FROM Transaction",
			wantErr:    true,
			errMessage: "New Relic account ID cannot be 0",
		},
		{
			name:       "query error",
			executor:   &mockNRDBExecutor{queryErr: errors.New("API error")},
			accountID:  123456,
			query:      "SELECT count(*) FROM Transaction",
			wantErr:    true,
			errMessage: "API error",
		},
		{
			name:      "query with line breaks",
			executor:  &mockNRDBExecutor{results: &nrdb.NRDBResultContainer{}},
			accountID: 123456,
			query:     "SELECT\ncount(*)\nFROM\nTransaction",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ExecuteNRQLQuery(context.Background(), tt.executor, tt.accountID, NormalizeQuery(tt.query))
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Nil(t, results)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, results)
			}
		})
	}
}

func TestHandleQuery_QueryHandler(t *testing.T) {
	tests := []struct {
		name       string
		queryJSON  string
		config     *models.PluginSettings
		executor   *mockNRDBExecutor
		wantErr    bool
		errMessage string
	}{
		{
			name: "successful query with default account ID",
			queryJSON: `{
				"queryText": "SELECT count(*) FROM Transaction"
			}`,
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					AccountId: 123456,
				},
			},
			executor: &mockNRDBExecutor{
				results: &nrdb.NRDBResultContainer{},
			},
			wantErr: false,
		},
		{
			name: "query with custom account ID",
			queryJSON: `{
				"queryText": "SELECT count(*) FROM Transaction",
				"accountID": 789012
			}`,
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					AccountId: 123456,
				},
			},
			executor: &mockNRDBExecutor{
				results: &nrdb.NRDBResultContainer{},
			},
			wantErr: false,
		},
		{
			name: "invalid query JSON",
			queryJSON: `{
				"queryText": "SELECT count(*) FROM Transaction",
				"accountID": "invalid"
			}`,
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					AccountId: 123456,
				},
			},
			executor: &mockNRDBExecutor{},
			wantErr:  true,
		},
		{
			name: "query execution error",
			queryJSON: `{
				"queryText": "SELECT count(*) FROM Transaction"
			}`,
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					AccountId: 123456,
				},
			},
			executor: &mockNRDBExecutor{
				queryErr: errors.New("API error"),
			},
			wantErr:    true,
			errMessage: "NRQL query execution failed",
		},
		{
			name: "empty query",
			queryJSON: `{
				"queryText": ""
			}`,
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					AccountId: 123456,
				},
			},
			executor:   &mockNRDBExecutor{},
			wantErr:    true,
			errMessage: "query text cannot be empty",
		},
		{
			name: "query with line breaks",
			queryJSON: `{
				"queryText": "SELECT\ncount(*)\nFROM\nTransaction"
			}`,
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					AccountId: 123456,
				},
			},
			executor: &mockNRDBExecutor{
				results: &nrdb.NRDBResultContainer{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := backend.DataQuery{
				RefID: "A",
				JSON:  []byte(tt.queryJSON),
			}

			resp := HandleQuery(context.Background(), tt.executor, tt.config, query)
			if tt.wantErr {
				assert.Error(t, resp.Error)
				if tt.errMessage != "" {
					assert.Contains(t, resp.Error.Error(), tt.errMessage)
				}
			} else {
				assert.NoError(t, resp.Error)
			}
		})
	}
}
