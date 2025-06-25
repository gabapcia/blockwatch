// Package retry provides a configurable retry mechanism for operations that may fail temporarily.
// It wraps the retry-go package from Avast and exposes a simple interface with functional
// options for customizing retry behavior.
//
// The package implements an exponential backoff strategy by default, which is suitable for
// most retry scenarios. It allows customization of retry attempts, delays, and callbacks.
//
// Basic usage:
//
//	r := retry.New()
//	err := r.Execute(context.Background(), func() error {
//	    // Your operation that might fail temporarily
//	    return someOperation()
//	})
//
// With custom options:
//
//	r := retry.New(
//	    retry.WithAttempts(5),
//	    retry.WithDelay(2*time.Second),
//	    retry.WithMaxDelay(10*time.Second),
//	    retry.WithLastErrorOnly(false),
//	)
package retry

import (
	"context"
	"time"

	retry "github.com/avast/retry-go/v4"
)

// Retry defines the interface for retry operations.
// Implementations of this interface provide a mechanism to execute operations
// with automatic retry logic in case of failures.
type Retry interface {
	// Execute runs the given function with configured retry logic.
	// It will retry the operation according to the configured parameters
	// if it returns an error.
	//
	// The context allows for cancellation and timeout control. If the context
	// is canceled or times out, the operation will stop retrying and return
	// the context error.
	//
	// The operation function should be idempotent (safe to call multiple times)
	// and should return nil on success or an error on failure.
	//
	// Execute returns nil if the operation succeeds within the configured
	// number of attempts, or an error if all attempts fail or the context is done.
	Execute(ctx context.Context, operation func() error) error
}

// config holds internal settings for the retry mechanism.
type config struct {
	attempts    uint          // maximum number of retry attempts
	delay       time.Duration // base delay between retry attempts
	maxDelay    time.Duration // maximum delay between retry attempts
	lastErrOnly bool          // whether to return only the last error
}

// Option defines a functional option for configuring the retry mechanism.
// Options are applied in the order they are provided to New().
type Option func(*config)

// retrier implements the Retry interface using the retry-go package.
type retrier struct {
	cfg config
}

// Compile-time assertion that retrier implements Retry interface
var _ Retry = (*retrier)(nil)

// New creates and returns a Retry implementation configured with
// the provided options. If no options are given, default values are used.
//
// Default configuration:
//   - attempts:    3 (1 initial attempt + 2 retries)
//   - delay:       1 second (base delay, will increase with exponential backoff)
//   - maxDelay:    5 seconds (maximum delay between retries)
//   - lastErrOnly: true (only the last error is returned)
//   - delayType:   Exponential backoff (not configurable)
//
// Example:
//
//	// Create a retry mechanism with default settings
//	r := retry.New()
//
//	// Create a retry mechanism with custom settings
//	r := retry.New(
//	    retry.WithAttempts(5),
//	    retry.WithDelay(500*time.Millisecond),
//	)
func New(opts ...Option) Retry {
	cfg := config{
		attempts:    3,
		delay:       1 * time.Second,
		maxDelay:    5 * time.Second,
		lastErrOnly: true,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &retrier{
		cfg: cfg,
	}
}

// Execute implements the Retry interface.
// It runs the given operation with retry logic according to the configured parameters.
//
// The operation is first attempted immediately. If it fails, it will be retried
// with exponential backoff delays between attempts, up to the configured maximum
// number of attempts.
//
// Example:
//
//	r := retry.New()
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	err := r.Execute(ctx, func() error {
//	    resp, err := http.Get("https://example.com")
//	    if err != nil {
//	        return err
//	    }
//	    defer resp.Body.Close()
//	    // Process response...
//	    return nil
//	})
func (r *retrier) Execute(ctx context.Context, operation func() error) error {
	options := []retry.Option{
		retry.Attempts(r.cfg.attempts),
		retry.Delay(r.cfg.delay),
		retry.MaxDelay(r.cfg.maxDelay),
		retry.DelayType(retry.BackOffDelay), // Use exponential backoff
		retry.LastErrorOnly(r.cfg.lastErrOnly),
		retry.Context(ctx), // Use the provided context for cancellation
	}

	return retry.Do(operation, options...)
}

// WithAttempts sets the maximum number of attempts (including the initial attempt).
// Default: 3 (1 initial attempt + 2 retries).
//
// Example:
//
//	// Configure for 5 total attempts (1 initial + 4 retries)
//	retry.New(retry.WithAttempts(5))
func WithAttempts(n uint) Option {
	return func(c *config) {
		c.attempts = n
	}
}

// WithDelay sets the base delay between retry attempts.
// This is the initial delay value used for the first retry.
// With exponential backoff, subsequent delays will increase.
// Default: 1 second.
//
// Example:
//
//	// Set initial delay to 500ms
//	retry.New(retry.WithDelay(500 * time.Millisecond))
func WithDelay(d time.Duration) Option {
	return func(c *config) {
		c.delay = d
	}
}

// WithMaxDelay sets the maximum delay between retry attempts.
// This caps the exponential growth of the delay to prevent
// excessively long waits between retries.
// Default: 5 seconds.
//
// Example:
//
//	// Set maximum delay to 10 seconds
//	retry.New(retry.WithMaxDelay(10 * time.Second))
func WithMaxDelay(d time.Duration) Option {
	return func(c *config) {
		c.maxDelay = d
	}
}

// WithLastErrorOnly sets whether to return only the last error.
// When true, only the error from the final attempt is returned.
// When false, all errors from all attempts are combined.
// Default: true.
//
// Example:
//
//	// Return all errors, not just the last one
//	retry.New(retry.WithLastErrorOnly(false))
func WithLastErrorOnly(b bool) Option {
	return func(c *config) {
		c.lastErrOnly = b
	}
}
