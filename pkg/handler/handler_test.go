package handler

import (
	"context"
	"testing"

	"newrelic-grafana-plugin/pkg/models"
	"newrelic-grafana-plugin/pkg/nrdbiface"
	"newrelic-grafana-plugin/pkg/testutil"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleQuery(t *testing.T) {
	tests := []struct {
		name           string
		query          backend.DataQuery
		executor       nrdbiface.NRDBQueryExecutor
		config         *models.PluginSettings
		expectedError  bool
		expectedFrames int
	}{
		{
			name:     "valid NRQL query",
			query:    testutil.CreateTestQuery(t, "A", "SELECT count(*) FROM Transaction"),
			executor: &mockNRDBExecutor{},
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					ApiKey:    "test-api-key",
					AccountId: 123456,
				},
			},
			expectedError:  false,
			expectedFrames: 2, // Count queries return 2 frames (table and time series)
		},
		{
			name:     "invalid query",
			query:    testutil.CreateTestQuery(t, "A", "INVALID QUERY"),
			executor: &mockNRDBExecutor{},
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					ApiKey:    "test-api-key",
					AccountId: 123456,
				},
			},
			expectedError:  false, // The query will still execute, as NRQL validation is not done locally
			expectedFrames: 2,
		},
		{
			name:     "executor error",
			query:    testutil.CreateTestQuery(t, "A", "SELECT count(*) FROM Transaction"),
			executor: &mockNRDBExecutor{queryErr: assert.AnError},
			config: &models.PluginSettings{
				Secrets: &models.SecretPluginSettings{
					ApiKey:    "test-api-key",
					AccountId: 123456,
				},
			},
			expectedError:  true,
			expectedFrames: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := HandleQuery(context.Background(), tt.executor, tt.config, tt.query)
			if tt.expectedError {
				assert.Error(t, resp.Error)
				assert.Nil(t, resp.Frames)
			} else {
				assert.NoError(t, resp.Error)
				assert.NotNil(t, resp.Frames)
				assert.Len(t, resp.Frames, tt.expectedFrames)
			}
		})
	}
}

func TestHandleQuery_TimeRange(t *testing.T) {
	query := testutil.CreateTestQuery(t, "A", "SELECT count(*) FROM Transaction")
	executor := &mockNRDBExecutor{}
	config := &models.PluginSettings{
		Secrets: &models.SecretPluginSettings{
			ApiKey:    "test-api-key",
			AccountId: 123456,
		},
	}

	resp := HandleQuery(context.Background(), executor, config, query)
	require.NoError(t, resp.Error)
	require.Len(t, resp.Frames, 2) // Count queries return 2 frames

	// Check the time series frame (second frame)
	timeSeriesFrame := resp.Frames[1]
	require.NotNil(t, timeSeriesFrame)
	require.Len(t, timeSeriesFrame.Fields, 2) // Time and value fields

	// Verify time field
	timeField := timeSeriesFrame.Fields[0]
	assert.Equal(t, "time", timeField.Name)
}

func TestHandleQuery_MultipleQueries(t *testing.T) {
	queries := []backend.DataQuery{
		testutil.CreateTestQuery(t, "A", "SELECT count(*) FROM Transaction"),
		testutil.CreateTestQuery(t, "B", "SELECT average(duration) FROM Transaction"),
	}

	executor := &mockNRDBExecutor{}
	config := &models.PluginSettings{
		Secrets: &models.SecretPluginSettings{
			ApiKey:    "test-api-key",
			AccountId: 123456,
		},
	}

	for _, query := range queries {
		resp := HandleQuery(context.Background(), executor, config, query)
		require.NoError(t, resp.Error)
		require.NotEmpty(t, resp.Frames)
	}
}
