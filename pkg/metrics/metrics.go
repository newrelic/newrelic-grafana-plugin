// Package metrics provides thread-safe performance monitoring and metrics collection
// for the New Relic Grafana plugin. It tracks query execution statistics, error rates,
// and concurrent operations using atomic operations to ensure data consistency.
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

// Internal storage with atomic access
type atomicMetrics struct {
	QueryCount        uint64
	ErrorCount        uint64
	TotalQueryTime    int64 // nanoseconds
	AverageQueryTime  int64 // nanoseconds
	LastQueryTime     int64 // Unix nanoseconds
	ConcurrentQueries int32
}

var (
	metrics = &atomicMetrics{}
)

// RecordQuery records metrics for a completed query
func RecordQuery(duration time.Duration, err error) {
	queryCount := atomic.AddUint64(&metrics.QueryCount, 1)
	if err != nil {
		atomic.AddUint64(&metrics.ErrorCount, 1)
	}

	// Update total query time atomically
	durationNanos := int64(duration)
	atomic.AddInt64(&metrics.TotalQueryTime, durationNanos)

	// Calculate and update average query time atomically
	newTotalTime := atomic.LoadInt64(&metrics.TotalQueryTime)
	newAverage := newTotalTime / int64(queryCount)
	atomic.StoreInt64(&metrics.AverageQueryTime, newAverage)

	// Update last query time atomically
	atomic.StoreInt64(&metrics.LastQueryTime, time.Now().UnixNano())
}

// IncrementConcurrentQueries increments the count of concurrent queries
func IncrementConcurrentQueries() {
	atomic.AddInt32(&metrics.ConcurrentQueries, 1)
}

// DecrementConcurrentQueries decrements the count of concurrent queries
func DecrementConcurrentQueries() {
	atomic.AddInt32(&metrics.ConcurrentQueries, -1)
}

// GetMetrics returns a copy of the current metrics with atomic access
func GetMetrics() *Metrics {
	return &Metrics{
		QueryCount:        atomic.LoadUint64(&metrics.QueryCount),
		ErrorCount:        atomic.LoadUint64(&metrics.ErrorCount),
		TotalQueryTime:    time.Duration(atomic.LoadInt64(&metrics.TotalQueryTime)),
		AverageQueryTime:  time.Duration(atomic.LoadInt64(&metrics.AverageQueryTime)),
		LastQueryTime:     time.Unix(0, atomic.LoadInt64(&metrics.LastQueryTime)),
		ConcurrentQueries: atomic.LoadInt32(&metrics.ConcurrentQueries),
	}
}

// ResetMetrics resets all metrics to zero (for testing)
func ResetMetrics() {
	atomic.StoreUint64(&metrics.QueryCount, 0)
	atomic.StoreUint64(&metrics.ErrorCount, 0)
	atomic.StoreInt64(&metrics.TotalQueryTime, 0)
	atomic.StoreInt64(&metrics.AverageQueryTime, 0)
	atomic.StoreInt64(&metrics.LastQueryTime, 0)
	atomic.StoreInt32(&metrics.ConcurrentQueries, 0)
}
