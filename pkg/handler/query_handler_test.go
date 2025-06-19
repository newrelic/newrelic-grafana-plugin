package handler

import (
	"context"

	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
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
				{"count": 42.0},
			},
		}, nil
	}
	return m.results, nil
}

// func TestExecuteNRQLQuery(t *testing.T) {
// 	tests := []struct {
// 		name       string
// 		executor   nrdbiface.NRDBQueryExecutor
// 		accountID  int
// 		query      string
// 		wantErr    bool
// 		errMessage string
// 	}{
// 		{
// 			name:      "successful query",
// 			executor:  &mockNRDBExecutor{results: &nrdb.NRDBResultContainer{}},
// 			accountID: 123456,
// 			query:     "SELECT count(*) FROM Transaction",
// 			wantErr:   false,
// 		},
// 		{
// 			name:       "nil executor",
// 			executor:   nil,
// 			accountID:  123456,
// 			query:      "SELECT count(*) FROM Transaction",
// 			wantErr:    true,
// 			errMessage: "NRDB query executor is nil, cannot execute query",
// 		},
// 		{
// 			name:       "empty query",
// 			executor:   &mockNRDBExecutor{},
// 			accountID:  123456,
// 			query:      "",
// 			wantErr:    true,
// 			errMessage: "NRQL query text cannot be empty",
// 		},
// 		{
// 			name:       "invalid account ID",
// 			executor:   &mockNRDBExecutor{},
// 			accountID:  0,
// 			query:      "SELECT count(*) FROM Transaction",
// 			wantErr:    true,
// 			errMessage: "New Relic account ID cannot be 0",
// 		},
// 		{
// 			name:       "query error",
// 			executor:   &mockNRDBExecutor{queryErr: errors.New("API error")},
// 			accountID:  123456,
// 			query:      "SELECT count(*) FROM Transaction",
// 			wantErr:    true,
// 			errMessage: "error from New Relic API",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			results, err := ExecuteNRQLQuery(context.Background(), tt.executor, tt.accountID, tt.query)
// 			if tt.wantErr {
// 				assert.Error(t, err)
// 				if tt.errMessage != "" {
// 					assert.Contains(t, err.Error(), tt.errMessage)
// 				}
// 				assert.Nil(t, results)
// 			} else {
// 				assert.NoError(t, err)
// 				assert.NotNil(t, results)
// 			}
// 		})
// 	}
// }

// func TestHandleQuery_QueryHandler(t *testing.T) {
// 	tests := []struct {
// 		name       string
// 		queryJSON  string
// 		config     *models.PluginSettings
// 		executor   *mockNRDBExecutor
// 		wantErr    bool
// 		errMessage string
// 	}{
// 		{
// 			name: "successful query with default account ID",
// 			queryJSON: `{
// 				"queryText": "SELECT count(*) FROM Transaction"
// 			}`,
// 			config: &models.PluginSettings{
// 				Secrets: &models.SecretPluginSettings{
// 					AccountId: 123456,
// 				},
// 			},
// 			executor: &mockNRDBExecutor{
// 				results: &nrdb.NRDBResultContainer{},
// 			},
// 			wantErr: false,
// 		},
// 		{
// 			name: "query with custom account ID",
// 			queryJSON: `{
// 				"queryText": "SELECT count(*) FROM Transaction",
// 				"accountID": 789012
// 			}`,
// 			config: &models.PluginSettings{
// 				Secrets: &models.SecretPluginSettings{
// 					AccountId: 123456,
// 				},
// 			},
// 			executor: &mockNRDBExecutor{
// 				results: &nrdb.NRDBResultContainer{},
// 			},
// 			wantErr: false,
// 		},
// 		{
// 			name: "invalid query JSON",
// 			queryJSON: `{
// 				"queryText": "SELECT count(*) FROM Transaction",
// 				"accountID": "invalid"
// 			}`,
// 			config: &models.PluginSettings{
// 				Secrets: &models.SecretPluginSettings{
// 					AccountId: 123456,
// 				},
// 			},
// 			executor: &mockNRDBExecutor{},
// 			wantErr:  true,
// 		},
// 		{
// 			name: "query execution error",
// 			queryJSON: `{
// 				"queryText": "SELECT count(*) FROM Transaction"
// 			}`,
// 			config: &models.PluginSettings{
// 				Secrets: &models.SecretPluginSettings{
// 					AccountId: 123456,
// 				},
// 			},
// 			executor: &mockNRDBExecutor{
// 				queryErr: errors.New("API error"),
// 			},
// 			wantErr:    true,
// 			errMessage: "NRQL query execution failed",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			query := backend.DataQuery{
// 				RefID: "A",
// 				JSON:  []byte(tt.queryJSON),
// 			}

// 			resp := HandleQuery(context.Background(), tt.executor, tt.config, query)
// 			if tt.wantErr {
// 				assert.Error(t, resp.Error)
// 				if tt.errMessage != "" {
// 					assert.Contains(t, resp.Error.Error(), tt.errMessage)
// 				}
// 			} else {
// 				assert.NoError(t, resp.Error)
// 			}
// 		})
// 	}
// }
