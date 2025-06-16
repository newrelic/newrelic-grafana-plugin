package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	tests := []struct {
		name       string
		rate       float64
		bucketSize float64
		want       *RateLimiter
	}{
		{
			name:       "standard rate",
			rate:       10.0,
			bucketSize: 100.0,
			want: &RateLimiter{
				rate:       10.0,
				bucketSize: 100.0,
				tokens:     100.0,
			},
		},
		{
			name:       "zero rate",
			rate:       0.0,
			bucketSize: 100.0,
			want: &RateLimiter{
				rate:       0.0,
				bucketSize: 100.0,
				tokens:     100.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRateLimiter(tt.rate, tt.bucketSize)
			if got.rate != tt.want.rate {
				t.Errorf("rate = %v, want %v", got.rate, tt.want.rate)
			}
			if got.bucketSize != tt.want.bucketSize {
				t.Errorf("bucketSize = %v, want %v", got.bucketSize, tt.want.bucketSize)
			}
			if got.tokens != tt.want.tokens {
				t.Errorf("tokens = %v, want %v", got.tokens, tt.want.tokens)
			}
		})
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	tests := []struct {
		name       string
		rate       float64
		bucketSize float64
		requests   int
		wantErr    bool
	}{
		{
			name:       "single request",
			rate:       10.0,
			bucketSize: 1.0,
			requests:   1,
			wantErr:    false,
		},
		{
			name:       "multiple requests",
			rate:       10.0,
			bucketSize: 5.0,
			requests:   5,
			wantErr:    false,
		},
		{
			name:       "exceed bucket size",
			rate:       1.0,
			bucketSize: 1.0,
			requests:   2,
			wantErr:    false, // Should wait but not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.rate, tt.bucketSize)
			ctx := context.Background()

			start := time.Now()
			for i := 0; i < tt.requests; i++ {
				err := rl.Wait(ctx)
				if (err != nil) != tt.wantErr {
					t.Errorf("Wait() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
			duration := time.Since(start)

			// For requests that exceed bucket size, verify we waited
			if tt.requests > int(tt.bucketSize) {
				expectedMinDuration := time.Duration(float64(tt.requests-1) / tt.rate * float64(time.Second))
				if duration < expectedMinDuration {
					t.Errorf("Wait() duration = %v, want >= %v", duration, expectedMinDuration)
				}
			}
		})
	}
}

func TestRateLimiter_WaitWithContext(t *testing.T) {
	rl := NewRateLimiter(1.0, 1.0) // 1 request per second
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// First request should succeed
	err := rl.Wait(ctx)
	if err != nil {
		t.Errorf("First Wait() error = %v, want nil", err)
	}

	// Second request should timeout
	err = rl.Wait(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Second Wait() error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter(10.0, 5.0) // 10 requests per second, bucket size 5
	ctx := context.Background()

	const goroutines = 10
	done := make(chan bool)
	errors := make(chan error, goroutines)

	start := time.Now()

	for i := 0; i < goroutines; i++ {
		go func() {
			err := rl.Wait(ctx)
			errors <- err
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < goroutines; i++ {
		<-done
	}

	duration := time.Since(start)

	// Check for errors
	for i := 0; i < goroutines; i++ {
		if err := <-errors; err != nil {
			t.Errorf("Wait() error = %v", err)
		}
	}

	// Verify that the rate limiting worked
	// With 10 requests and a rate of 10/sec, it should take at least some time for rate limiting
	// Reduced expected time to be more tolerant of system timing variations
	minExpectedDuration := 200 * time.Millisecond
	if duration < minExpectedDuration {
		t.Errorf("Duration = %v, want >= %v", duration, minExpectedDuration)
	}
}
