package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryer_Do(t *testing.T) {
	t.Run("success on first attempt", func(t *testing.T) {
		r := New(Config{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
		})

		attempts := 0
		err := r.Do(func() error {
			attempts++
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
	})

	t.Run("success after retries", func(t *testing.T) {
		r := New(Config{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
		})

		attempts := 0
		err := r.Do(func() error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary error")
			}
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("max attempts reached", func(t *testing.T) {
		r := New(Config{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
		})

		attempts := 0
		err := r.Do(func() error {
			attempts++
			return errors.New("permanent error")
		})

		if !errors.Is(err, ErrMaxAttemptsReached) {
			t.Errorf("expected max attempts error, got %v", err)
		}

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})
}

func TestRetryer_DoWithContext(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		r := New(Config{
			MaxAttempts:  5,
			InitialDelay: 100 * time.Millisecond,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		attempts := 0
		err := r.DoWithContext(ctx, func(ctx context.Context) error {
			attempts++
			return errors.New("error")
		})

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected context deadline exceeded, got %v", err)
		}

		if attempts > 2 {
			t.Errorf("expected few attempts due to timeout, got %d", attempts)
		}
	})

	t.Run("success with context", func(t *testing.T) {
		r := New(Config{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
		})

		ctx := context.Background()
		attempts := 0

		err := r.DoWithContext(ctx, func(ctx context.Context) error {
			attempts++
			if attempts < 2 {
				return errors.New("temporary error")
			}
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", attempts)
		}
	})
}

func TestRetryer_DoWithData(t *testing.T) {
	t.Run("success returns data", func(t *testing.T) {
		r := New(Config{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
		})

		data, err := r.DoWithData(func() (any, error) {
			return "success", nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if data != "success" {
			t.Errorf("expected 'success', got %v", data)
		}
	})

	t.Run("failure after retries", func(t *testing.T) {
		r := New(Config{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
		})

		attempts := 0
		data, err := r.DoWithData(func() (any, error) {
			attempts++
			return nil, errors.New("error")
		})

		if !errors.Is(err, ErrMaxAttemptsReached) {
			t.Errorf("expected max attempts error, got %v", err)
		}

		if data != nil {
			t.Errorf("expected nil data, got %v", data)
		}

		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})
}

func TestRetryStrategies(t *testing.T) {
	testCases := []struct {
		name     string
		strategy Strategy
	}{
		{"fixed", StrategyFixed},
		{"linear", StrategyLinear},
		{"exponential", StrategyExponential},
		{"exponential_jitter", StrategyExponentialJitter},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := New(Config{
				MaxAttempts:  3,
				InitialDelay: 10 * time.Millisecond,
				MaxDelay:     100 * time.Millisecond,
				Strategy:     tc.strategy,
				Multiplier:   2.0,
				Jitter:       0.1,
			})

			attempts := 0
			delays := make([]time.Duration, 0)

			r.config.OnRetry = func(attempt int, delay time.Duration, err error) {
				delays = append(delays, delay)
			}

			r.Do(func() error {
				attempts++
				if attempts < 3 {
					return errors.New("error")
				}
				return nil
			})

			if len(delays) != 2 {
				t.Errorf("expected 2 delays, got %d", len(delays))
			}
		})
	}
}

func TestRetryIf(t *testing.T) {
	t.Run("custom retry condition", func(t *testing.T) {
		specificErr := errors.New("retryable error")
		otherErr := errors.New("non-retryable error")

		r := New(Config{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			RetryIf: func(err error) bool {
				return errors.Is(err, specificErr)
			},
		})

		// Should retry on specific error
		attempts := 0
		err := r.Do(func() error {
			attempts++
			if attempts < 2 {
				return specificErr
			}
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		if attempts != 2 {
			t.Errorf("expected 2 attempts, got %d", attempts)
		}

		// Should not retry on other error
		attempts = 0
		err = r.Do(func() error {
			attempts++
			return otherErr
		})

		if err != otherErr {
			t.Errorf("expected other error, got %v", err)
		}

		if attempts != 1 {
			t.Errorf("expected 1 attempt, got %d", attempts)
		}
	})
}

func TestOnRetry(t *testing.T) {
	t.Run("callback is called", func(t *testing.T) {
		callbackCalled := 0

		r := New(Config{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
			OnRetry: func(attempt int, delay time.Duration, err error) {
				callbackCalled++
			},
		})

		attempts := 0
		r.Do(func() error {
			attempts++
			if attempts < 3 {
				return errors.New("error")
			}
			return nil
		})

		if callbackCalled != 2 {
			t.Errorf("expected callback called 2 times, got %d", callbackCalled)
		}
	})
}

func TestConvenienceFunctions(t *testing.T) {
	t.Run("Do with defaults", func(t *testing.T) {
		attempts := 0
		err := Do(func() error {
			attempts++
			if attempts < 2 {
				return errors.New("error")
			}
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("DoWithContext with defaults", func(t *testing.T) {
		ctx := context.Background()
		attempts := 0

		err := DoWithContext(ctx, func(ctx context.Context) error {
			attempts++
			if attempts < 2 {
				return errors.New("error")
			}
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("DoWithConfig", func(t *testing.T) {
		config := Config{
			MaxAttempts:  5,
			InitialDelay: 10 * time.Millisecond,
		}

		attempts := 0
		err := DoWithConfig(config, func() error {
			attempts++
			if attempts < 2 {
				return errors.New("error")
			}
			return nil
		})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestRetryConditions(t *testing.T) {
	t.Run("RetryOnAnyError", func(t *testing.T) {
		if !RetryOnAnyError(errors.New("any error")) {
			t.Error("expected true for any error")
		}

		if RetryOnAnyError(nil) {
			t.Error("expected false for nil error")
		}
	})

	t.Run("RetryOnSpecificErrors", func(t *testing.T) {
		err1 := errors.New("error 1")
		err2 := errors.New("error 2")
		err3 := errors.New("error 3")

		retryIf := RetryOnSpecificErrors(err1, err2)

		if !retryIf(err1) {
			t.Error("expected true for err1")
		}

		if retryIf(err3) {
			t.Error("expected false for err3")
		}

		if retryIf(nil) {
			t.Error("expected false for nil")
		}
	})

	t.Run("NotRetryableErrors", func(t *testing.T) {
		err1 := errors.New("retryable")
		err2 := errors.New("not retryable")

		retryIf := NotRetryableErrors(err2)

		if !retryIf(err1) {
			t.Error("expected true for retryable error")
		}

		if retryIf(err2) {
			t.Error("expected false for excluded error")
		}

		if retryIf(nil) {
			t.Error("expected false for nil")
		}
	})
}

func TestRetryerWithMetrics(t *testing.T) {
	t.Run("successful retry metrics", func(t *testing.T) {
		rwm := NewWithMetrics(Config{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
		})

		attempts := 0
		rwm.Do(func() error {
			attempts++
			if attempts < 2 {
				return errors.New("error")
			}
			return nil
		})

		metrics := rwm.Metrics()
		if metrics.TotalAttempts != 1 {
			t.Errorf("expected 1 total attempt in metrics, got %d", metrics.TotalAttempts)
		}

		if metrics.SuccessfulRetries != 1 {
			t.Errorf("expected 1 successful retry, got %d", metrics.SuccessfulRetries)
		}
	})

	t.Run("failed retry metrics", func(t *testing.T) {
		rwm := NewWithMetrics(Config{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
		})

		rwm.Do(func() error {
			return errors.New("permanent error")
		})

		metrics := rwm.Metrics()
		if metrics.FailedRetries != 1 {
			t.Errorf("expected 1 failed retry, got %d", metrics.FailedRetries)
		}
	})

	t.Run("reset metrics", func(t *testing.T) {
		rwm := NewWithMetrics(Config{
			MaxAttempts:  3,
			InitialDelay: 10 * time.Millisecond,
		})

		rwm.Do(func() error {
			return nil
		})

		rwm.ResetMetrics()
		metrics := rwm.Metrics()

		if metrics.TotalAttempts != 0 {
			t.Errorf("expected 0 attempts after reset, got %d", metrics.TotalAttempts)
		}
	})
}

func TestDelayCalculation(t *testing.T) {
	t.Run("respects max delay", func(t *testing.T) {
		r := New(Config{
			MaxAttempts:  10,
			InitialDelay: 1 * time.Second,
			MaxDelay:     5 * time.Second,
			Strategy:     StrategyExponential,
			Multiplier:   2.0,
		})

		// Attempt 5 should exceed max delay with exponential backoff
		delay := r.calculateDelay(5)

		if delay > 5*time.Second {
			t.Errorf("delay exceeded max delay: %v", delay)
		}
	})

	t.Run("fixed strategy", func(t *testing.T) {
		r := New(Config{
			InitialDelay: 100 * time.Millisecond,
			Strategy:     StrategyFixed,
		})

		delay1 := r.calculateDelay(1)
		delay2 := r.calculateDelay(5)

		if delay1 != delay2 {
			t.Errorf("fixed strategy should have same delay, got %v and %v", delay1, delay2)
		}
	})
}
