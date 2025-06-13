package models

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// PluginSettingsError represents an error specifically related to plugin settings.
type PluginSettingsError struct {
	Msg string
	Err error // Wrapped error
}

func (e *PluginSettingsError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("plugin settings error: %s: %v", e.Msg, e.Err)
	}
	return fmt.Sprintf("plugin settings error: %s", e.Msg)
}

func (e *PluginSettingsError) Unwrap() error {
	return e.Err
}

// PluginSettings holds the configuration settings for the New Relic data source.
type PluginSettings struct {
	Path    string                `json:"path"`
	Secrets *SecretPluginSettings `json:"-"`
}

// SecretPluginSettings holds sensitive data like API keys and Account IDs.
type SecretPluginSettings struct {
	ApiKey    string `json:"apiKey"`
	AccountId int    `json:"accountID"`
}

// LoadPluginSettings unmarshals the JSON data and decrypted secure JSON data
// from Grafana's DataSourceInstanceSettings into a PluginSettings struct.
func LoadPluginSettings(source backend.DataSourceInstanceSettings) (*PluginSettings, error) {
	settings := PluginSettings{}
	err := json.Unmarshal(source.JSONData, &settings)
	if err != nil {
		return nil, &PluginSettingsError{Msg: "could not unmarshal PluginSettings JSON", Err: err}
	}

	secretSettings, err := loadSecretPluginSettings(source.DecryptedSecureJSONData)

	if err != nil {
		return nil, &PluginSettingsError{Msg: "could not load secure plugin settings", Err: err}
	}

	settings.Secrets = secretSettings

	return &settings, nil
}

// loadSecretPluginSettings extracts secure data from the decrypted map.
func loadSecretPluginSettings(source map[string]string) (*SecretPluginSettings, error) {

	apiKey := source["apiKey"]
	if apiKey == "" {
		return nil, &PluginSettingsError{Msg: "API key is missing in secure settings"}
	}

	accountIdStr := source["accountID"]
	if accountIdStr == "" {
		return nil, &PluginSettingsError{Msg: "Account ID is missing in secure settings"}
	}

	accountId, err := strconv.Atoi(accountIdStr)
	if err != nil {
		return nil, &PluginSettingsError{Msg: fmt.Sprintf("could not convert accountID '%s' to int", accountIdStr), Err: err}
	}

	return &SecretPluginSettings{
		ApiKey:    apiKey,
		AccountId: accountId,
	}, nil
}
