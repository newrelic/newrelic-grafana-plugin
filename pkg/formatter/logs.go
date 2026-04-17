// Package formatter — log-specific formatting for Grafana's Logs panel.
// Converts New Relic Log event results into Grafana log-lines data frames
// with proper metadata so they render in the Logs panel with severity
// color-coding, expandable labels, and log search.
package formatter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/newrelic/newrelic-client-go/v2/pkg/nrdb"
)

// Fields consumed by the log formatter and excluded from the labels JSON.
var logConsumedFields = map[string]bool{
	// Timestamp
	"timestamp": true,
	// Message variants
	"message": true, "log_message": true, "messageBody": true, "msg": true,
	// Severity variants
	"level": true, "severity": true, "log_severity": true, "log.level": true,
	// ID fields
	"messageId": true, "nr.guid": true,
	// Internal NR metadata
	"inspectedCount": true, "beginTimeSeconds": true, "endTimeSeconds": true,
	"eventType": true,
}

// severityAliases maps non-canonical severity strings to Grafana canonical levels.
var severityAliases = map[string]string{
	"fatal":         "critical",
	"emerg":         "critical",
	"emergency":     "critical",
	"alert":         "critical",
	"crit":          "critical",
	"err":           "error",
	"warn":          "warning",
	"notice":        "info",
	"informational": "info",
	"dbug":          "debug",
}

// FormatLogResults converts NR Log query results into a Grafana log-lines
// data frame with proper metadata for the Logs panel visualization.
func FormatLogResults(results *nrdb.NRDBResultContainer, query backend.DataQuery, executedQuery string) *backend.DataResponse {
	resp := &backend.DataResponse{}

	if len(results.Results) == 0 {
		log.DefaultLogger.Debug("Log formatter: no results")
		return resp
	}

	n := len(results.Results)

	timestamps := make([]time.Time, n)
	bodies := make([]string, n)
	severities := make([]string, n)
	ids := make([]string, n)
	labelsList := make([]json.RawMessage, n)

	for i, result := range results.Results {
		timestamps[i] = extractLogTimestamp(result)
		bodies[i] = extractLogBody(result)
		severities[i] = extractLogSeverity(result)
		ids[i] = generateLogID(i, result)
		labelsList[i] = buildLogLabels(result)
	}

	frame := data.NewFrame("log-lines",
		data.NewField("timestamp", nil, timestamps),
		data.NewField("body", nil, bodies),
		data.NewField("severity", nil, severities),
		data.NewField("id", nil, ids),
		data.NewField("labels", nil, labelsList),
	)

	frame.Meta = &data.FrameMeta{
		Type:                   data.FrameTypeLogLines,
		PreferredVisualization: data.VisTypeLogs,
		ExecutedQueryString:    executedQuery,
		Custom: map[string]interface{}{
			"limit": n,
		},
	}

	log.DefaultLogger.Debug("Log formatter: built frame", "rows", n)
	resp.Frames = append(resp.Frames, frame)
	return resp
}

// extractLogTimestamp converts the NR timestamp field (epoch milliseconds) to time.Time.
func extractLogTimestamp(result map[string]interface{}) time.Time {
	if ts, ok := result["timestamp"].(float64); ok && ts > 0 {
		return time.Unix(0, int64(ts)*int64(time.Millisecond))
	}
	return time.Now()
}

// extractLogBody returns the log message body, checking multiple NR field names.
func extractLogBody(result map[string]interface{}) string {
	for _, key := range []string{"message", "log_message", "messageBody", "msg"} {
		if val, ok := result[key]; ok {
			if s, ok := val.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// extractLogSeverity returns a normalized severity string for Grafana color-coding.
func extractLogSeverity(result map[string]interface{}) string {
	raw := ""
	for _, key := range []string{"level", "severity", "log_severity", "log.level"} {
		if val, ok := result[key]; ok {
			if s, ok := val.(string); ok && s != "" {
				raw = s
				break
			}
		}
	}
	if raw == "" {
		return ""
	}

	lower := strings.ToLower(strings.TrimSpace(raw))

	// Check alias map first
	if canonical, ok := severityAliases[lower]; ok {
		return canonical
	}

	// Already a canonical Grafana level?
	switch lower {
	case "critical", "error", "warning", "info", "debug", "trace", "unknown":
		return lower
	}

	return lower
}

// generateLogID creates a unique ID for a log line.
func generateLogID(index int, result map[string]interface{}) string {
	// Prefer NR-provided IDs
	for _, key := range []string{"messageId", "nr.guid"} {
		if val, ok := result[key]; ok {
			if s, ok := val.(string); ok && s != "" {
				return s
			}
		}
	}

	// Fallback: index + timestamp
	ts := extractLogTimestamp(result)
	return fmt.Sprintf("%d-%d", index, ts.UnixNano())
}

// buildLogLabels creates a JSON object of all attributes not consumed by other fields.
// Nil values are skipped, and complex values are converted to strings to ensure
// Grafana can always display them in the log detail view.
func buildLogLabels(result map[string]interface{}) json.RawMessage {
	labels := make(map[string]interface{})
	for k, v := range result {
		if logConsumedFields[k] || v == nil {
			continue
		}
		// Flatten complex types to strings for reliable display
		switch val := v.(type) {
		case string, float64, bool:
			labels[k] = val
		case int, int64:
			labels[k] = val
		default:
			// Arrays, maps, etc. — marshal to string
			if b, err := json.Marshal(val); err == nil {
				labels[k] = string(b)
			} else {
				labels[k] = fmt.Sprintf("%v", val)
			}
		}
	}

	if len(labels) == 0 {
		return json.RawMessage("{}")
	}

	b, err := json.Marshal(labels)
	if err != nil {
		log.DefaultLogger.Error("Failed to marshal log labels", "error", err)
		return json.RawMessage("{}")
	}
	return b
}
