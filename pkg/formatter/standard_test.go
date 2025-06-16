package formatter

import (
	"testing"
	"time"

	"newrelic-grafana-plugin/pkg/constant"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatStandardQuery(t *testing.T) {
	now := time.Now()
	query := backend.DataQuery{
		TimeRange: backend.TimeRange{
			From: now.Add(-1 * time.Hour),
			To:   now,
		},
	}

	tests := []struct {
		name     string
		results  *nrdb.NRDBResultContainer
		query    backend.DataQuery
		validate func(t *testing.T, resp *backend.DataResponse)
	}{
		{
			name: "single result",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"timestamp": now.Unix(),
						"value":     42.0,
					},
				},
			},
			query: query,
			validate: func(t *testing.T, resp *backend.DataResponse) {
				require.Len(t, resp.Frames, 1)
				frame := resp.Frames[0]
				assert.Equal(t, constant.StandardResponseFrameName, frame.Name)
				assert.Equal(t, data.VisTypeGraph, frame.Meta.PreferredVisualization)
				require.Len(t, frame.Fields, 2)
				assert.Equal(t, constant.TimeFieldName, frame.Fields[0].Name)
				assert.Equal(t, "value", frame.Fields[1].Name)
			},
		},
		{
			name: "multiple results",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{
						"timestamp": now.Unix(),
						"value":     42.0,
					},
					{
						"timestamp": now.Add(time.Minute).Unix(),
						"value":     43.0,
					},
				},
			},
			query: query,
			validate: func(t *testing.T, resp *backend.DataResponse) {
				require.Len(t, resp.Frames, 1)
				frame := resp.Frames[0]
				assert.Equal(t, constant.StandardResponseFrameName, frame.Name)
				assert.Equal(t, data.VisTypeGraph, frame.Meta.PreferredVisualization)
				require.Len(t, frame.Fields, 2)
				assert.Equal(t, constant.TimeFieldName, frame.Fields[0].Name)
				assert.Equal(t, "value", frame.Fields[1].Name)
				assert.Equal(t, 2, frame.Fields[0].Len())
				assert.Equal(t, 2, frame.Fields[1].Len())
			},
		},
		{
			name: "empty results",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{},
			},
			query: query,
			validate: func(t *testing.T, resp *backend.DataResponse) {
				require.Empty(t, resp.Frames)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := formatStandardQuery(tt.results, tt.query)
			tt.validate(t, resp)
		})
	}
}
