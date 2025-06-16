package dataformatter

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

func TestIsCountQuery(t *testing.T) {
	tests := []struct {
		name     string
		results  *nrdb.NRDBResultContainer
		expected bool
	}{
		{
			name: "simple count query",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{"count": 42.0},
				},
			},
			expected: true,
		},
		{
			name: "not a count query",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{"value": 42.0},
				},
			},
			expected: false,
		},
		{
			name: "empty results",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCountQuery(tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatQueryResults(t *testing.T) {
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
			name: "simple count query",
			results: &nrdb.NRDBResultContainer{
				Results: []nrdb.NRDBResult{
					{"count": 42.0},
				},
			},
			query: query,
			validate: func(t *testing.T, resp *backend.DataResponse) {
				require.Len(t, resp.Frames, 2)
				assert.Equal(t, "count", resp.Frames[0].Name)
				assert.Equal(t, data.VisType(data.VisTypeTable), resp.Frames[0].Meta.PreferredVisualization)
				assert.Equal(t, constant.CountTimeSeriesFrameName, resp.Frames[1].Name)
				assert.Equal(t, data.VisTypeGraph, resp.Frames[1].Meta.PreferredVisualization)
			},
		},
		{
			name: "faceted count query",
			results: &nrdb.NRDBResultContainer{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"service"},
				},
				Results: []nrdb.NRDBResult{
					{
						"count": 42.0,
						"facet": []interface{}{"service1"},
					},
					{
						"count": 24.0,
						"facet": []interface{}{"service2"},
					},
				},
			},
			query: query,
			validate: func(t *testing.T, resp *backend.DataResponse) {
				require.Len(t, resp.Frames, 2)
				assert.Equal(t, constant.FacetedFrameName, resp.Frames[0].Name)
				assert.Equal(t, data.VisType(data.VisTypeTable), resp.Frames[0].Meta.PreferredVisualization)
				assert.Equal(t, constant.FacetedTimeSeriesFrameName, resp.Frames[1].Name)
				assert.Equal(t, data.VisTypeGraph, resp.Frames[1].Meta.PreferredVisualization)
			},
		},
		{
			name: "standard query",
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
				assert.Equal(t, constant.StandardResponseFrameName, resp.Frames[0].Name)
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
			resp := FormatQueryResults(tt.results, tt.query)
			tt.validate(t, resp)
		})
	}
}

func TestExtractCountValue(t *testing.T) {
	tests := []struct {
		name     string
		result   nrdb.NRDBResult
		expected float64
	}{
		{
			name:     "valid count",
			result:   nrdb.NRDBResult{"count": 42.0},
			expected: 42.0,
		},
		{
			name:     "invalid count type",
			result:   nrdb.NRDBResult{"count": "42"},
			expected: 0.0,
		},
		{
			name:     "missing count",
			result:   nrdb.NRDBResult{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCountValue(tt.result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFacetNames(t *testing.T) {
	tests := []struct {
		name     string
		results  *nrdb.NRDBResultContainer
		expected []string
	}{
		{
			name: "with facets",
			results: &nrdb.NRDBResultContainer{
				Metadata: nrdb.NRDBMetadata{
					Facets: []string{"service", "host"},
				},
			},
			expected: []string{"service", "host"},
		},
		{
			name: "no facets",
			results: &nrdb.NRDBResultContainer{
				Metadata: nrdb.NRDBMetadata{},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFacetNames(tt.results)
			assert.Equal(t, tt.expected, result)
		})
	}
}
