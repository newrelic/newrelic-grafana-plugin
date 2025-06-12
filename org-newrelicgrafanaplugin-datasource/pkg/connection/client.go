package connection

import (
	"fmt"

	"github.com/newrelic/newrelic-client-go/newrelic"
)

// NewRelicClientError represents an error specifically related to New Relic client operations.
type NewRelicClientError struct {
	Msg string
	Err error // Wrapped error
}

func (e *NewRelicClientError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("new relic client error: %s: %v", e.Msg, e.Err)
	}
	return fmt.Sprintf("new relic client error: %s", e.Msg)
}

func (e *NewRelicClientError) Unwrap() error {
	return e.Err
}

// GetClient initializes and returns a New Relic client using the provided API key.
func GetClient(apiKey string) (*newrelic.NewRelic, error) {
	if apiKey == "" {
		return nil, &NewRelicClientError{Msg: "New Relic API key cannot be empty"}
	}

	client, err := newrelic.New(newrelic.ConfigPersonalAPIKey(apiKey))
	if err != nil {
		return nil, &NewRelicClientError{Msg: "failed to initialize New Relic client", Err: err}
	}
	return client, nil
}
