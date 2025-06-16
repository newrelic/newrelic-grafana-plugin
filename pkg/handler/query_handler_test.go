package handler

import (
	"context"
	"errors"
	"testing"

	"newrelic-grafana-plugin/pkg/models"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	"github.com/stretchr/testify/assert"
)

// mockNRDBClient embeds nrdb.Nrdb and overrides QueryWithContext
type mockNRDBClient struct {
	nrdb.Nrdb
	queryErr error
	results  *nrdb.NRDBResultContainer
}

func (m *mockNRDBClient) QueryWithContext(ctx context.Context, accountID int, query nrdb.NRQL) (*nrdb.NRDBResultContainer, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return m.results, nil
}

func TestExecuteNRQLQuery(t *testing.T) {
	tests := []struct {
		name       string
		client     *mockNRDBClient
		accountID  int
		query      string
		wantErr    bool
		errMessage string
	}{
		{
			name:      "successful query",
			client:    &mockNRDBClient{results: &nrdb.NRDBResultContainer{}},
			accountID: 123456,
			query:     "SELECT count(*) FROM Transaction",
			wantErr:   false,
		},
		{
			name:       "nil client",
			client:     nil,
			accountID:  123456,
			query:      "SELECT count(*) FROM Transaction",
			wantErr:    true,
			errMessage: "New Relic client is nil, cannot execute query",
		},
		{
			name:       "empty query",
			client:     &mockNRDBClient{},
			accountID:  123456,
			query:      "",
			wantErr:    true,
			errMessage: "NRQL query text cannot be empty",
		},
		{
			name:       "invalid account ID",
			client:     &mockNRDBClient{},
			accountID:  0,
			query:      "SELECT count(*) FROM Transaction",
			wantErr:    true,
			errMessage: "New Relic account ID cannot be 0",
		},
		{
			name:       "query error",
			client:     &mockNRDBClient{queryErr: errors.New("API error")},
			accountID:  123456,
			query:      "SELECT count(*) FROM Transaction",
			wantErr:    true,
			errMessage: "error from New Relic API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var client *newrelic.NewRelic
			if tt.client != nil {
				client = &newrelic.NewRelic{
					Nrdb: tt.client,
				}
			}
			results, err := ExecuteNRQLQuery(context.Background(), client, tt.accountID, tt.query)
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

func TestHandleQuery(t *testing.T) {
	tests := []struct {
		name       string
		queryJSON  string
		config     *models.PluginSettings
		client     *mockNRDBClient
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
			client: &mockNRDBClient{
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
			client: &mockNRDBClient{
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
			client:  &mockNRDBClient{},
			wantErr: true,
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
			client: &mockNRDBClient{
				queryErr: errors.New("API error"),
			},
			wantErr:    true,
			errMessage: "NRQL query execution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := backend.DataQuery{
				RefID: "A",
				JSON:  []byte(tt.queryJSON),
			}

			client := &newrelic.NewRelic{
				Nrdb: tt.client,
			}
			resp := HandleQuery(context.Background(), client, tt.config, query)
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
