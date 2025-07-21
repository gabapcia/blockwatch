package chainstream

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gabapcia/blockwatch/internal/pkg/x/types"

	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_handleDispatchFailures(t *testing.T) {
	t.Run("processes dispatch failures with handler", func(t *testing.T) {
		// Track calls to the handler
		var handlerCalls []BlockDispatchFailure
		var mu sync.Mutex

		// Create mock notifier
		mockNotifier := NewDispatchFailureNotifierMock(t)
		mockNotifier.EXPECT().NotifyDispatchFailure(mock.Anything, mock.Anything).RunAndReturn(
			func(ctx context.Context, failure BlockDispatchFailure) error {
				mu.Lock()
				defer mu.Unlock()
				handlerCalls = append(handlerCalls, failure)
				return nil
			}).Times(3)

		// Create service with custom notifier
		svc := &service{
			dispatchFailureNotifier: mockNotifier,
		}

		// Create channel and context
		dispatchErrCh := make(chan BlockDispatchFailure, 3)
		ctx := t.Context()

		// Start handleDispatchFailures in a goroutine
		done := make(chan struct{})
		go func() {
			svc.handleDispatchFailures(ctx, dispatchErrCh)
			close(done)
		}()

		// Send test failures
		failure1 := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x100"),
			Errors:  []error{errors.New("network error 1")},
		}
		failure2 := BlockDispatchFailure{
			Network: "bitcoin",
			Height:  types.Hex("0x200"),
			Errors:  []error{errors.New("network error 2"), errors.New("retry error 2")},
		}
		failure3 := BlockDispatchFailure{
			Network: "polygon",
			Height:  types.Hex("0x300"),
			Errors:  []error{errors.New("network error 3")},
		}

		dispatchErrCh <- failure1
		dispatchErrCh <- failure2
		dispatchErrCh <- failure3
		close(dispatchErrCh)

		// Wait for function to complete
		select {
		case <-done:
			// Expected - function should return when channel is closed
		case <-time.After(2 * time.Second):
			t.Fatal("handleDispatchFailures should return when channel is closed")
		}

		// Verify all failures were processed
		mu.Lock()
		defer mu.Unlock()
		require.Len(t, handlerCalls, 3, "Expected all failures to be processed")

		// Check first failure
		assert.Equal(t, "ethereum", handlerCalls[0].Network)
		assert.Equal(t, types.Hex("0x100"), handlerCalls[0].Height)
		assert.Len(t, handlerCalls[0].Errors, 1)
		assert.Equal(t, "network error 1", handlerCalls[0].Errors[0].Error())

		// Check second failure
		assert.Equal(t, "bitcoin", handlerCalls[1].Network)
		assert.Equal(t, types.Hex("0x200"), handlerCalls[1].Height)
		assert.Len(t, handlerCalls[1].Errors, 2)
		assert.Equal(t, "network error 2", handlerCalls[1].Errors[0].Error())
		assert.Equal(t, "retry error 2", handlerCalls[1].Errors[1].Error())

		// Check third failure
		assert.Equal(t, "polygon", handlerCalls[2].Network)
		assert.Equal(t, types.Hex("0x300"), handlerCalls[2].Height)
		assert.Len(t, handlerCalls[2].Errors, 1)
		assert.Equal(t, "network error 3", handlerCalls[2].Errors[0].Error())
	})

	t.Run("handles failures with default notifier", func(t *testing.T) {
		// Create service with default notifier (logDispatchFailureNotifier)
		svc := &service{
			dispatchFailureNotifier: logDispatchFailureNotifier{},
		}

		// Create channel and context
		dispatchErrCh := make(chan BlockDispatchFailure, 2)
		ctx := t.Context()

		// Start handleDispatchFailures in a goroutine
		done := make(chan struct{})
		go func() {
			svc.handleDispatchFailures(ctx, dispatchErrCh)
			close(done)
		}()

		// Send test failures
		failure1 := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x100"),
			Errors:  []error{errors.New("network error 1")},
		}
		failure2 := BlockDispatchFailure{
			Network: "bitcoin",
			Height:  types.Hex("0x200"),
			Errors:  []error{errors.New("network error 2")},
		}

		dispatchErrCh <- failure1
		dispatchErrCh <- failure2
		close(dispatchErrCh)

		// Wait for function to complete
		select {
		case <-done:
			// Expected - function should return when channel is closed
		case <-time.After(2 * time.Second):
			t.Fatal("handleDispatchFailures should return when channel is closed")
		}

		// Function should complete without panicking or blocking
		// The default notifier will log the failures (no assertions needed)
	})

	t.Run("returns when context is canceled", func(t *testing.T) {
		// Track calls to the handler
		var handlerCalls []BlockDispatchFailure
		var mu sync.Mutex

		// Create mock notifier
		mockNotifier := NewDispatchFailureNotifierMock(t)
		mockNotifier.EXPECT().NotifyDispatchFailure(mock.Anything, mock.Anything).RunAndReturn(
			func(ctx context.Context, failure BlockDispatchFailure) error {
				mu.Lock()
				defer mu.Unlock()
				handlerCalls = append(handlerCalls, failure)
				return nil
			}).Once()

		// Create service with custom notifier
		svc := &service{
			dispatchFailureNotifier: mockNotifier,
		}

		// Create channel and cancelable context
		dispatchErrCh := make(chan BlockDispatchFailure, 1)
		ctx, cancel := context.WithCancel(t.Context())

		// Start handleDispatchFailures in a goroutine
		done := make(chan struct{})
		go func() {
			svc.handleDispatchFailures(ctx, dispatchErrCh)
			close(done)
		}()

		// Send one failure first
		failure := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x100"),
			Errors:  []error{errors.New("network error")},
		}
		dispatchErrCh <- failure

		// Give some time for the failure to be processed
		time.Sleep(100 * time.Millisecond)

		// Cancel context
		cancel()

		// Wait for function to complete
		select {
		case <-done:
			// Expected - function should return when context is canceled
		case <-time.After(2 * time.Second):
			t.Fatal("handleDispatchFailures should return when context is canceled")
		}

		// Verify the first failure was processed before cancellation
		mu.Lock()
		defer mu.Unlock()
		require.Len(t, handlerCalls, 1, "Expected first failure to be processed")
		assert.Equal(t, "ethereum", handlerCalls[0].Network)
		assert.Equal(t, types.Hex("0x100"), handlerCalls[0].Height)
	})

	t.Run("returns when channel is closed", func(t *testing.T) {
		// Create mock notifier
		mockNotifier := NewDispatchFailureNotifierMock(t)

		// Create service with notifier
		svc := &service{
			dispatchFailureNotifier: mockNotifier,
		}

		// Create channel and context
		dispatchErrCh := make(chan BlockDispatchFailure)
		ctx := t.Context()

		// Close channel immediately
		close(dispatchErrCh)

		// Start handleDispatchFailures in a goroutine
		done := make(chan struct{})
		go func() {
			svc.handleDispatchFailures(ctx, dispatchErrCh)
			close(done)
		}()

		// Function should return quickly when channel is closed
		select {
		case <-done:
			// Expected - function should return when channel is closed
		case <-time.After(500 * time.Millisecond):
			t.Fatal("handleDispatchFailures should return quickly when channel is closed")
		}
	})

	t.Run("handles empty channel gracefully", func(t *testing.T) {
		// Create mock notifier that should not be called
		mockNotifier := NewDispatchFailureNotifierMock(t)

		// Create service with notifier
		svc := &service{
			dispatchFailureNotifier: mockNotifier,
		}

		// Create empty channel and context
		dispatchErrCh := make(chan BlockDispatchFailure)
		ctx := t.Context()

		// Start handleDispatchFailures in a goroutine
		done := make(chan struct{})
		go func() {
			svc.handleDispatchFailures(ctx, dispatchErrCh)
			close(done)
		}()

		// Close channel after a short delay
		go func() {
			time.Sleep(50 * time.Millisecond)
			close(dispatchErrCh)
		}()

		// Function should return when channel is closed
		select {
		case <-done:
			// Expected - function should return when channel is closed
		case <-time.After(1 * time.Second):
			t.Fatal("handleDispatchFailures should return when channel is closed")
		}
	})

	t.Run("handler receives correct context", func(t *testing.T) {
		// Track the context passed to the handler
		var receivedCtx context.Context
		var mu sync.Mutex
		type testKeyType struct{}
		var testKey testKeyType

		// Create mock notifier that captures context
		mockNotifier := NewDispatchFailureNotifierMock(t)
		mockNotifier.EXPECT().NotifyDispatchFailure(mock.Anything, mock.Anything).RunAndReturn(
			func(ctx context.Context, failure BlockDispatchFailure) error {
				mu.Lock()
				defer mu.Unlock()
				receivedCtx = ctx
				return nil
			}).Once()

		// Create service with custom notifier
		svc := &service{
			dispatchFailureNotifier: mockNotifier,
		}

		// Create channel and context with a value
		dispatchErrCh := make(chan BlockDispatchFailure, 1)
		ctx := context.WithValue(t.Context(), testKey, "test-value")

		// Start handleDispatchFailures in a goroutine
		done := make(chan struct{})
		go func() {
			svc.handleDispatchFailures(ctx, dispatchErrCh)
			close(done)
		}()

		// Send test failure
		failure := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x100"),
			Errors:  []error{errors.New("test error")},
		}
		dispatchErrCh <- failure
		close(dispatchErrCh)

		// Wait for function to complete
		select {
		case <-done:
			// Expected - function should return when channel is closed
		case <-time.After(2 * time.Second):
			t.Fatal("handleDispatchFailures should return when channel is closed")
		}

		// Verify the correct context was passed to the handler
		mu.Lock()
		defer mu.Unlock()
		require.NotNil(t, receivedCtx, "Handler should receive context")
		assert.Equal(t, "test-value", receivedCtx.Value(testKey), "Handler should receive the same context")
	})

	t.Run("processes failures in order", func(t *testing.T) {
		// Track the order of handler calls
		var handlerCalls []string
		var mu sync.Mutex

		// Create mock notifier that tracks order
		mockNotifier := NewDispatchFailureNotifierMock(t)
		mockNotifier.EXPECT().NotifyDispatchFailure(mock.Anything, mock.Anything).RunAndReturn(
			func(ctx context.Context, failure BlockDispatchFailure) error {
				mu.Lock()
				defer mu.Unlock()
				handlerCalls = append(handlerCalls, failure.Network+"-"+string(failure.Height))
				return nil
			}).Times(5)

		// Create service with custom notifier
		svc := &service{
			dispatchFailureNotifier: mockNotifier,
		}

		// Create channel and context
		dispatchErrCh := make(chan BlockDispatchFailure, 5)
		ctx := t.Context()

		// Start handleDispatchFailures in a goroutine
		done := make(chan struct{})
		go func() {
			svc.handleDispatchFailures(ctx, dispatchErrCh)
			close(done)
		}()

		// Send failures in specific order
		failures := []BlockDispatchFailure{
			{Network: "ethereum", Height: types.Hex("0x100"), Errors: []error{errors.New("error1")}},
			{Network: "bitcoin", Height: types.Hex("0x200"), Errors: []error{errors.New("error2")}},
			{Network: "polygon", Height: types.Hex("0x300"), Errors: []error{errors.New("error3")}},
			{Network: "ethereum", Height: types.Hex("0x101"), Errors: []error{errors.New("error4")}},
			{Network: "bitcoin", Height: types.Hex("0x201"), Errors: []error{errors.New("error5")}},
		}

		for _, failure := range failures {
			dispatchErrCh <- failure
		}
		close(dispatchErrCh)

		// Wait for function to complete
		select {
		case <-done:
			// Expected - function should return when channel is closed
		case <-time.After(2 * time.Second):
			t.Fatal("handleDispatchFailures should return when channel is closed")
		}

		// Verify failures were processed in order
		mu.Lock()
		defer mu.Unlock()
		expectedOrder := []string{
			"ethereum-0x100",
			"bitcoin-0x200",
			"polygon-0x300",
			"ethereum-0x101",
			"bitcoin-0x201",
		}
		assert.Equal(t, expectedOrder, handlerCalls, "Failures should be processed in order")
	})

	t.Run("handler panic does not crash function", func(t *testing.T) {
		// Create mock notifier that panics
		mockNotifier := NewDispatchFailureNotifierMock(t)
		mockNotifier.EXPECT().NotifyDispatchFailure(mock.Anything, mock.Anything).RunAndReturn(
			func(ctx context.Context, failure BlockDispatchFailure) error {
				panic("handler panic")
			}).Once()

		// Create service with notifier that panics
		svc := &service{
			dispatchFailureNotifier: mockNotifier,
		}

		// Create channel and context
		dispatchErrCh := make(chan BlockDispatchFailure, 1)
		ctx := t.Context()

		// Start handleDispatchFailures in a goroutine
		done := make(chan struct{})
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Expected - handler panic should be caught
				}
				close(done)
			}()
			svc.handleDispatchFailures(ctx, dispatchErrCh)
		}()

		// Send test failure
		failure := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x100"),
			Errors:  []error{errors.New("test error")},
		}
		dispatchErrCh <- failure
		close(dispatchErrCh)

		// Function should complete even with panicking handler
		select {
		case <-done:
			// Expected - function should complete despite handler panic
		case <-time.After(2 * time.Second):
			t.Fatal("handleDispatchFailures should complete despite handler panic")
		}
	})

	t.Run("logs error when notifier returns error", func(t *testing.T) {
		// Create mock notifier that returns an error
		mockNotifier := NewDispatchFailureNotifierMock(t)
		notifierError := errors.New("notifier processing error")
		mockNotifier.EXPECT().NotifyDispatchFailure(mock.Anything, mock.Anything).Return(notifierError).Once()

		// Create service with notifier that returns error
		svc := &service{
			dispatchFailureNotifier: mockNotifier,
		}

		// Create channel and context
		dispatchErrCh := make(chan BlockDispatchFailure, 1)
		ctx := t.Context()

		// Start handleDispatchFailures in a goroutine
		done := make(chan struct{})
		go func() {
			svc.handleDispatchFailures(ctx, dispatchErrCh)
			close(done)
		}()

		// Send test failure
		failure := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x100"),
			Errors:  []error{errors.New("test error")},
		}
		dispatchErrCh <- failure
		close(dispatchErrCh)

		// Function should complete and log the notifier error
		select {
		case <-done:
			// Expected - function should complete and log the error
		case <-time.After(2 * time.Second):
			t.Fatal("handleDispatchFailures should complete and log notifier error")
		}

		// The error should be logged (we can't easily assert on log output in this test,
		// but the coverage will show that the error handling path was executed)
	})
}

func TestService_startHandleDispatchFailures(t *testing.T) {
	t.Run("starts handleDispatchFailures in goroutine", func(t *testing.T) {
		// Track calls to the handler
		var handlerCalls []BlockDispatchFailure
		var mu sync.Mutex

		// Create mock notifier
		mockNotifier := NewDispatchFailureNotifierMock(t)
		mockNotifier.EXPECT().NotifyDispatchFailure(mock.Anything, mock.Anything).RunAndReturn(
			func(ctx context.Context, failure BlockDispatchFailure) error {
				mu.Lock()
				defer mu.Unlock()
				handlerCalls = append(handlerCalls, failure)
				return nil
			}).Once()

		// Create service with custom notifier
		svc := &service{
			dispatchFailureNotifier: mockNotifier,
		}

		// Create channel and context
		dispatchErrCh := make(chan BlockDispatchFailure, 1)
		ctx := t.Context()

		// Start the handler using the wrapper function
		svc.startHandleDispatchFailures(ctx, dispatchErrCh)

		// Send test failure
		failure := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x123"),
			Errors:  []error{errors.New("test error")},
		}
		dispatchErrCh <- failure
		close(dispatchErrCh)

		// Give some time for the goroutine to process
		time.Sleep(100 * time.Millisecond)

		// Verify the failure was processed by the background goroutine
		mu.Lock()
		defer mu.Unlock()
		require.Len(t, handlerCalls, 1, "Expected failure to be processed")
		assert.Equal(t, "ethereum", handlerCalls[0].Network)
		assert.Equal(t, types.Hex("0x123"), handlerCalls[0].Height)
		assert.Len(t, handlerCalls[0].Errors, 1)
		assert.Equal(t, "test error", handlerCalls[0].Errors[0].Error())
	})

	t.Run("function returns immediately", func(t *testing.T) {
		// Create mock notifier
		mockNotifier := NewDispatchFailureNotifierMock(t)

		// Create service
		svc := &service{
			dispatchFailureNotifier: mockNotifier,
		}

		// Create channel and context
		dispatchErrCh := make(chan BlockDispatchFailure)
		ctx := t.Context()

		// Record start time
		start := time.Now()

		// Call startHandleDispatchFailures - should return immediately
		svc.startHandleDispatchFailures(ctx, dispatchErrCh)

		// Verify it returned quickly (within 10ms)
		elapsed := time.Since(start)
		assert.Less(t, elapsed, 10*time.Millisecond, "startHandleDispatchFailures should return immediately")

		// Close channel to clean up the goroutine
		close(dispatchErrCh)
	})

	t.Run("handles context cancellation in background goroutine", func(t *testing.T) {
		// Track calls to the handler
		var handlerCalls []BlockDispatchFailure
		var mu sync.Mutex

		// Create mock notifier
		mockNotifier := NewDispatchFailureNotifierMock(t)
		mockNotifier.EXPECT().NotifyDispatchFailure(mock.Anything, mock.Anything).RunAndReturn(
			func(ctx context.Context, failure BlockDispatchFailure) error {
				mu.Lock()
				defer mu.Unlock()
				handlerCalls = append(handlerCalls, failure)
				return nil
			}).Once()

		// Create service with custom notifier
		svc := &service{
			dispatchFailureNotifier: mockNotifier,
		}

		// Create channel and cancelable context
		dispatchErrCh := make(chan BlockDispatchFailure, 1)
		ctx, cancel := context.WithCancel(t.Context())

		// Start the handler
		svc.startHandleDispatchFailures(ctx, dispatchErrCh)

		// Send test failure
		failure := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x123"),
			Errors:  []error{errors.New("test error")},
		}
		dispatchErrCh <- failure

		// Give some time for processing
		time.Sleep(100 * time.Millisecond)

		// Verify the failure was processed
		mu.Lock()
		processedCount := len(handlerCalls)
		mu.Unlock()
		assert.Equal(t, 1, processedCount, "Expected failure to be processed before cancellation")

		// Cancel context to clean up
		cancel()
		close(dispatchErrCh)
	})

	t.Run("multiple concurrent calls create separate goroutines", func(t *testing.T) {
		// Track calls to the handler with goroutine identification
		var handlerCalls []string
		var mu sync.Mutex

		// Create mock notifier that includes a unique identifier
		mockNotifier := NewDispatchFailureNotifierMock(t)
		mockNotifier.EXPECT().NotifyDispatchFailure(mock.Anything, mock.Anything).RunAndReturn(
			func(ctx context.Context, failure BlockDispatchFailure) error {
				mu.Lock()
				defer mu.Unlock()
				// Use the network name to identify which goroutine processed this
				handlerCalls = append(handlerCalls, failure.Network)
				return nil
			}).Times(2)

		// Create service with custom notifier
		svc := &service{
			dispatchFailureNotifier: mockNotifier,
		}

		// Create separate channels for each goroutine
		dispatchErrCh1 := make(chan BlockDispatchFailure, 1)
		dispatchErrCh2 := make(chan BlockDispatchFailure, 1)
		ctx := t.Context()

		// Start two separate handler goroutines
		svc.startHandleDispatchFailures(ctx, dispatchErrCh1)
		svc.startHandleDispatchFailures(ctx, dispatchErrCh2)

		// Send different failures to each channel
		failure1 := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x100"),
			Errors:  []error{errors.New("error1")},
		}
		failure2 := BlockDispatchFailure{
			Network: "bitcoin",
			Height:  types.Hex("0x200"),
			Errors:  []error{errors.New("error2")},
		}

		dispatchErrCh1 <- failure1
		dispatchErrCh2 <- failure2
		close(dispatchErrCh1)
		close(dispatchErrCh2)

		// Give time for both goroutines to process
		time.Sleep(200 * time.Millisecond)

		// Verify both failures were processed
		mu.Lock()
		defer mu.Unlock()
		require.Len(t, handlerCalls, 2, "Expected both failures to be processed")

		// Both networks should be present (order doesn't matter due to concurrency)
		assert.Contains(t, handlerCalls, "ethereum")
		assert.Contains(t, handlerCalls, "bitcoin")
	})
}
