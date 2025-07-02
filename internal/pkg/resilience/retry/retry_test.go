package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetry_Execute(t *testing.T) {
	t.Run("successful operation", func(t *testing.T) {
		r := New()
		callCount := 0

		errs := r.Execute(t.Context(), func() error {
			callCount++
			return nil
		})

		assert.Empty(t, errs, "No errors should be returned for successful operation")
		assert.Equal(t, 1, callCount, "Operation should be called exactly once")
	})

	t.Run("retry until success", func(t *testing.T) {
		r := New(WithAttempts(3))
		callCount := 0

		errs := r.Execute(t.Context(), func() error {
			callCount++
			if callCount < 2 {
				return errors.New("temporary error")
			}
			return nil
		})

		assert.Empty(t, errs, "No errors should be returned for successful operation")
		assert.Equal(t, 2, callCount, "Operation should be called exactly twice")
	})

	t.Run("retry exhausted", func(t *testing.T) {
		r := New(
			WithAttempts(3),
			WithDelay(1*time.Millisecond), // Use small delay for faster tests
			WithMaxDelay(5*time.Millisecond),
		)
		callCount := 0
		expectedErr := errors.New("persistent error")

		errs := r.Execute(t.Context(), func() error {
			callCount++
			return expectedErr
		})

		assert.NotEmpty(t, errs, "Errors should be returned when all attempts fail")
		assert.Len(t, errs, 3, "Should return all errors")
		assert.Contains(t, errs, expectedErr, "Expected error should be in the error list")
		assert.Equal(t, 3, callCount, "Operation should be called exactly 3 times")
	})

	t.Run("context cancellation", func(t *testing.T) {
		r := New(
			WithAttempts(5),
			WithDelay(100*time.Millisecond),
		)
		callCount := 0

		// Create a context that will be canceled after the first attempt
		ctx, cancel := context.WithCancel(t.Context())

		// Cancel the context after a short delay
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		errs := r.Execute(ctx, func() error {
			callCount++
			return errors.New("error that would normally trigger retry")
		})

		assert.NotEmpty(t, errs, "Errors should be returned when context is canceled")
		assert.Equal(t, 1, callCount, "Operation should be called exactly once due to context cancellation")
		// Check if any of the errors is context.Canceled
		var foundCanceledError bool
		for _, err := range errs {
			if errors.Is(err, context.Canceled) {
				foundCanceledError = true
				break
			}
		}
		assert.True(t, foundCanceledError, "Should contain context.Canceled error")
	})
}

func TestRetry_Options(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		r := New()
		retrier, ok := r.(*retrier)
		require.True(t, ok, "Expected r to be of type *retrier")

		assert.Equal(t, uint(3), retrier.cfg.attempts, "Default attempts should be 3")
		assert.Equal(t, 1*time.Second, retrier.cfg.delay, "Default delay should be 1s")
		assert.Equal(t, 5*time.Second, retrier.cfg.maxDelay, "Default maxDelay should be 5s")
	})

	t.Run("custom options", func(t *testing.T) {
		r := New(
			WithAttempts(5),
			WithDelay(2*time.Second),
			WithMaxDelay(10*time.Second),
		)
		retrier, ok := r.(*retrier)
		require.True(t, ok, "Expected r to be of type *retrier")

		assert.Equal(t, uint(5), retrier.cfg.attempts, "Attempts should be 5")
		assert.Equal(t, 2*time.Second, retrier.cfg.delay, "Delay should be 2s")
		assert.Equal(t, 10*time.Second, retrier.cfg.maxDelay, "MaxDelay should be 10s")
	})
}
