// Package config provides configuration management for the New Relic Grafana plugin.
// It handles loading and validating plugin settings from Grafana.
package config

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// SettingsError represents an error specifically related to plugin settings.
type SettingsError struct {
	Msg string
	Err error // Wrapped error
}

func (e *SettingsError) Error() string {
	if e.Err != nil {
		if e.Msg != "" {
			return fmt.Sprintf("%s: %v", e.Msg, e.Err)
		}
		return fmt.Sprintf("%v", e.Err)
	}
	return fmt.Sprintf(" %s", e.Msg)
}

func (e *SettingsError) Unwrap() error {
	return e.Err
}

// Settings holds the configuration settings for the New Relic data source.
type Settings struct {
	Path    string                `json:"path"`
	Secrets *SecretSettings `json:"-"`
}

// SecretSettings holds sensitive data like API keys and Account IDs.
type SecretSettings struct {
	ApiKey    string `json:"apiKey"`
	AccountId int    `json:"accountID"`
}

// LoadSettings unmarshals the JSON data and decrypted secure JSON data
// from Grafana's DataSourceInstanceSettings into a Settings struct.
func LoadSettings(source backend.DataSourceInstanceSettings) (*Settings, error) {
	settings := Settings{}
	err := json.Unmarshal(source.JSONData, &settings)
	if err != nil {
		return nil, &SettingsError{Msg: "could not unmarshal Settings JSON", Err: err}
	}

	secretSettings, err := loadSecretSettings(source.DecryptedSecureJSONData)
	if err != nil {
		return nil, &SettingsError{Err: err}
	}

	settings.Secrets = secretSettings
	return &settings, nil
}

// loadSecretSettings extracts secure data from the decrypted map.
func loadSecretSettings(source map[string]string) (*SecretSettings, error) {
	apiKey := source["apiKey"]
	if apiKey == "" {
		return nil, &SettingsError{Msg: "Enter New Relic API key."}
	}

	accountIdStr := source["accountID"]
	if accountIdStr == "" {
		return nil, &SettingsError{Msg: "Enter an account ID. This must be a valid, positive number."}
	}

	accountId, err := strconv.Atoi(accountIdStr)
	if err != nil {
		return nil, &SettingsError{Msg: fmt.Sprintf("could not convert accountID '%s' to int", accountIdStr), Err: err}
	}

	return &SecretSettings{
		ApiKey:    apiKey,
		AccountId: accountId,
	}, nil
} 