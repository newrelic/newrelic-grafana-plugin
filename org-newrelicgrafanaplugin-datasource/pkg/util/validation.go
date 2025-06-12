package util

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/org/newrelic-grafana-plugin/pkg/models"
)

// ValidatePluginSettings checks the essential settings for the plugin.
func ValidatePluginSettings(config *models.PluginSettings) error {
	if config == nil || config.Secrets == nil {
		return fmt.Errorf("plugin settings or secrets are not loaded")
	}
	if config.Secrets.ApiKey == "" {
		return fmt.Errorf("API key is missing in plugin settings")
	}
	if config.Secrets.AccountId == 0 { // Assuming 0 is an invalid account ID
		return fmt.Errorf("new relic account ID is missing or invalid in plugin settings")
	}
	return nil
}

// CheckHealth provides a standardized health check response for Grafana.
func CheckHealth(config *models.PluginSettings) (*backend.CheckHealthResult, error) {
	if err := ValidatePluginSettings(config); err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Data source configuration error: %s", err.Error()),
		}, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Data source is working",
	}, nil
}
