package validator

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/newrelic/newrelic-client-go/v2/newrelic"
	nrErrors "github.com/newrelic/newrelic-client-go/v2/pkg/errors"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/models"
)

// ValidatePluginSettings checks the essential settings for the plugin.
func ValidatePluginSettingsOperation(config *models.PluginSettings) error {
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

var ValidatePluginSettings = func(config *models.PluginSettings) error {
	// This function can be mocked in tests to simulate different validation scenarios.
	return ValidatePluginSettingsOperation(config)
}

// CheckHealth performs a comprehensive health check based on plugin settings
// and attempts a connection to the New Relic API.
// It now accepts a New Relic client and context for API interaction.
func CheckHealth(ctx context.Context, config *models.PluginSettings, nrClient *newrelic.NewRelic) (*backend.CheckHealthResult, error) {
	// First, perform basic settings validation
	if err := ValidatePluginSettings(config); err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Data source configuration error: %s", err.Error()),
		}, nil
	}

	// Now, attempt a lightweight API call to New Relic
	// A good health check would be to query for something simple that
	// verifies authentication and basic connectivity without incurring
	// high cost or data transfer. Fetching a single account detail or
	// a simple NRQL query like 'SELECT 1 FROM Dual' (if supported for health)
	// would be ideal.

	testNRQLQuery := "SELECT 1 FROM Dual" // or a lightweight event/metric query if Dual isn't available

	// Ensure the client is not nil before attempting to use it
	if nrClient == nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: "New Relic client is not initialized for health check.",
		}, nil
	}

	// Attempt a query (using the account ID from settings)
	// We don't care about the results, just that the call succeeds.
	_, err := nrClient.Nrdb.QueryWithContext(ctx, config.Secrets.AccountId, nrdb.NRQL(testNRQLQuery))
	if err != nil {

		if _, ok := err.(*nrErrors.UnauthorizedError); ok {

			return &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: "An error occurred with connecting to NewRelic.Could not connect to NewRelic. This usually happens when the API key is incorrect.",
			}, nil
		}

		// If there's an error, it indicates a connectivity or authentication issue.
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("Failed to connect to New Relic API or authenticate. Error: %s", err.Error()),
		}, nil
	}

	// If all checks pass
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Successfully connected to New Relic API.",
	}, nil
}
