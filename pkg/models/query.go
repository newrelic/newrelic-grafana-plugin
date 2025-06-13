package models

// QueryModel represents the structure of a single query sent from Grafana.
// This struct will be unmarshaled from the JSON data in backend.DataQuery.
type QueryModel struct {
	QueryText string `json:"queryText"`
	AccountID int    `json:"accountID"` // Optional, overrides the default account ID from settings
}
