// Package validation provides functionality for validating plugin settings and queries.
package validation

import (
	"context"
	"fmt"

	"newrelic-grafana-plugin/pkg/config"
	"newrelic-grafana-plugin/pkg/nrdbiface"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	nrErrors "github.com/newrelic/newrelic-client-go/v2/pkg/errors"
)

// ValidateSettings validates the plugin configuration settings.
// It checks for required fields and valid values.
func ValidateSettings(settings *config.Settings) error {
	if settings == nil {
		return fmt.Errorf("plugin settings cannot be nil")
	}

	if settings.Secrets.ApiKey == "" {
		return fmt.Errorf("New Relic API key cannot be empty")
	}

	if settings.Secrets.AccountId <= 0 {
		return fmt.Errorf("New Relic account ID must be a positive number")
	}

	return nil
}

// CheckHealth performs a health check of the plugin settings.
// It validates the configuration and tests the connection to New Relic.
func CheckHealth(ctx context.Context, settings *config.Settings, executor nrdbiface.NRDBQueryExecutor) (*backend.CheckHealthResult, error) {
	if settings == nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "Plugin settings are not configured",
		}, nil
	}

	if executor == nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "NRDB query executor is not initialized for health check",
		}, nil
	}

	// Try to execute a simple query to verify connection
	_, err := executor.QueryWithContext(ctx, settings.Secrets.AccountId, "SELECT 1")
	if err != nil {
		if _, ok := err.(*nrErrors.UnauthorizedError); ok {
			return &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: "An error occurred with connecting to NewRelic. Could not connect to NewRelic. This usually happens when the API key is incorrect.",
			}, nil
		}
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Failed to connect to New Relic API or authenticate. Error: %v", err),
		}, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Successfully connected to New Relic API.",
	}, nil
}
