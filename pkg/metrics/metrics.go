package metrics

import (
	"sync/atomic"
	"time"
)

// Metrics tracks various metrics for the plugin
type Metrics struct {
	QueryCount        uint64
	ErrorCount        uint64
	TotalQueryTime    time.Duration
	AverageQueryTime  time.Duration
	LastQueryTime     time.Time
	ConcurrentQueries int32
}

var (
	metrics = &Metrics{}
)

// RecordQuery records metrics for a completed query
func RecordQuery(duration time.Duration, err error) {
	atomic.AddUint64(&metrics.QueryCount, 1)
	if err != nil {
		atomic.AddUint64(&metrics.ErrorCount, 1)
	}
	metrics.TotalQueryTime += duration
	metrics.AverageQueryTime = metrics.TotalQueryTime / time.Duration(metrics.QueryCount)
	metrics.LastQueryTime = time.Now()
}

// IncrementConcurrentQueries increments the count of concurrent queries
func IncrementConcurrentQueries() {
	atomic.AddInt32(&metrics.ConcurrentQueries, 1)
}

// DecrementConcurrentQueries decrements the count of concurrent queries
func DecrementConcurrentQueries() {
	atomic.AddInt32(&metrics.ConcurrentQueries, -1)
}

// GetMetrics returns the current metrics
func GetMetrics() *Metrics {
	return metrics
}
