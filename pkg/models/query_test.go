package models

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestLoadPluginSettings_Success(t *testing.T) {
	jsonData := `{
		"path": "/some/path"
	}`
	secureData := map[string]string{
		"apiKey":    "test_api_key",
		"accountID": "12345",
	}

	settings := backend.DataSourceInstanceSettings{
		JSONData:                []byte(jsonData),
		DecryptedSecureJSONData: secureData,
	}

	pluginSettings, err := LoadPluginSettings(settings)
	if err != nil {
		t.Fatalf("LoadPluginSettings failed with error: %v", err)
	}

	expectedSettings := &PluginSettings{
		Path: "/some/path",
		Secrets: &SecretPluginSettings{
			ApiKey:    "test_api_key",
			AccountId: 12345,
		},
	}

	if !reflect.DeepEqual(pluginSettings, expectedSettings) {
		t.Errorf("LoadPluginSettings returned %+v, expected %+v", pluginSettings, expectedSettings)
	}
}

func TestLoadPluginSettings_InvalidJSON(t *testing.T) {
	jsonData := `invalid json`
	secureData := map[string]string{
		"apiKey":    "test_api_key",
		"accountId": "12345",
	}

	settings := backend.DataSourceInstanceSettings{
		JSONData:                []byte(jsonData),
		DecryptedSecureJSONData: secureData,
	}

	_, err := LoadPluginSettings(settings)
	if err == nil {
		t.Fatal("LoadPluginSettings did not return error for invalid JSON")
	}

	psErr, ok := err.(*PluginSettingsError)
	if !ok {
		t.Fatalf("Expected PluginSettingsError, got %T", err)
	}
	if psErr.Msg != "could not unmarshal PluginSettings JSON" {
		t.Errorf("Expected error message 'could not unmarshal PluginSettings JSON', got '%s'", psErr.Msg)
	}
	if psErr.Unwrap() == nil {
		t.Error("Expected wrapped error, but got nil")
	}
}

func TestLoadPluginSettings_MissingAPIKey(t *testing.T) {
	jsonData := `{}`
	secureData := map[string]string{
		"accountID": "12345", // Missing apiKey
	}

	settings := backend.DataSourceInstanceSettings{
		JSONData:                []byte(jsonData),
		DecryptedSecureJSONData: secureData,
	}

	_, err := LoadPluginSettings(settings)
	if err == nil {
		t.Fatal("LoadPluginSettings did not return error for missing API key")
	}

	psErr, ok := err.(*PluginSettingsError)
	if !ok {
		t.Fatalf("Expected PluginSettingsError, got %T", err)
	}
	if psErr.Msg != "" { // Top level error has no message, only wrapped error
		t.Errorf("Expected top-level error message '', got '%s'", psErr.Msg)
	}
	// Check the unwrapped error for the specific secure setting error
	unwrappedErr := psErr.Unwrap()
	if unwrappedErr == nil || unwrappedErr.Error() != " Enter New Relic API key." {
		t.Errorf("Expected unwrapped error ' Enter New Relic API key.', got '%v'", unwrappedErr)
	}
}

func TestLoadPluginSettings_MissingAccountID(t *testing.T) {
	jsonData := `{}`
	secureData := map[string]string{
		"apiKey": "test_api_key", // Missing accountID
	}

	settings := backend.DataSourceInstanceSettings{
		JSONData:                []byte(jsonData),
		DecryptedSecureJSONData: secureData,
	}

	_, err := LoadPluginSettings(settings)
	if err == nil {
		t.Fatal("LoadPluginSettings did not return error for missing Account ID")
	}

	psErr, ok := err.(*PluginSettingsError)
	if !ok {
		t.Fatalf("Expected PluginSettingsError, got %T", err)
	}
	if psErr.Msg != "" {
		t.Errorf("Expected top-level error message '', got '%s'", psErr.Msg)
	}
	unwrappedErr := psErr.Unwrap()
	if unwrappedErr == nil || unwrappedErr.Error() != " Enter an account ID. This must be a valid, positive number." {
		t.Errorf("Expected unwrapped error ' Enter an account ID. This must be a valid, positive number.', got '%v'", unwrappedErr)
	}
}

func TestLoadPluginSettings_InvalidAccountID(t *testing.T) {
	jsonData := `{}`
	secureData := map[string]string{
		"apiKey":    "test_api_key",
		"accountID": "not_an_int", // Invalid account ID
	}

	settings := backend.DataSourceInstanceSettings{
		JSONData:                []byte(jsonData),
		DecryptedSecureJSONData: secureData,
	}

	_, err := LoadPluginSettings(settings)
	if err == nil {
		t.Fatal("LoadPluginSettings did not return error for invalid Account ID")
	}

	psErr, ok := err.(*PluginSettingsError)
	if !ok {
		t.Fatalf("Expected PluginSettingsError, got %T", err)
	}
	if psErr.Msg != "" {
		t.Errorf("Expected top-level error message '', got '%s'", psErr.Msg)
	}
	unwrappedErr := psErr.Unwrap()
	if unwrappedErr == nil {
		t.Error("Expected wrapped error, but got nil")
	}
	if !reflect.TypeOf(unwrappedErr).AssignableTo(reflect.TypeOf(&PluginSettingsError{})) {
		t.Errorf("Expected unwrapped error to be PluginSettingsError, got %T", unwrappedErr)
	}
	// Check the message of the nested PluginSettingsError
	nestedPsErr, ok := unwrappedErr.(*PluginSettingsError)
	if !ok || nestedPsErr.Msg != "could not convert accountID 'not_an_int' to int" {
		t.Errorf("Expected nested error message 'could not convert accountID 'not_an_int' to int', got '%v'", unwrappedErr)
	}
	if nestedPsErr.Unwrap() == nil {
		t.Error("Expected nested wrapped error, but got nil")
	}
}

func TestQueryModel_Unmarshal(t *testing.T) {
	jsonStr := `{
		"queryText": "SELECT uniqueCount(session) FROM PageView",
		"accountID": 67890
	}`
	var qm QueryModel
	err := json.Unmarshal([]byte(jsonStr), &qm)
	if err != nil {
		t.Fatalf("Unmarshal QueryModel failed: %v", err)
	}

	expectedQm := QueryModel{
		QueryText: "SELECT uniqueCount(session) FROM PageView",
		AccountID: 67890,
	}

	if !reflect.DeepEqual(qm, expectedQm) {
		t.Errorf("Unmarshal QueryModel got %+v, expected %+v", qm, expectedQm)
	}
}
