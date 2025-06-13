package dataformatter

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
)

func TestIsCountQuery(t *testing.T) {
	// Test case 1: Is a count query
	countResults := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{"count": 123.0},
		},
	}
	if !IsCountQuery(countResults) {
		t.Error("Expected IsCountQuery to return true for a count query")
	}

	// Test case 2: Not a count query (different field)
	otherResults := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{"value": 45.0},
		},
	}
	if IsCountQuery(otherResults) {
		t.Error("Expected IsCountQuery to return false for a non-count query")
	}

	// Test case 3: Not a count query (multiple results)
	multiResults := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{"count": 1.0},
			{"count": 2.0},
		},
	}
	if IsCountQuery(multiResults) {
		t.Error("Expected IsCountQuery to return false for multiple results even with 'count'")
	}

	// Test case 4: Empty results
	emptyResults := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{},
	}
	if IsCountQuery(emptyResults) {
		t.Error("Expected IsCountQuery to return false for empty results")
	}

	// Test case 5: Nil results container
	if IsCountQuery(nil) {
		t.Error("Expected IsCountQuery to return false for nil results container")
	}
}

func TestFormatCountQueryResults(t *testing.T) {
	countResults := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{"count": 123.45},
		},
	}

	resp := FormatCountQueryResults(countResults)

	if resp == nil {
		t.Fatal("FormatCountQueryResults returned nil response")
	}
	if len(resp.Frames) != 1 {
		t.Fatalf("Expected 1 frame, got %d", len(resp.Frames))
	}

	frame := resp.Frames[0]
	if frame.Name != "" { // Count frames usually don't have a name set
		t.Errorf("Expected frame name to be empty, got '%s'", frame.Name)
	}
	if len(frame.Fields) != 1 {
		t.Fatalf("Expected 1 field in frame, got %d", len(frame.Fields))
	}

	field := frame.Fields[0]
	if field.Name != "count" {
		t.Errorf("Expected field name 'count', got '%s'", field.Name)
	}
	if field.Type() != data.FieldTypeFloat64 {
		t.Errorf("Expected field type Float64, got %s", field.Type().String())
	}
	if field.Len() != 1 {
		t.Errorf("Expected field length 1, got %d", field.Len())
	}
	if val := field.At(0).(float64); val != 123.45 {
		t.Errorf("Expected count value 123.45, got %f", val)
	}
}

func TestFormatRegularQueryResults_TimeSeries(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond) // Truncate for consistent comparison
	nrResults := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{"timestamp": float64(now.UnixMilli()), "value": 10.5, "name": "A"},
			{"timestamp": float64(now.Add(time.Minute).UnixMilli()), "value": 20.1, "name": "B"},
		},
	}
	query := backend.DataQuery{
		TimeRange: backend.TimeRange{From: now, To: now.Add(2 * time.Minute)},
	}

	resp := FormatRegularQueryResults(nrResults, query)

	if resp == nil {
		t.Fatal("FormatRegularQueryResults returned nil response")
	}
	if len(resp.Frames) != 1 {
		t.Fatalf("Expected 1 frame, got %d", len(resp.Frames))
	}

	frame := resp.Frames[0]
	if frame.Name != "response" {
		t.Errorf("Expected frame name 'response', got '%s'", frame.Name)
	}
	if len(frame.Fields) != 3 { // time, value, name
		t.Fatalf("Expected 3 fields, got %d", len(frame.Fields))
	}

	// Check Time field
	timeField := frame.Fields[0]
	if timeField.Name != "time" || timeField.Type() != data.FieldTypeTime {
		t.Errorf("Expected time field, got %s:%s", timeField.Name, timeField.Type().String())
	}
	expectedTimes := []time.Time{
		time.Unix(now.Unix(), 0),
		time.Unix(now.Add(time.Minute).Unix(), 0),
	}
	for i := 0; i < timeField.Len(); i++ {
		if !timeField.At(i).(time.Time).Equal(expectedTimes[i]) {
			t.Errorf("Time field at index %d: Expected %v, got %v", i, expectedTimes[i], timeField.At(i))
		}
	}

	// Check Value field
	valueField := frame.Fields[1] // Order might vary based on map iteration, check names
	if valueField.Name != "value" || valueField.Type() != data.FieldTypeFloat64 {
		// Attempt to find by name if order isn't guaranteed
		for _, f := range frame.Fields {
			if f.Name == "value" {
				valueField = f
				break
			}
		}
		if valueField.Name != "value" || valueField.Type() != data.FieldTypeFloat64 {
			t.Errorf("Expected value field (float64), got %s:%s", valueField.Name, valueField.Type().String())
		}
	}
	expectedValues := []float64{10.5, 20.1}
	for i := 0; i < valueField.Len(); i++ {
		if valueField.At(i).(float64) != expectedValues[i] {
			t.Errorf("Value field at index %d: Expected %f, got %f", i, expectedValues[i], valueField.At(i))
		}
	}

	// Check Name field
	nameField := frame.Fields[2] // Order might vary
	if nameField.Name != "name" || nameField.Type() != data.FieldTypeString {
		for _, f := range frame.Fields {
			if f.Name == "name" {
				nameField = f
				break
			}
		}
		if nameField.Name != "name" || nameField.Type() != data.FieldTypeString {
			t.Errorf("Expected name field (string), got %s:%s", nameField.Name, nameField.Type().String())
		}
	}
	expectedNames := []string{"A", "B"}
	for i := 0; i < nameField.Len(); i++ {
		if nameField.At(i).(string) != expectedNames[i] {
			t.Errorf("Name field at index %d: Expected %s, got %s", i, expectedNames[i], nameField.At(i))
		}
	}
}

func TestFormatRegularQueryResults_MixedTypes(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	nrResults := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{"timestamp": float64(now.UnixMilli()), "cpu": 50.5, "is_active": true, "status": "running"},
			{"timestamp": float64(now.Add(time.Minute).UnixMilli()), "cpu": 60.0, "is_active": false, "status": "stopped"},
			{"timestamp": float64(now.Add(2 * time.Minute).UnixMilli()), "cpu": 70.1, "is_active": true, "status": nil, "mixed": 123}, // nil string, int for mixed
		},
	}
	query := backend.DataQuery{
		TimeRange: backend.TimeRange{From: now, To: now.Add(3 * time.Minute)},
	}

	resp := FormatRegularQueryResults(nrResults, query)
	if resp == nil {
		t.Fatal("FormatRegularQueryResults returned nil response")
	}
	if len(resp.Frames) != 1 {
		t.Fatalf("Expected 1 frame, got %d", len(resp.Frames))
	}
	frame := resp.Frames[0]

	// Helper to get field by name
	getField := func(name string) *data.Field {
		for _, f := range frame.Fields {
			if f.Name == name {
				return f
			}
		}
		return nil
	}

	// Test CPU (float64)
	cpuField := getField("cpu")
	if cpuField == nil || cpuField.Type() != data.FieldTypeFloat64 {
		t.Errorf("CPU field: Expected float64, got %v (%T)", cpuField.Type(), cpuField.Type())
	}
	expectedCPUs := []float64{50.5, 60.0, 70.1}
	for i, val := range expectedCPUs {
		if cpuField.At(i).(float64) != val {
			t.Errorf("CPU[%d]: Expected %f, got %f", i, val, cpuField.At(i).(float64))
		}
	}

	// Test IsActive (bool)
	isActiveField := getField("is_active")
	if isActiveField == nil || isActiveField.Type() != data.FieldTypeBool {
		t.Errorf("IsActive field: Expected bool, got %v (%T)", isActiveField.Type(), isActiveField.Type())
	}
	expectedActive := []bool{true, false, true}
	for i, val := range expectedActive {
		if isActiveField.At(i).(bool) != val {
			t.Errorf("IsActive[%d]: Expected %t, got %t", i, val, isActiveField.At(i).(bool))
		}
	}

	// Test Status (string, including nil conversion)
	statusField := getField("status")
	if statusField == nil || statusField.Type() != data.FieldTypeString {
		t.Errorf("Status field: Expected string, got %v (%T)", statusField.Type(), statusField.Type())
	}
	expectedStatus := []string{"running", "stopped", ""} // nil becomes empty string
	for i, val := range expectedStatus {
		if statusField.At(i).(string) != val {
			t.Errorf("Status[%d]: Expected '%s', got '%s'", i, val, statusField.At(i).(string))
		}
	}

	// Test Mixed (converted to string)
	mixedField := getField("mixed")
	if mixedField == nil || mixedField.Type() != data.FieldTypeString {
		t.Errorf("Mixed field: Expected string (due to default), got %v (%T)", mixedField.Type(), mixedField.Type())
	}
	// The first two results don't have "mixed", so they will be empty strings.
	// The third has 123 (float64 from JSON unmarshal if not explicitly handled as int, then stringified)
	expectedMixed := []string{"", "", "123"}
	for i, val := range expectedMixed {
		if mixedField.At(i).(string) != val {
			t.Errorf("Mixed[%d]: Expected '%s', got '%s'", i, val, mixedField.At(i).(string))
		}
	}
}

func TestFormatRegularQueryResults_NoResults(t *testing.T) {
	nrResults := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{},
	}
	query := backend.DataQuery{}

	resp := FormatRegularQueryResults(nrResults, query)
	if resp == nil {
		t.Fatal("FormatRegularQueryResults returned nil response")
	}
	if len(resp.Frames) != 0 {
		t.Errorf("Expected 0 frames for no results, got %d", len(resp.Frames))
	}
}

func TestFormatRegularQueryResults_NoTimestamp(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	nrResults := &nrdb.NRDBResultContainer{
		Results: []nrdb.NRDBResult{
			{"value": 10.0}, // No timestamp
			{"value": 20.0},
		},
	}
	query := backend.DataQuery{
		TimeRange: backend.TimeRange{From: now, To: now.Add(time.Minute)},
	}

	resp := FormatRegularQueryResults(nrResults, query)
	if resp == nil {
		t.Fatal("FormatRegularQueryResults returned nil response")
	}
	if len(resp.Frames) != 1 {
		t.Fatalf("Expected 1 frame, got %d", len(resp.Frames))
	}

	frame := resp.Frames[0]
	timeField := frame.Fields[0]
	if timeField.Name != "time" || timeField.Type() != data.FieldTypeTime {
		t.Errorf("Expected time field, got %s:%s", timeField.Name, timeField.Type().String())
	}

	// Expect all timestamps to be the query's From time
	expectedTime := query.TimeRange.From
	for i := 0; i < timeField.Len(); i++ {
		if !timeField.At(i).(time.Time).Equal(expectedTime) {
			t.Errorf("Time field at index %d: Expected %v, got %v", i, expectedTime, timeField.At(i))
		}
	}
}
