// Package validation provides functionality for validating plugin settings and queries.
package validation

import (
	"fmt"

	"source.datanerd.us/after/newrelic-grafana-plugin/pkg/models"
)

// ValidateQuery validates a query model.
// It checks for required fields and valid values.
func ValidateQuery(query *models.QueryModel) error {
	if query == nil {
		return fmt.Errorf("query model cannot be nil")
	}

	if query.QueryText == "" {
		return fmt.Errorf("query text cannot be empty")
	}

	if query.AccountID < 0 {
		return fmt.Errorf("account ID cannot be negative")
	}

	return nil
}

// ValidateQueryRequest validates a query request.
// It checks for required fields and valid values.
func ValidateQueryRequest(request *backend.QueryDataRequest) error {
	if request == nil {
		return fmt.Errorf("query request cannot be nil")
	}

	if len(request.Queries) == 0 {
		return fmt.Errorf("query request must contain at least one query")
	}

	for i, query := range request.Queries {
		if query.RefID == "" {
			return fmt.Errorf("query at index %d must have a RefID", i)
		}
	}

	return nil
} 