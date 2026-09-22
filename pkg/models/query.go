package models

// QueryModel represents the structure of a single query sent from Grafana.
// This struct will be unmarshaled from the JSON data in backend.DataQuery.
type QueryModel struct {
	QueryText      string `json:"queryText"`
	UseGrafanaTime bool   `json:"useGrafanaTime"` // Whether to use Grafana's time picker
	AccountID      int    `json:"accountID"`      // Optional, overrides the default account ID from settings
	TimeoutSeconds int    `json:"timeoutSeconds"` // Optional per-query timeout override in seconds; 0 means unset
}
