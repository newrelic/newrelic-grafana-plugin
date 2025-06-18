package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetrics_RecordQuery(t *testing.T) {
	// Reset metrics before the entire test
	ResetMetrics()

	// First test - successful query
	t.Run("successful query", func(t *testing.T) {
		RecordQuery(100*time.Millisecond, nil)

		got := GetMetrics()
		assert.Equal(t, uint64(1), got.QueryCount)
		assert.Equal(t, uint64(0), got.ErrorCount)
		assert.Equal(t, 100*time.Millisecond, got.TotalQueryTime)
		assert.Equal(t, 100*time.Millisecond, got.AverageQueryTime)
	})

	// Second test - failed query (builds on the first)
	t.Run("failed query", func(t *testing.T) {
		RecordQuery(200*time.Millisecond, assert.AnError)

		got := GetMetrics()
		assert.Equal(t, uint64(2), got.QueryCount)
		assert.Equal(t, uint64(1), got.ErrorCount)
		assert.Equal(t, 300*time.Millisecond, got.TotalQueryTime)
		assert.Equal(t, 150*time.Millisecond, got.AverageQueryTime)
	})
}

func TestMetrics_ConcurrentQueries(t *testing.T) {
	tests := []struct {
		name     string
		ops      func()
		expected int32
	}{
		{
			name: "increment only",
			ops: func() {
				IncrementConcurrentQueries()
			},
			expected: 1,
		},
		{
			name: "increment and decrement",
			ops: func() {
				IncrementConcurrentQueries()
				DecrementConcurrentQueries()
			},
			expected: 0,
		},
		{
			name: "multiple increments",
			ops: func() {
				IncrementConcurrentQueries()
				IncrementConcurrentQueries()
				IncrementConcurrentQueries()
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset metrics before each test
			ResetMetrics()

			tt.ops()

			got := GetMetrics()
			if got.ConcurrentQueries != tt.expected {
				t.Errorf("ConcurrentQueries = %v, want %v", got.ConcurrentQueries, tt.expected)
			}
		})
	}
}

func TestMetrics_Concurrency(t *testing.T) {
	// Reset metrics before test
	ResetMetrics()

	// Launch multiple goroutines to test concurrent access
	const goroutines = 100
	done := make(chan bool)

	for i := 0; i < goroutines; i++ {
		go func() {
			IncrementConcurrentQueries()
			RecordQuery(100*time.Millisecond, nil)
			DecrementConcurrentQueries()
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < goroutines; i++ {
		<-done
	}

	got := GetMetrics()
	if got.ConcurrentQueries != 0 {
		t.Errorf("ConcurrentQueries = %v, want 0", got.ConcurrentQueries)
	}
	if got.QueryCount != goroutines {
		t.Errorf("QueryCount = %v, want %v", got.QueryCount, goroutines)
	}
}
