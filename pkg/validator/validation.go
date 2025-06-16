package validator

import (
	"context"

	"newrelic-grafana-plugin/pkg/models"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	"github.com/newrelic/newrelic-client-go/v2/pkg/errors"
)

// ValidatePluginSettings validates the plugin settings
func ValidatePluginSettings(settings *models.PluginSettings) error {
	if settings == nil {
		return &models.PluginSettingsError{Msg: "plugin settings cannot be nil"}
	}

	if settings.Secrets == nil {
		return &models.PluginSettingsError{Msg: "plugin secrets cannot be nil"}
	}

	if settings.Secrets.ApiKey == "" {
		return &models.PluginSettingsError{Msg: "API key cannot be empty"}
	}

	if settings.Secrets.AccountId <= 0 {
		return &models.PluginSettingsError{Msg: "account ID must be a positive number"}
	}

	return nil
}

// CheckHealth checks the health of the New Relic connection
func CheckHealth(ctx context.Context, settings *models.PluginSettings, client *newrelic.NewRelic) (*backend.CheckHealthResult, error) {
	if client == nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "New Relic client is not initialized for health check.",
		}, nil
	}

	// Try a simple query to check connectivity
	_, err := client.Nrdb.QueryWithContext(ctx, settings.Secrets.AccountId, "SELECT 1")
	if err != nil {
		switch err.(type) {
		case *errors.UnauthorizedError:
			return &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: "An error occurred with connecting to NewRelic.Could not connect to NewRelic. This usually happens when the API key is incorrect.",
			}, nil
		default:
			return &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: "Failed to connect to New Relic API or authenticate. Error: " + err.Error(),
			}, nil
		}
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Successfully connected to New Relic API.",
	}, nil
}
