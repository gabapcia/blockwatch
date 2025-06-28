package chainwatch

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	retrytest "github.com/gabapcia/blockwatch/internal/pkg/resilience/retry/mocks"
	"github.com/gabapcia/blockwatch/internal/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestService_handleDispatchFailures(t *testing.T) {
	t.Run("processes dispatch failures with handler", func(t *testing.T) {
		// Track calls to the handler
		var handlerCalls []BlockDispatchFailure
		var mu sync.Mutex

		// Create service with custom handler
		svc := &service{
			onDispatchFailure: func(ctx context.Context, dispatchFailure BlockDispatchFailure) {
				mu.Lock()
				defer mu.Unlock()
				handlerCalls = append(handlerCalls, dispatchFailure)
			},
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

	t.Run("ignores failures when no handler is set", func(t *testing.T) {
		// Create service with no handler
		svc := &service{
			onDispatchFailure: nil,
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
		// No assertions needed since we're just verifying it doesn't crash
	})

	t.Run("returns when context is canceled", func(t *testing.T) {
		// Track calls to the handler
		var handlerCalls []BlockDispatchFailure
		var mu sync.Mutex

		// Create service with custom handler
		svc := &service{
			onDispatchFailure: func(ctx context.Context, dispatchFailure BlockDispatchFailure) {
				mu.Lock()
				defer mu.Unlock()
				handlerCalls = append(handlerCalls, dispatchFailure)
			},
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
		// Create service with handler
		svc := &service{
			onDispatchFailure: func(ctx context.Context, dispatchFailure BlockDispatchFailure) {
				// Handler implementation doesn't matter for this test
			},
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
		// Create service with handler
		svc := &service{
			onDispatchFailure: func(ctx context.Context, dispatchFailure BlockDispatchFailure) {
				t.Fatal("Handler should not be called for empty channel")
			},
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

		// Create service with custom handler that captures context
		svc := &service{
			onDispatchFailure: func(ctx context.Context, dispatchFailure BlockDispatchFailure) {
				mu.Lock()
				defer mu.Unlock()
				receivedCtx = ctx
			},
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

		// Create service with custom handler that tracks order
		svc := &service{
			onDispatchFailure: func(ctx context.Context, dispatchFailure BlockDispatchFailure) {
				mu.Lock()
				defer mu.Unlock()
				handlerCalls = append(handlerCalls, dispatchFailure.Network+"-"+string(dispatchFailure.Height))
			},
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
		// Create service with handler that panics
		svc := &service{
			onDispatchFailure: func(ctx context.Context, dispatchFailure BlockDispatchFailure) {
				panic("handler panic")
			},
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
}

func TestService_startHandleDispatchFailures(t *testing.T) {
	t.Run("starts handleDispatchFailures in goroutine", func(t *testing.T) {
		// Track calls to the handler
		var handlerCalls []BlockDispatchFailure
		var mu sync.Mutex

		// Create service with custom handler
		svc := &service{
			onDispatchFailure: func(ctx context.Context, dispatchFailure BlockDispatchFailure) {
				mu.Lock()
				defer mu.Unlock()
				handlerCalls = append(handlerCalls, dispatchFailure)
			},
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
		// Create service
		svc := &service{
			onDispatchFailure: func(ctx context.Context, dispatchFailure BlockDispatchFailure) {
				// Handler implementation doesn't matter for this test
			},
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

		// Create service with custom handler
		svc := &service{
			onDispatchFailure: func(ctx context.Context, dispatchFailure BlockDispatchFailure) {
				mu.Lock()
				defer mu.Unlock()
				handlerCalls = append(handlerCalls, dispatchFailure)
			},
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

		// Create service with custom handler that includes a unique identifier
		svc := &service{
			onDispatchFailure: func(ctx context.Context, dispatchFailure BlockDispatchFailure) {
				mu.Lock()
				defer mu.Unlock()
				// Use the network name to identify which goroutine processed this
				handlerCalls = append(handlerCalls, dispatchFailure.Network)
			},
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

func TestService_retryFailedBlockFetches(t *testing.T) {
	t.Run("successful retry - block recovered", func(t *testing.T) {
		// Setup mocks
		retryMock := retrytest.NewRetry(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service with mocked dependencies
		svc := &service{
			networks: map[string]Blockchain{
				"ethereum": blockchainMock,
			},
			retry: retryMock,
		}

		// Mock successful retry
		retryMock.EXPECT().Execute(mock.Anything, mock.AnythingOfType("func() error")).
			Run(func(ctx context.Context, operation func() error) {
				// Execute the operation to simulate successful retry
				operation()
			}).Return(nil)

		// Mock successful block fetch
		expectedBlock := Block{
			Height: types.Hex("0x123"),
			Hash:   "block-hash-123",
			Transactions: []Transaction{
				{Hash: "tx1", From: "addr1", To: "addr2"},
			},
		}
		blockchainMock.EXPECT().FetchBlockByHeight(mock.Anything, types.Hex("0x123")).
			Return(expectedBlock, nil)

		// Create channels
		retryCh := make(chan BlockDispatchFailure, 1)
		recoveredCh := make(chan ObservedBlock, 1)
		finalErrorCh := make(chan BlockDispatchFailure, 1)

		// Use test context
		ctx := t.Context()

		// Start the retry function in a goroutine
		go svc.retryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Send the network error to retry channel
		inputError := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x123"),
			Errors:  []error{errors.New("original fetch error")},
		}
		retryCh <- inputError
		close(retryCh) // Close to signal end of input

		// Wait for recovered block
		select {
		case recovered := <-recoveredCh:
			assert.Equal(t, "ethereum", recovered.Network)
			assert.Equal(t, types.Hex("0x123"), recovered.Height)
			assert.Equal(t, "block-hash-123", recovered.Hash)
			assert.Equal(t, []Transaction{{Hash: "tx1", From: "addr1", To: "addr2"}}, recovered.Transactions)
		case <-time.After(2 * time.Second):
			t.Fatal("Expected recovered block to be sent")
		}

		// Verify no final error was sent
		select {
		case <-finalErrorCh:
			t.Fatal("Expected no final error to be sent")
		case <-time.After(100 * time.Millisecond):
			// Expected - no final error should be sent
		}
	})

	t.Run("retry fails - network not registered", func(t *testing.T) {
		// Setup mocks
		retryMock := retrytest.NewRetry(t)

		// Create service with empty networks map
		svc := &service{
			networks: map[string]Blockchain{},
			retry:    retryMock,
		}

		// Mock retry that returns ErrNetworkNotRegistered
		retryMock.EXPECT().Execute(mock.Anything, mock.AnythingOfType("func() error")).
			RunAndReturn(func(ctx context.Context, operation func() error) []error {
				// Execute the operation to simulate the actual retry logic
				err := operation()
				if err != nil {
					return []error{err}
				}
				return nil
			}).Return([]error{ErrNetworkNotRegistered})

		// Create channels
		retryCh := make(chan BlockDispatchFailure, 1)
		recoveredCh := make(chan ObservedBlock, 1)
		finalErrorCh := make(chan BlockDispatchFailure, 1)

		// Use test context
		ctx := t.Context()

		// Start the retry function
		go svc.retryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Send network error for unknown network
		inputError := BlockDispatchFailure{
			Network: "unknown-network",
			Height:  types.Hex("0x456"),
			Errors:  []error{errors.New("original fetch error")},
		}
		retryCh <- inputError
		close(retryCh)

		// Wait for final error
		select {
		case finalErr := <-finalErrorCh:
			assert.Equal(t, "unknown-network", finalErr.Network)
			assert.Equal(t, types.Hex("0x456"), finalErr.Height)
			assert.Len(t, finalErr.Errors, 2, "Expected original error plus retry error")
			assert.Contains(t, finalErr.Errors[0].Error(), "original fetch error")
			assert.Contains(t, finalErr.Errors[1].Error(), "network not registered")
		case <-time.After(2 * time.Second):
			t.Fatal("Expected final error to be sent")
		}

		// Verify no recovery was sent
		select {
		case <-recoveredCh:
			t.Fatal("Expected no recovered block to be sent")
		case <-time.After(100 * time.Millisecond):
			// Expected - no recovery should be sent
		}
	})

	t.Run("retry fails - persistent blockchain error", func(t *testing.T) {
		// Setup mocks
		retryMock := retrytest.NewRetry(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := &service{
			networks: map[string]Blockchain{
				"ethereum": blockchainMock,
			},
			retry: retryMock,
		}

		persistentErr := errors.New("persistent blockchain error")

		// Mock retry that executes the operation and returns the persistent error
		retryMock.EXPECT().Execute(mock.Anything, mock.AnythingOfType("func() error")).
			Run(func(ctx context.Context, operation func() error) {
				// Execute the operation to simulate the actual retry logic
				operation()
			}).Return([]error{persistentErr})

		// Mock blockchain client that returns error
		blockchainMock.EXPECT().FetchBlockByHeight(mock.Anything, types.Hex("0x789")).
			Return(Block{}, persistentErr)

		// Create channels
		retryCh := make(chan BlockDispatchFailure, 1)
		recoveredCh := make(chan ObservedBlock, 1)
		finalErrorCh := make(chan BlockDispatchFailure, 1)

		// Use test context
		ctx := t.Context()

		// Start the retry function
		go svc.retryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Send network error
		inputError := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x789"),
			Errors:  []error{errors.New("original fetch error")},
		}
		retryCh <- inputError
		close(retryCh)

		// Wait for final error
		select {
		case finalErr := <-finalErrorCh:
			assert.Equal(t, "ethereum", finalErr.Network)
			assert.Equal(t, types.Hex("0x789"), finalErr.Height)
			assert.Len(t, finalErr.Errors, 2, "Expected original error plus retry error")
			assert.Contains(t, finalErr.Errors[0].Error(), "original fetch error")
			assert.Contains(t, finalErr.Errors[1].Error(), "persistent blockchain error")
		case <-time.After(2 * time.Second):
			t.Fatal("Expected final error to be sent")
		}

		// Verify no recovery was sent
		select {
		case <-recoveredCh:
			t.Fatal("Expected no recovered block to be sent")
		case <-time.After(100 * time.Millisecond):
			// Expected - no recovery should be sent
		}
	})

	t.Run("context canceled during retry", func(t *testing.T) {
		// Setup mocks
		retryMock := retrytest.NewRetry(t)

		// Create service
		svc := &service{
			networks: map[string]Blockchain{
				"ethereum": NewBlockchainMock(t),
			},
			retry: retryMock,
		}

		// Mock retry that returns context canceled error
		retryMock.EXPECT().Execute(mock.Anything, mock.AnythingOfType("func() error")).
			Return([]error{context.Canceled})

		// Create channels
		retryCh := make(chan BlockDispatchFailure, 1)
		recoveredCh := make(chan ObservedBlock, 1)
		finalErrorCh := make(chan BlockDispatchFailure, 1)

		// Use test context
		ctx := t.Context()

		// Start the retry function
		go svc.retryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Send network error
		inputError := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0xabc"),
			Errors:  []error{errors.New("original fetch error")},
		}
		retryCh <- inputError
		close(retryCh)

		// Wait for final error
		select {
		case finalErr := <-finalErrorCh:
			assert.Equal(t, "ethereum", finalErr.Network)
			assert.Equal(t, types.Hex("0xabc"), finalErr.Height)
			assert.Len(t, finalErr.Errors, 2, "Expected original error plus retry error")
			assert.Contains(t, finalErr.Errors[0].Error(), "original fetch error")
			assert.Contains(t, finalErr.Errors[1].Error(), "context canceled")
		case <-time.After(2 * time.Second):
			t.Fatal("Expected final error to be sent")
		}

		// Verify no recovery was sent
		select {
		case <-recoveredCh:
			t.Fatal("Expected no recovered block to be sent")
		case <-time.After(100 * time.Millisecond):
			// Expected - no recovery should be sent
		}
	})

	t.Run("handles multiple errors correctly", func(t *testing.T) {
		// Setup mocks
		retryMock := retrytest.NewRetry(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := &service{
			networks: map[string]Blockchain{
				"ethereum": blockchainMock,
			},
			retry: retryMock,
		}

		// Setup expectations for multiple retry attempts
		// First error - successful recovery
		retryMock.EXPECT().Execute(mock.Anything, mock.AnythingOfType("func() error")).
			Run(func(ctx context.Context, operation func() error) {
				operation()
			}).Return(nil).Once()

		blockchainMock.EXPECT().FetchBlockByHeight(mock.Anything, types.Hex("0x100")).
			Return(Block{Height: types.Hex("0x100"), Hash: "hash100"}, nil).Once()

		// Second error - failed recovery
		retryMock.EXPECT().Execute(mock.Anything, mock.AnythingOfType("func() error")).
			Run(func(ctx context.Context, operation func() error) {
				operation()
			}).Return([]error{errors.New("persistent error")}).Once()

		blockchainMock.EXPECT().FetchBlockByHeight(mock.Anything, types.Hex("0x200")).
			Return(Block{}, errors.New("persistent error")).Once()

		// Create channels
		retryCh := make(chan BlockDispatchFailure, 2)
		recoveredCh := make(chan ObservedBlock, 2)
		finalErrorCh := make(chan BlockDispatchFailure, 2)

		ctx := t.Context()

		// Start the retry function
		go svc.retryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Send two network errors
		retryCh <- BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x100"),
			Errors:  []error{errors.New("error 1")},
		}
		retryCh <- BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x200"),
			Errors:  []error{errors.New("error 2")},
		}
		close(retryCh)

		// Collect results
		var recoveredBlocks []ObservedBlock
		var finalErrors []BlockDispatchFailure

		timeout := time.After(3 * time.Second)
		expectedResults := 2
		receivedResults := 0

		for receivedResults < expectedResults {
			select {
			case recovered := <-recoveredCh:
				recoveredBlocks = append(recoveredBlocks, recovered)
				receivedResults++
			case finalErr := <-finalErrorCh:
				finalErrors = append(finalErrors, finalErr)
				receivedResults++
			case <-timeout:
				t.Fatalf("Timeout waiting for results. Received %d out of %d expected results", receivedResults, expectedResults)
			}
		}

		// Verify results
		require.Len(t, recoveredBlocks, 1, "Expected one recovered block")
		require.Len(t, finalErrors, 1, "Expected one final error")

		// Check recovered block
		assert.Equal(t, "ethereum", recoveredBlocks[0].Network)
		assert.Equal(t, types.Hex("0x100"), recoveredBlocks[0].Height)
		assert.Equal(t, "hash100", recoveredBlocks[0].Hash)

		// Check final error
		assert.Equal(t, "ethereum", finalErrors[0].Network)
		assert.Equal(t, types.Hex("0x200"), finalErrors[0].Height)
		assert.Len(t, finalErrors[0].Errors, 2, "Expected original error plus retry error")
		assert.Contains(t, finalErrors[0].Errors[0].Error(), "error 2")
		assert.Contains(t, finalErrors[0].Errors[1].Error(), "persistent error")
	})

	t.Run("empty retry channel", func(t *testing.T) {
		// Setup mocks (no expectations needed as no calls should be made)
		retryMock := retrytest.NewRetry(t)

		// Create service
		svc := &service{
			networks: map[string]Blockchain{},
			retry:    retryMock,
		}

		// Create channels
		retryCh := make(chan BlockDispatchFailure)
		recoveredCh := make(chan ObservedBlock, 1)
		finalErrorCh := make(chan BlockDispatchFailure, 1)

		ctx := t.Context()

		// Close retry channel immediately to simulate empty input
		close(retryCh)

		// Start the retry function
		done := make(chan struct{})
		go func() {
			svc.retryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)
			close(done)
		}()

		// Function should return quickly when retry channel is closed
		select {
		case <-done:
			// Expected - function should return when retryCh is closed
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Function should return quickly when retry channel is closed")
		}

		// Verify no messages were sent to output channels
		select {
		case <-recoveredCh:
			t.Fatal("Expected no messages in recovered channel")
		case <-finalErrorCh:
			t.Fatal("Expected no messages in final error channel")
		default:
			// Expected - no messages should be sent
		}
	})

	t.Run("context done before sending final error", func(t *testing.T) {
		// Setup mocks
		retryMock := retrytest.NewRetry(t)

		// Create service
		svc := &service{
			networks: map[string]Blockchain{},
			retry:    retryMock,
		}

		// Mock retry that returns an error but also checks if context is done
		retryMock.EXPECT().Execute(mock.Anything, mock.AnythingOfType("func() error")).
			RunAndReturn(func(ctx context.Context, operation func() error) []error {
				// Execute the operation to simulate the actual retry logic
				err := operation()
				if err != nil {
					return []error{err}
				}
				return nil
			}).Return([]error{errors.New("retry failed")})

		// Create channels with no buffer to simulate blocking
		retryCh := make(chan BlockDispatchFailure, 1)
		recoveredCh := make(chan ObservedBlock)
		finalErrorCh := make(chan BlockDispatchFailure) // No buffer - will block

		// Create context that will be canceled
		ctx, cancel := context.WithCancel(t.Context())

		// Start the retry function
		done := make(chan struct{})
		go func() {
			svc.retryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)
			close(done)
		}()

		// Send network error
		retryCh <- BlockDispatchFailure{
			Network: "unknown",
			Height:  types.Hex("0x123"),
			Errors:  []error{errors.New("original error")},
		}

		// Give a small delay to ensure the retry operation starts
		time.Sleep(10 * time.Millisecond)

		// Cancel context to simulate context done before final error can be sent
		cancel()

		// Function should return due to context cancellation
		select {
		case <-done:
			// Expected - function should return when context is canceled
		case <-time.After(1 * time.Second):
			t.Fatal("Function should return when context is canceled")
		}

		// Verify no final error was sent (because context was canceled)
		select {
		case <-finalErrorCh:
			t.Fatal("Expected no final error to be sent when context is canceled")
		default:
			// Expected - no final error should be sent
		}
	})
}

func TestService_startRetryFailedBlockFetches(t *testing.T) {
	t.Run("starts retry function in goroutine", func(t *testing.T) {
		// Setup mocks
		retryMock := retrytest.NewRetry(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := &service{
			networks: map[string]Blockchain{
				"ethereum": blockchainMock,
			},
			retry: retryMock,
		}

		// Mock successful retry
		retryMock.EXPECT().Execute(mock.Anything, mock.AnythingOfType("func() error")).
			Run(func(ctx context.Context, operation func() error) {
				operation()
			}).Return(nil)

		// Mock successful block fetch
		expectedBlock := Block{
			Height: types.Hex("0x123"),
			Hash:   "test-hash",
		}
		blockchainMock.EXPECT().FetchBlockByHeight(mock.Anything, types.Hex("0x123")).
			Return(expectedBlock, nil)

		// Create channels
		retryCh := make(chan BlockDispatchFailure, 1)
		recoveredCh := make(chan ObservedBlock, 1)
		finalErrorCh := make(chan BlockDispatchFailure, 1)

		ctx := t.Context()

		// Start the retry function using the wrapper
		svc.startRetryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Send network error
		inputError := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x123"),
			Errors:  []error{errors.New("original error")},
		}
		retryCh <- inputError
		close(retryCh)

		// Verify that the goroutine processes the error
		select {
		case recovered := <-recoveredCh:
			assert.Equal(t, "ethereum", recovered.Network)
			assert.Equal(t, types.Hex("0x123"), recovered.Height)
			assert.Equal(t, "test-hash", recovered.Hash)
		case <-time.After(2 * time.Second):
			t.Fatal("Expected recovered block to be sent")
		}

		// Verify no final error was sent
		select {
		case <-finalErrorCh:
			t.Fatal("Expected no final error to be sent")
		case <-time.After(100 * time.Millisecond):
			// Expected - no final error should be sent
		}
	})

	t.Run("function returns immediately", func(t *testing.T) {
		// Setup mocks
		retryMock := retrytest.NewRetry(t)

		// Create service
		svc := &service{
			networks: map[string]Blockchain{},
			retry:    retryMock,
		}

		// Create channels
		retryCh := make(chan BlockDispatchFailure)
		recoveredCh := make(chan ObservedBlock)
		finalErrorCh := make(chan BlockDispatchFailure)

		ctx := t.Context()

		// Record start time
		start := time.Now()

		// Call startRetryFailedBlockFetches - should return immediately
		svc.startRetryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Verify it returned quickly (within 10ms)
		elapsed := time.Since(start)
		assert.Less(t, elapsed, 10*time.Millisecond, "startRetryFailedBlockFetches should return immediately")

		// Close channels to clean up the goroutine
		close(retryCh)
	})

	t.Run("handles context cancellation in background goroutine", func(t *testing.T) {
		// Setup mocks
		retryMock := retrytest.NewRetry(t)

		// Create service
		svc := &service{
			networks: map[string]Blockchain{},
			retry:    retryMock,
		}

		// Mock retry that returns an error
		retryMock.EXPECT().Execute(mock.Anything, mock.AnythingOfType("func() error")).
			Return([]error{errors.New("retry failed")})

		// Create channels
		retryCh := make(chan BlockDispatchFailure, 1)
		recoveredCh := make(chan ObservedBlock)
		finalErrorCh := make(chan BlockDispatchFailure, 1)

		// Create context that will be canceled
		ctx, cancel := context.WithCancel(t.Context())

		// Start the retry function
		svc.startRetryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Send network error
		retryCh <- BlockDispatchFailure{
			Network: "unknown",
			Height:  types.Hex("0x123"),
			Errors:  []error{errors.New("original error")},
		}

		// Wait for final error to be processed
		select {
		case finalErr := <-finalErrorCh:
			assert.Equal(t, "unknown", finalErr.Network)
			assert.Equal(t, types.Hex("0x123"), finalErr.Height)
			assert.Len(t, finalErr.Errors, 2, "Expected original error plus retry error")
			assert.Contains(t, finalErr.Errors[0].Error(), "original error")
			assert.Contains(t, finalErr.Errors[1].Error(), "retry failed")
		case <-time.After(1 * time.Second):
			t.Fatal("Expected final error to be sent")
		}

		// Cancel context to clean up
		cancel()
		close(retryCh)
	})
}

func TestService_dispatchSubscriptionEvents(t *testing.T) {
	t.Run("successful block events", func(t *testing.T) {
		// Create service (no mocks needed for this function)
		svc := &service{}

		// Create channels
		eventsCh := make(chan BlockchainEvent, 2)
		blocksCh := make(chan ObservedBlock, 2)
		errorsCh := make(chan BlockDispatchFailure, 2)

		ctx := t.Context()

		// Start consuming subscription
		go svc.dispatchSubscriptionEvents(ctx, "ethereum", eventsCh, blocksCh, errorsCh)

		// Send successful block events
		block1 := Block{
			Height: types.Hex("0x100"),
			Hash:   "hash100",
			Transactions: []Transaction{
				{Hash: "tx1", From: "addr1", To: "addr2"},
			},
		}
		block2 := Block{
			Height: types.Hex("0x101"),
			Hash:   "hash101",
			Transactions: []Transaction{
				{Hash: "tx2", From: "addr3", To: "addr4"},
			},
		}

		eventsCh <- BlockchainEvent{
			Height: types.Hex("0x100"),
			Block:  block1,
			Err:    nil,
		}
		eventsCh <- BlockchainEvent{
			Height: types.Hex("0x101"),
			Block:  block2,
			Err:    nil,
		}
		close(eventsCh)

		// Collect results
		var receivedBlocks []ObservedBlock
		timeout := time.After(2 * time.Second)
		expectedBlocks := 2

		for len(receivedBlocks) < expectedBlocks {
			select {
			case block := <-blocksCh:
				receivedBlocks = append(receivedBlocks, block)
			case <-timeout:
				t.Fatalf("Timeout waiting for blocks. Received %d out of %d expected blocks", len(receivedBlocks), expectedBlocks)
			}
		}

		// Verify results
		require.Len(t, receivedBlocks, 2, "Expected two blocks")

		// Check first block
		assert.Equal(t, "ethereum", receivedBlocks[0].Network)
		assert.Equal(t, types.Hex("0x100"), receivedBlocks[0].Height)
		assert.Equal(t, "hash100", receivedBlocks[0].Hash)
		assert.Equal(t, []Transaction{{Hash: "tx1", From: "addr1", To: "addr2"}}, receivedBlocks[0].Transactions)

		// Check second block
		assert.Equal(t, "ethereum", receivedBlocks[1].Network)
		assert.Equal(t, types.Hex("0x101"), receivedBlocks[1].Height)
		assert.Equal(t, "hash101", receivedBlocks[1].Hash)
		assert.Equal(t, []Transaction{{Hash: "tx2", From: "addr3", To: "addr4"}}, receivedBlocks[1].Transactions)

		// Verify no errors were sent
		select {
		case <-errorsCh:
			t.Fatal("Expected no errors to be sent")
		case <-time.After(100 * time.Millisecond):
			// Expected - no errors should be sent
		}
	})

	t.Run("error events", func(t *testing.T) {
		// Create service
		svc := &service{}

		// Create channels
		eventsCh := make(chan BlockchainEvent, 2)
		blocksCh := make(chan ObservedBlock, 2)
		errorsCh := make(chan BlockDispatchFailure, 2)

		ctx := t.Context()

		// Start consuming subscription
		go svc.dispatchSubscriptionEvents(ctx, "ethereum", eventsCh, blocksCh, errorsCh)

		// Send error events
		err1 := errors.New("fetch error 1")
		err2 := errors.New("fetch error 2")

		eventsCh <- BlockchainEvent{
			Height: types.Hex("0x200"),
			Block:  Block{}, // zero value when error is set
			Err:    err1,
		}
		eventsCh <- BlockchainEvent{
			Height: types.Hex("0x201"),
			Block:  Block{}, // zero value when error is set
			Err:    err2,
		}
		close(eventsCh)

		// Collect results
		var receivedErrors []BlockDispatchFailure
		timeout := time.After(2 * time.Second)
		expectedErrors := 2

		for len(receivedErrors) < expectedErrors {
			select {
			case err := <-errorsCh:
				receivedErrors = append(receivedErrors, err)
			case <-timeout:
				t.Fatalf("Timeout waiting for errors. Received %d out of %d expected errors", len(receivedErrors), expectedErrors)
			}
		}

		// Verify results
		require.Len(t, receivedErrors, 2, "Expected two errors")

		// Check first error
		assert.Equal(t, "ethereum", receivedErrors[0].Network)
		assert.Equal(t, types.Hex("0x200"), receivedErrors[0].Height)
		assert.Len(t, receivedErrors[0].Errors, 1, "Expected one error")
		assert.Equal(t, "fetch error 1", receivedErrors[0].Errors[0].Error())

		// Check second error
		assert.Equal(t, "ethereum", receivedErrors[1].Network)
		assert.Equal(t, types.Hex("0x201"), receivedErrors[1].Height)
		assert.Len(t, receivedErrors[1].Errors, 1, "Expected one error")
		assert.Equal(t, "fetch error 2", receivedErrors[1].Errors[0].Error())

		// Verify no blocks were sent
		select {
		case <-blocksCh:
			t.Fatal("Expected no blocks to be sent")
		case <-time.After(100 * time.Millisecond):
			// Expected - no blocks should be sent
		}
	})

	t.Run("mixed success and error events", func(t *testing.T) {
		// Create service
		svc := &service{}

		// Create channels
		eventsCh := make(chan BlockchainEvent, 3)
		blocksCh := make(chan ObservedBlock, 3)
		errorsCh := make(chan BlockDispatchFailure, 3)

		ctx := t.Context()

		// Start consuming subscription
		go svc.dispatchSubscriptionEvents(ctx, "bitcoin", eventsCh, blocksCh, errorsCh)

		// Send mixed events
		successBlock := Block{
			Height: types.Hex("0x300"),
			Hash:   "hash300",
		}

		eventsCh <- BlockchainEvent{
			Height: types.Hex("0x300"),
			Block:  successBlock,
			Err:    nil,
		}
		eventsCh <- BlockchainEvent{
			Height: types.Hex("0x301"),
			Block:  Block{},
			Err:    errors.New("network error"),
		}
		eventsCh <- BlockchainEvent{
			Height: types.Hex("0x302"),
			Block: Block{
				Height: types.Hex("0x302"),
				Hash:   "hash302",
			},
			Err: nil,
		}
		close(eventsCh)

		// Collect results
		var receivedBlocks []ObservedBlock
		var receivedErrors []BlockDispatchFailure
		timeout := time.After(2 * time.Second)
		expectedTotal := 3
		receivedTotal := 0

		for receivedTotal < expectedTotal {
			select {
			case block := <-blocksCh:
				receivedBlocks = append(receivedBlocks, block)
				receivedTotal++
			case err := <-errorsCh:
				receivedErrors = append(receivedErrors, err)
				receivedTotal++
			case <-timeout:
				t.Fatalf("Timeout waiting for events. Received %d out of %d expected events", receivedTotal, expectedTotal)
			}
		}

		// Verify results
		require.Len(t, receivedBlocks, 2, "Expected two successful blocks")
		require.Len(t, receivedErrors, 1, "Expected one error")

		// Check successful blocks
		assert.Equal(t, "bitcoin", receivedBlocks[0].Network)
		assert.Equal(t, types.Hex("0x300"), receivedBlocks[0].Height)
		assert.Equal(t, "hash300", receivedBlocks[0].Hash)

		assert.Equal(t, "bitcoin", receivedBlocks[1].Network)
		assert.Equal(t, types.Hex("0x302"), receivedBlocks[1].Height)
		assert.Equal(t, "hash302", receivedBlocks[1].Hash)

		// Check error
		assert.Equal(t, "bitcoin", receivedErrors[0].Network)
		assert.Equal(t, types.Hex("0x301"), receivedErrors[0].Height)
		assert.Len(t, receivedErrors[0].Errors, 1, "Expected one error")
		assert.Equal(t, "network error", receivedErrors[0].Errors[0].Error())
	})

	t.Run("context canceled while processing", func(t *testing.T) {
		// Create service
		svc := &service{}

		// Create channels
		eventsCh := make(chan BlockchainEvent, 2)
		blocksCh := make(chan ObservedBlock) // No buffer to simulate blocking
		errorsCh := make(chan BlockDispatchFailure, 2)

		// Create context that will be canceled
		ctx, cancel := context.WithCancel(t.Context())

		// Start consuming subscription
		done := make(chan struct{})
		go func() {
			svc.dispatchSubscriptionEvents(ctx, "ethereum", eventsCh, blocksCh, errorsCh)
			close(done)
		}()

		// Send a successful event first
		eventsCh <- BlockchainEvent{
			Height: types.Hex("0x400"),
			Block: Block{
				Height: types.Hex("0x400"),
				Hash:   "hash400",
			},
			Err: nil,
		}

		// Cancel context before the block can be sent (since blocksCh has no buffer and no reader)
		cancel()

		// Function should return due to context cancellation
		select {
		case <-done:
			// Expected - function should return when context is canceled
		case <-time.After(1 * time.Second):
			t.Fatal("Function should return when context is canceled")
		}

		// Verify no block was sent (because context was canceled)
		select {
		case <-blocksCh:
			t.Fatal("Expected no block to be sent when context is canceled")
		default:
			// Expected - no block should be sent
		}
	})

	t.Run("empty events channel", func(t *testing.T) {
		// Create service
		svc := &service{}

		// Create channels
		eventsCh := make(chan BlockchainEvent)
		blocksCh := make(chan ObservedBlock, 1)
		errorsCh := make(chan BlockDispatchFailure, 1)

		ctx := t.Context()

		// Close events channel immediately to simulate empty input
		close(eventsCh)

		// Start consuming subscription
		done := make(chan struct{})
		go func() {
			svc.dispatchSubscriptionEvents(ctx, "ethereum", eventsCh, blocksCh, errorsCh)
			close(done)
		}()

		// Function should return quickly when events channel is closed
		select {
		case <-done:
			// Expected - function should return when eventsCh is closed
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Function should return quickly when events channel is closed")
		}

		// Verify no messages were sent to output channels
		select {
		case <-blocksCh:
			t.Fatal("Expected no messages in blocks channel")
		case <-errorsCh:
			t.Fatal("Expected no messages in errors channel")
		default:
			// Expected - no messages should be sent
		}
	})

	t.Run("different network names", func(t *testing.T) {
		// Create service
		svc := &service{}

		// Create channels
		eventsCh := make(chan BlockchainEvent, 2)
		blocksCh := make(chan ObservedBlock, 2)
		errorsCh := make(chan BlockDispatchFailure, 2)

		ctx := t.Context()

		// Test with different network name
		networkName := "polygon"

		// Start consuming subscription
		go svc.dispatchSubscriptionEvents(ctx, networkName, eventsCh, blocksCh, errorsCh)

		// Send events
		eventsCh <- BlockchainEvent{
			Height: types.Hex("0x500"),
			Block: Block{
				Height: types.Hex("0x500"),
				Hash:   "hash500",
			},
			Err: nil,
		}
		eventsCh <- BlockchainEvent{
			Height: types.Hex("0x501"),
			Block:  Block{},
			Err:    errors.New("test error"),
		}
		close(eventsCh)

		// Collect results
		var receivedBlocks []ObservedBlock
		var receivedErrors []BlockDispatchFailure
		timeout := time.After(2 * time.Second)
		expectedTotal := 2
		receivedTotal := 0

		for receivedTotal < expectedTotal {
			select {
			case block := <-blocksCh:
				receivedBlocks = append(receivedBlocks, block)
				receivedTotal++
			case err := <-errorsCh:
				receivedErrors = append(receivedErrors, err)
				receivedTotal++
			case <-timeout:
				t.Fatalf("Timeout waiting for events. Received %d out of %d expected events", receivedTotal, expectedTotal)
			}
		}

		// Verify network names are correctly set
		require.Len(t, receivedBlocks, 1)
		require.Len(t, receivedErrors, 1)

		assert.Equal(t, networkName, receivedBlocks[0].Network)
		assert.Equal(t, networkName, receivedErrors[0].Network)
	})
}

func TestService_launchAllNetworkSubscriptions(t *testing.T) {
	t.Run("successful subscription with no checkpoint", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := &service{
			networks: map[string]Blockchain{
				"ethereum": blockchainMock,
			},
			checkpointStorage: checkpointMock,
		}

		// Mock no checkpoint found
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Mock successful subscription with empty start height
		eventsCh := make(chan BlockchainEvent)
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(eventsCh), nil)

		// Create channels
		blocksCh := make(chan ObservedBlock, 1)
		errorsCh := make(chan BlockDispatchFailure, 1)

		ctx := t.Context()

		// Start subscriptions
		err := svc.launchAllNetworkSubscriptions(ctx, blocksCh, errorsCh)

		// Verify no error
		assert.NoError(t, err)

		// Close the events channel to clean up the goroutine
		close(eventsCh)
	})

	t.Run("successful subscription with existing checkpoint", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := &service{
			networks: map[string]Blockchain{
				"ethereum": blockchainMock,
			},
			checkpointStorage: checkpointMock,
		}

		// Mock existing checkpoint
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex("0x100"), nil)

		// Mock successful subscription with incremented start height (0x100 + 1 = 0x101)
		eventsCh := make(chan BlockchainEvent)
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("0x101")).
			Return((<-chan BlockchainEvent)(eventsCh), nil)

		// Create channels
		blocksCh := make(chan ObservedBlock, 1)
		errorsCh := make(chan BlockDispatchFailure, 1)

		ctx := t.Context()

		// Start subscriptions
		err := svc.launchAllNetworkSubscriptions(ctx, blocksCh, errorsCh)

		// Verify no error
		assert.NoError(t, err)

		// Close the events channel to clean up the goroutine
		close(eventsCh)
	})

	t.Run("multiple networks", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		ethereumMock := NewBlockchainMock(t)
		bitcoinMock := NewBlockchainMock(t)

		// Create service with multiple networks
		svc := &service{
			networks: map[string]Blockchain{
				"ethereum": ethereumMock,
				"bitcoin":  bitcoinMock,
			},
			checkpointStorage: checkpointMock,
		}

		// Mock checkpoints for both networks
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "bitcoin").
			Return(types.Hex("0x200"), nil)

		// Mock successful subscriptions
		ethereumEventsCh := make(chan BlockchainEvent)
		bitcoinEventsCh := make(chan BlockchainEvent)

		ethereumMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(ethereumEventsCh), nil)
		bitcoinMock.EXPECT().Subscribe(mock.Anything, types.Hex("0x201")).
			Return((<-chan BlockchainEvent)(bitcoinEventsCh), nil)

		// Create channels
		blocksCh := make(chan ObservedBlock, 2)
		errorsCh := make(chan BlockDispatchFailure, 2)

		ctx := t.Context()

		// Start subscriptions
		err := svc.launchAllNetworkSubscriptions(ctx, blocksCh, errorsCh)

		// Verify no error
		assert.NoError(t, err)

		// Close the events channels to clean up the goroutines
		close(ethereumEventsCh)
		close(bitcoinEventsCh)
	})

	t.Run("checkpoint load error", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := &service{
			networks: map[string]Blockchain{
				"ethereum": blockchainMock,
			},
			checkpointStorage: checkpointMock,
		}

		// Mock checkpoint load error (not ErrNoCheckpointFound)
		checkpointError := errors.New("database connection failed")
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), checkpointError)

		// Create channels
		blocksCh := make(chan ObservedBlock, 1)
		errorsCh := make(chan BlockDispatchFailure, 1)

		ctx := t.Context()

		// Start subscriptions
		err := svc.launchAllNetworkSubscriptions(ctx, blocksCh, errorsCh)

		// Verify error is returned
		assert.Error(t, err)
		assert.Equal(t, checkpointError, err)
	})

	t.Run("subscription error", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := &service{
			networks: map[string]Blockchain{
				"ethereum": blockchainMock,
			},
			checkpointStorage: checkpointMock,
		}

		// Mock successful checkpoint load
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Mock subscription error
		subscriptionError := errors.New("failed to connect to blockchain node")
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(nil), subscriptionError)

		// Create channels
		blocksCh := make(chan ObservedBlock, 1)
		errorsCh := make(chan BlockDispatchFailure, 1)

		ctx := t.Context()

		// Start subscriptions
		err := svc.launchAllNetworkSubscriptions(ctx, blocksCh, errorsCh)

		// Verify error is returned
		assert.Error(t, err)
		assert.Equal(t, subscriptionError, err)
	})

	t.Run("no networks configured", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)

		// Create service with no networks
		svc := &service{
			networks:          map[string]Blockchain{},
			checkpointStorage: checkpointMock,
		}

		// Create channels
		blocksCh := make(chan ObservedBlock, 1)
		errorsCh := make(chan BlockDispatchFailure, 1)

		ctx := t.Context()

		// Start subscriptions
		err := svc.launchAllNetworkSubscriptions(ctx, blocksCh, errorsCh)

		// Verify no error (empty loop should complete successfully)
		assert.NoError(t, err)
	})

	t.Run("integration with dispatchSubscriptionEvents", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := &service{
			networks: map[string]Blockchain{
				"ethereum": blockchainMock,
			},
			checkpointStorage: checkpointMock,
		}

		// Mock checkpoint load
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Create events channel that we can control
		eventsCh := make(chan BlockchainEvent, 1)
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(eventsCh), nil)

		// Create channels
		blocksCh := make(chan ObservedBlock, 1)
		errorsCh := make(chan BlockDispatchFailure, 1)

		ctx := t.Context()

		// Start subscriptions
		err := svc.launchAllNetworkSubscriptions(ctx, blocksCh, errorsCh)
		assert.NoError(t, err)

		// Send a test event to verify the dispatchSubscriptionEvents goroutine is working
		testBlock := Block{
			Height: types.Hex("0x123"),
			Hash:   "test-hash",
		}
		eventsCh <- BlockchainEvent{
			Height: types.Hex("0x123"),
			Block:  testBlock,
			Err:    nil,
		}
		close(eventsCh)

		// Verify the block is received
		select {
		case receivedBlock := <-blocksCh:
			assert.Equal(t, "ethereum", receivedBlock.Network)
			assert.Equal(t, types.Hex("0x123"), receivedBlock.Height)
			assert.Equal(t, "test-hash", receivedBlock.Hash)
		case <-time.After(1 * time.Second):
			t.Fatal("Expected block to be received from dispatchSubscriptionEvents goroutine")
		}
	})
}
