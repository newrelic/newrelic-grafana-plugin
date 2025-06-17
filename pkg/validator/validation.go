// Package validator provides validation functions for plugin settings and health checks.
// It ensures that configuration parameters are valid and that the New Relic API
// connection is working properly before processing queries.
package validator

import (
	"context"
	"fmt"

	"newrelic-grafana-plugin/pkg/models"
	"newrelic-grafana-plugin/pkg/nrdbiface"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/newrelic/newrelic-client-go/v2/pkg/errors"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
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

// CheckHealth checks the health of the New Relic connection using an NRDB query executor
func CheckHealth(ctx context.Context, settings *models.PluginSettings, executor nrdbiface.NRDBQueryExecutor) (*backend.CheckHealthResult, error) {
	if executor == nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "NRDB query executor is not initialized for health check.",
		}, nil
	}

	// First, perform basic settings validation
	if err := ValidatePluginSettings(settings); err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Plugin configuration validation failed: %s", err.Error()),
		}, nil
	}

	// Try a comprehensive test query to check connectivity and account access
	// This query is more realistic than "SELECT 1" and tests both API connectivity
	// and account-level permissions similar to "SELECT 1 FROM dual" in Oracle
	testQuery := "SELECT count(*) FROM Transaction SINCE 1 hour ago LIMIT 1"

	result, err := executor.QueryWithContext(ctx, settings.Secrets.AccountId, nrdb.NRQL(testQuery))
	if err != nil {
		switch err.(type) {
		case *errors.UnauthorizedError:
			return &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: fmt.Sprintf("Authentication failed for account ID %d. Please verify your API key is correct and has access to this account.", settings.Secrets.AccountId),
			}, nil
		default:
			return &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: fmt.Sprintf("Failed to connect to New Relic API (Account ID: %d). Error: %s", settings.Secrets.AccountId, err.Error()),
			}, nil
		}
	}

	// Verify we got a valid response structure
	if result == nil || len(result.Results) == 0 {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("New Relic API returned an empty response for account ID %d. Please verify the account has data or the account ID is correct.", settings.Secrets.AccountId),
		}, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: fmt.Sprintf("✅ Successfully connected to New Relic API! Account ID %d is accessible and returned %d data point(s). Test query executed successfully.", settings.Secrets.AccountId, len(result.Results)),
	}, nil
}
