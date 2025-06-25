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

		err := r.Execute(t.Context(), func() error {
			callCount++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, callCount, "Operation should be called exactly once")
	})

	t.Run("retry until success", func(t *testing.T) {
		r := New(WithAttempts(3))
		callCount := 0

		err := r.Execute(t.Context(), func() error {
			callCount++
			if callCount < 2 {
				return errors.New("temporary error")
			}
			return nil
		})

		assert.NoError(t, err)
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

		err := r.Execute(t.Context(), func() error {
			callCount++
			return expectedErr
		})

		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
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

		err := r.Execute(ctx, func() error {
			callCount++
			return errors.New("error that would normally trigger retry")
		})

		assert.Error(t, err)
		assert.Equal(t, 1, callCount, "Operation should be called exactly once due to context cancellation")
		assert.ErrorIs(t, err, context.Canceled)
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
		assert.True(t, retrier.cfg.lastErrOnly, "Default lastErrOnly should be true")
	})

	t.Run("custom options", func(t *testing.T) {
		r := New(
			WithAttempts(5),
			WithDelay(2*time.Second),
			WithMaxDelay(10*time.Second),
			WithLastErrorOnly(false),
		)
		retrier, ok := r.(*retrier)
		require.True(t, ok, "Expected r to be of type *retrier")

		assert.Equal(t, uint(5), retrier.cfg.attempts, "Attempts should be 5")
		assert.Equal(t, 2*time.Second, retrier.cfg.delay, "Delay should be 2s")
		assert.Equal(t, 10*time.Second, retrier.cfg.maxDelay, "MaxDelay should be 10s")
		assert.False(t, retrier.cfg.lastErrOnly, "LastErrOnly should be false")
	})
}
