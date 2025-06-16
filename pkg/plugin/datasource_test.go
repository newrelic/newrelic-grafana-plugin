package plugin

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDatasource(t *testing.T) {
	ds, err := NewDatasource(context.Background(), backend.DataSourceInstanceSettings{})
	require.NoError(t, err)
	assert.NotNil(t, ds)
}

func TestDispose(t *testing.T) {
	ds := &Datasource{}
	// Should not panic
	ds.Dispose()
}

func TestQueryData(t *testing.T) {
	tests := []struct {
		name    string
		queries []backend.DataQuery
		wantErr bool
	}{
		{
			name:    "empty queries",
			queries: []backend.DataQuery{},
			wantErr: false,
		},
		{
			name: "single query",
			queries: []backend.DataQuery{
				{RefID: "A"},
			},
			wantErr: false,
		},
		{
			name: "multiple queries",
			queries: []backend.DataQuery{
				{RefID: "A"},
				{RefID: "B"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Datasource{}
			resp, err := ds.QueryData(
				context.Background(),
				&backend.QueryDataRequest{
					Queries: tt.queries,
				},
			)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, len(tt.queries), len(resp.Responses))
			}
		})
	}
}

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name     string
		settings backend.DataSourceInstanceSettings
		wantErr  bool
	}{
		{
			name: "valid settings",
			settings: backend.DataSourceInstanceSettings{
				JSONData: []byte(`{"apiKey": "test-key"}`),
			},
			wantErr: false,
		},
		{
			name: "invalid settings",
			settings: backend.DataSourceInstanceSettings{
				JSONData: []byte(`{}`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := &Datasource{}
			resp, err := ds.CheckHealth(
				context.Background(),
				&backend.CheckHealthRequest{
					PluginContext: backend.PluginContext{
						DataSourceInstanceSettings: &tt.settings,
					},
				},
			)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}
