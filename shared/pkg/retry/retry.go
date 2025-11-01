package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

var (
	// ErrMaxAttemptsReached is returned when maximum retry attempts are reached
	ErrMaxAttemptsReached = errors.New("maximum retry attempts reached")
)

// Strategy defines the retry strategy
type Strategy string

const (
	// StrategyFixed uses fixed delay between retries
	StrategyFixed Strategy = "fixed"

	// StrategyLinear increases delay linearly
	StrategyLinear Strategy = "linear"

	// StrategyExponential uses exponential backoff
	StrategyExponential Strategy = "exponential"

	// StrategyExponentialJitter uses exponential backoff with jitter
	StrategyExponentialJitter Strategy = "exponential_jitter"
)

// Config holds retry configuration
type Config struct {
	// MaxAttempts is the maximum number of retry attempts (0 means infinite)
	MaxAttempts int

	// InitialDelay is the initial delay before first retry
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration

	// Strategy determines the backoff strategy
	Strategy Strategy

	// Multiplier is used for exponential backoff (default 2.0)
	Multiplier float64

	// Jitter adds randomness to delay (0.0 to 1.0)
	Jitter float64

	// RetryIf is called to determine if an error should trigger a retry
	RetryIf func(error) bool

	// OnRetry is called before each retry attempt
	OnRetry func(attempt int, delay time.Duration, err error)
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Strategy:     StrategyExponentialJitter,
		Multiplier:   2.0,
		Jitter:       0.1,
		RetryIf: func(err error) bool {
			return err != nil
		},
	}
}

// Retryer handles retry logic
type Retryer struct {
	config Config
}

// New creates a new Retryer with the given configuration
func New(config Config) *Retryer {
	r := &Retryer{
		config: config,
	}

	// Set defaults
	if r.config.InitialDelay == 0 {
		r.config.InitialDelay = 100 * time.Millisecond
	}

	if r.config.MaxDelay == 0 {
		r.config.MaxDelay = 30 * time.Second
	}

	if r.config.Strategy == "" {
		r.config.Strategy = StrategyExponential
	}

	if r.config.Multiplier == 0 {
		r.config.Multiplier = 2.0
	}

	if r.config.RetryIf == nil {
		r.config.RetryIf = func(err error) bool {
			return err != nil
		}
	}

	return r
}

// Do executes the function with retries
func (r *Retryer) Do(fn func() error) error {
	return r.DoWithContext(context.Background(), func(ctx context.Context) error {
		return fn()
	})
}

// DoWithContext executes the function with retries and context support
func (r *Retryer) DoWithContext(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error
	attempt := 0

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the function
		err := fn(ctx)

		// Success case
		if !r.config.RetryIf(err) {
			return err
		}

		lastErr = err
		attempt++

		// Check if we've reached max attempts
		if r.config.MaxAttempts > 0 && attempt >= r.config.MaxAttempts {
			return fmt.Errorf("%w: %v", ErrMaxAttemptsReached, lastErr)
		}

		// Calculate delay
		delay := r.calculateDelay(attempt)

		// Call OnRetry callback
		if r.config.OnRetry != nil {
			r.config.OnRetry(attempt, delay, err)
		}

		// Wait before retry
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// DoWithData executes the function with retries and returns data on success
func (r *Retryer) DoWithData(fn func() (any, error)) (any, error) {
	return r.DoWithDataContext(context.Background(), func(ctx context.Context) (any, error) {
		return fn()
	})
}

// DoWithDataContext executes the function with retries, context, and returns data
func (r *Retryer) DoWithDataContext(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	var lastErr error
	var data any
	attempt := 0

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Execute the function
		result, err := fn(ctx)

		// Success case
		if !r.config.RetryIf(err) {
			return result, err
		}

		lastErr = err
		data = result
		attempt++

		// Check if we've reached max attempts
		if r.config.MaxAttempts > 0 && attempt >= r.config.MaxAttempts {
			return data, fmt.Errorf("%w: %v", ErrMaxAttemptsReached, lastErr)
		}

		// Calculate delay
		delay := r.calculateDelay(attempt)

		// Call OnRetry callback
		if r.config.OnRetry != nil {
			r.config.OnRetry(attempt, delay, err)
		}

		// Wait before retry
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (r *Retryer) calculateDelay(attempt int) time.Duration {
	var delay time.Duration

	switch r.config.Strategy {
	case StrategyFixed:
		delay = r.config.InitialDelay

	case StrategyLinear:
		delay = r.config.InitialDelay * time.Duration(attempt)

	case StrategyExponential:
		delay = r.config.InitialDelay * time.Duration(math.Pow(r.config.Multiplier, float64(attempt-1)))

	case StrategyExponentialJitter:
		baseDelay := r.config.InitialDelay * time.Duration(math.Pow(r.config.Multiplier, float64(attempt-1)))
		jitter := time.Duration(float64(baseDelay) * r.config.Jitter * (rand.Float64()*2 - 1))
		delay = baseDelay + jitter

	default:
		delay = r.config.InitialDelay
	}

	// Cap at max delay
	if delay > r.config.MaxDelay {
		delay = r.config.MaxDelay
	}

	// Ensure minimum delay
	if delay < 0 {
		delay = r.config.InitialDelay
	}

	return delay
}

// Convenience functions

// Do executes fn with default retry configuration
func Do(fn func() error) error {
	return New(DefaultConfig()).Do(fn)
}

// DoWithContext executes fn with default retry configuration and context
func DoWithContext(ctx context.Context, fn func(context.Context) error) error {
	return New(DefaultConfig()).DoWithContext(ctx, fn)
}

// DoWithConfig executes fn with custom configuration
func DoWithConfig(config Config, fn func() error) error {
	return New(config).Do(fn)
}

// DoWithConfigContext executes fn with custom configuration and context
func DoWithConfigContext(ctx context.Context, config Config, fn func(context.Context) error) error {
	return New(config).DoWithContext(ctx, fn)
}

// Predefined retry conditions

// RetryOnAnyError retries on any error
func RetryOnAnyError(err error) bool {
	return err != nil
}

// RetryOnSpecificErrors retries only on specific errors
func RetryOnSpecificErrors(targets ...error) func(error) bool {
	return func(err error) bool {
		if err == nil {
			return false
		}
		for _, target := range targets {
			if errors.Is(err, target) {
				return true
			}
		}
		return false
	}
}

// RetryOnTemporary retries on temporary errors (implements Temporary interface)
func RetryOnTemporary(err error) bool {
	if err == nil {
		return false
	}
	type temporary interface {
		Temporary() bool
	}
	if te, ok := err.(temporary); ok {
		return te.Temporary()
	}
	return false
}

// NotRetryableErrors creates a retry condition that excludes specific errors
func NotRetryableErrors(excluded ...error) func(error) bool {
	return func(err error) bool {
		if err == nil {
			return false
		}
		for _, e := range excluded {
			if errors.Is(err, e) {
				return false
			}
		}
		return true
	}
}

// RetryMetrics holds statistics about retry operations
type RetryMetrics struct {
	TotalAttempts    int
	SuccessfulRetries int
	FailedRetries    int
	TotalDelay       time.Duration
}

// RetryerWithMetrics wraps Retryer with metrics collection
type RetryerWithMetrics struct {
	retryer *Retryer
	metrics RetryMetrics
}

// NewWithMetrics creates a new Retryer with metrics collection
func NewWithMetrics(config Config) *RetryerWithMetrics {
	originalOnRetry := config.OnRetry

	rwm := &RetryerWithMetrics{}

	config.OnRetry = func(attempt int, delay time.Duration, err error) {
		rwm.metrics.TotalAttempts++
		rwm.metrics.TotalDelay += delay

		if originalOnRetry != nil {
			originalOnRetry(attempt, delay, err)
		}
	}

	rwm.retryer = New(config)
	return rwm
}

// Do executes the function with retries and metrics collection
func (rwm *RetryerWithMetrics) Do(fn func() error) error {
	err := rwm.retryer.Do(fn)
	if err == nil || !errors.Is(err, ErrMaxAttemptsReached) {
		rwm.metrics.SuccessfulRetries++
	} else {
		rwm.metrics.FailedRetries++
	}
	return err
}

// Metrics returns the collected metrics
func (rwm *RetryerWithMetrics) Metrics() RetryMetrics {
	return rwm.metrics
}

// ResetMetrics resets the metrics counters
func (rwm *RetryerWithMetrics) ResetMetrics() {
	rwm.metrics = RetryMetrics{}
}
