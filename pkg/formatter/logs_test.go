package formatter

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
)

func TestFormatLogResults_BasicFormatting(t *testing.T) {
	now := float64(time.Now().UnixMilli())
	results := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{"timestamp": now, "message": "Request started", "level": "info", "hostname": "web-1", "service_name": "api"},
			{"timestamp": now + 1000, "message": "Request failed", "level": "error", "hostname": "web-2", "service_name": "api"},
			{"timestamp": now + 2000, "message": "Debug trace", "level": "debug", "hostname": "web-1", "service_name": "worker"},
		},
	}

	query := backend.DataQuery{RefID: "A", TimeRange: backend.TimeRange{}}
	resp := FormatLogResults(results, query, "SELECT * FROM Log SINCE 1 hour ago")

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if len(resp.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(resp.Frames))
	}

	frame := resp.Frames[0]

	// Check frame name
	if frame.Name != "log-lines" {
		t.Errorf("expected frame name 'log-lines', got '%s'", frame.Name)
	}

	// Check field count
	if len(frame.Fields) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(frame.Fields))
	}

	// Check field names
	expectedFields := []string{"timestamp", "body", "severity", "id", "labels"}
	for i, name := range expectedFields {
		if frame.Fields[i].Name != name {
			t.Errorf("field %d: expected name '%s', got '%s'", i, name, frame.Fields[i].Name)
		}
	}

	// Check metadata
	if frame.Meta == nil {
		t.Fatal("expected frame meta, got nil")
	}
	if frame.Meta.Type != data.FrameTypeLogLines {
		t.Errorf("expected frame type %s, got %s", data.FrameTypeLogLines, frame.Meta.Type)
	}
	if frame.Meta.PreferredVisualization != data.VisTypeLogs {
		t.Errorf("expected preferred vis %s, got %s", data.VisTypeLogs, frame.Meta.PreferredVisualization)
	}
	if frame.Meta.ExecutedQueryString != "SELECT * FROM Log SINCE 1 hour ago" {
		t.Errorf("expected executed query in meta, got '%s'", frame.Meta.ExecutedQueryString)
	}

	// Check body values
	bodies := frame.Fields[1] // body field
	if bodies.Len() != 3 {
		t.Fatalf("expected 3 body values, got %d", bodies.Len())
	}
	if bodies.At(0) != "Request started" {
		t.Errorf("expected 'Request started', got '%v'", bodies.At(0))
	}
	if bodies.At(1) != "Request failed" {
		t.Errorf("expected 'Request failed', got '%v'", bodies.At(1))
	}

	// Check severity values
	severities := frame.Fields[2] // severity field
	if severities.At(0) != "info" {
		t.Errorf("expected severity 'info', got '%v'", severities.At(0))
	}
	if severities.At(1) != "error" {
		t.Errorf("expected severity 'error', got '%v'", severities.At(1))
	}
	if severities.At(2) != "debug" {
		t.Errorf("expected severity 'debug', got '%v'", severities.At(2))
	}
}

func TestFormatLogResults_AlternativeFieldNames(t *testing.T) {
	now := float64(time.Now().UnixMilli())
	results := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{"timestamp": now, "log_message": "Alt message", "log_severity": "WARN"},
		},
	}

	query := backend.DataQuery{RefID: "A"}
	resp := FormatLogResults(results, query, "SELECT * FROM Log")

	frame := resp.Frames[0]
	body := frame.Fields[1].At(0).(string)
	if body != "Alt message" {
		t.Errorf("expected body from log_message, got '%s'", body)
	}

	severity := frame.Fields[2].At(0).(string)
	if severity != "warning" {
		t.Errorf("expected normalized severity 'warning', got '%s'", severity)
	}
}

func TestFormatLogResults_MissingFields(t *testing.T) {
	results := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{"hostname": "web-1"}, // no timestamp, no message, no level
		},
	}

	query := backend.DataQuery{RefID: "A"}
	resp := FormatLogResults(results, query, "SELECT * FROM Log")

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if len(resp.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(resp.Frames))
	}

	frame := resp.Frames[0]
	body := frame.Fields[1].At(0).(string)
	if body != "" {
		t.Errorf("expected empty body, got '%s'", body)
	}
	severity := frame.Fields[2].At(0).(string)
	if severity != "" {
		t.Errorf("expected empty severity, got '%s'", severity)
	}
}

func TestFormatLogResults_EmptyResults(t *testing.T) {
	results := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{},
	}

	query := backend.DataQuery{RefID: "A"}
	resp := FormatLogResults(results, query, "SELECT * FROM Log")

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	if len(resp.Frames) != 0 {
		t.Errorf("expected 0 frames for empty results, got %d", len(resp.Frames))
	}
}

func TestFormatLogResults_LabelsExcludeConsumedFields(t *testing.T) {
	now := float64(time.Now().UnixMilli())
	results := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{
				"timestamp":    now,
				"message":      "Test log",
				"level":        "info",
				"hostname":     "web-1",
				"entity.name":  "my-service",
				"logtype":      "application",
				"custom_field": "custom_value",
			},
		},
	}

	query := backend.DataQuery{RefID: "A"}
	resp := FormatLogResults(results, query, "SELECT * FROM Log")

	frame := resp.Frames[0]
	labelsRaw := frame.Fields[4].At(0).(json.RawMessage)

	var labels map[string]interface{}
	if err := json.Unmarshal(labelsRaw, &labels); err != nil {
		t.Fatalf("failed to parse labels JSON: %v", err)
	}

	// Should contain non-consumed fields
	if _, ok := labels["hostname"]; !ok {
		t.Error("expected 'hostname' in labels")
	}
	if _, ok := labels["entity.name"]; !ok {
		t.Error("expected 'entity.name' in labels")
	}
	if _, ok := labels["logtype"]; !ok {
		t.Error("expected 'logtype' in labels")
	}
	if _, ok := labels["custom_field"]; !ok {
		t.Error("expected 'custom_field' in labels")
	}

	// Should NOT contain consumed fields
	if _, ok := labels["timestamp"]; ok {
		t.Error("'timestamp' should be excluded from labels")
	}
	if _, ok := labels["message"]; ok {
		t.Error("'message' should be excluded from labels")
	}
	if _, ok := labels["level"]; ok {
		t.Error("'level' should be excluded from labels")
	}
}

func TestFormatLogResults_UniqueIDs(t *testing.T) {
	now := float64(time.Now().UnixMilli())
	results := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{"timestamp": now, "message": "log 1"},
			{"timestamp": now + 1, "message": "log 2"},
			{"timestamp": now + 2, "message": "log 3"},
		},
	}

	query := backend.DataQuery{RefID: "A"}
	resp := FormatLogResults(results, query, "SELECT * FROM Log")

	frame := resp.Frames[0]
	ids := frame.Fields[3] // id field

	seen := make(map[string]bool)
	for i := 0; i < ids.Len(); i++ {
		id := ids.At(i).(string)
		if id == "" {
			t.Errorf("row %d: expected non-empty ID", i)
		}
		if seen[id] {
			t.Errorf("row %d: duplicate ID '%s'", i, id)
		}
		seen[id] = true
	}
}

func TestFormatLogResults_IDFromNRGuid(t *testing.T) {
	results := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{"timestamp": float64(time.Now().UnixMilli()), "message": "test", "nr.guid": "abc-123-def"},
		},
	}

	query := backend.DataQuery{RefID: "A"}
	resp := FormatLogResults(results, query, "SELECT * FROM Log")

	id := resp.Frames[0].Fields[3].At(0).(string)
	if id != "abc-123-def" {
		t.Errorf("expected ID 'abc-123-def' from nr.guid, got '%s'", id)
	}
}

func TestExtractLogSeverity_Normalization(t *testing.T) {
	tests := []struct {
		input    map[string]interface{}
		expected string
	}{
		{map[string]interface{}{"level": "INFO"}, "info"},
		{map[string]interface{}{"level": "ERROR"}, "error"},
		{map[string]interface{}{"level": "WARN"}, "warning"},
		{map[string]interface{}{"level": "warn"}, "warning"},
		{map[string]interface{}{"level": "FATAL"}, "critical"},
		{map[string]interface{}{"level": "crit"}, "critical"},
		{map[string]interface{}{"level": "emerg"}, "critical"},
		{map[string]interface{}{"level": "err"}, "error"},
		{map[string]interface{}{"level": "notice"}, "info"},
		{map[string]interface{}{"level": "DEBUG"}, "debug"},
		{map[string]interface{}{"severity": "warning"}, "warning"},
		{map[string]interface{}{}, ""},
	}

	for _, tt := range tests {
		got := extractLogSeverity(tt.input)
		if got != tt.expected {
			t.Errorf("extractLogSeverity(%v) = '%s', expected '%s'", tt.input, got, tt.expected)
		}
	}
}
