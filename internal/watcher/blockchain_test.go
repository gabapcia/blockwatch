package watcher

import (
	"context"
	"errors"
	"testing"
	"time"

	retrytest "github.com/gabapcia/blockwatch/internal/pkg/resilience/retry/mocks"
	"github.com/gabapcia/blockwatch/internal/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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
		retryCh := make(chan NetworkError, 1)
		recoveredCh := make(chan NetworkBlock, 1)
		finalErrorCh := make(chan NetworkError, 1)

		// Use test context
		ctx := t.Context()

		// Start the retry function in a goroutine
		go svc.retryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Send the network error to retry channel
		inputError := NetworkError{
			Network: "ethereum",
			Height:  types.Hex("0x123"),
			Err:     errors.New("original fetch error"),
		}
		retryCh <- inputError
		close(retryCh) // Close to signal end of input

		// Wait for recovered block
		select {
		case recovered := <-recoveredCh:
			assert.Equal(t, "ethereum", recovered.Network)
			assert.Equal(t, types.Hex("0x123"), recovered.Block.Height)
			assert.Equal(t, "block-hash-123", recovered.Block.Hash)
			assert.Equal(t, []Transaction{{Hash: "tx1", From: "addr1", To: "addr2"}}, recovered.Block.Transactions)
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
			RunAndReturn(func(ctx context.Context, operation func() error) error {
				// Execute the operation to simulate the actual retry logic
				return operation()
			}).Return(ErrNetworkNotRegistered)

		// Create channels
		retryCh := make(chan NetworkError, 1)
		recoveredCh := make(chan NetworkBlock, 1)
		finalErrorCh := make(chan NetworkError, 1)

		// Use test context
		ctx := t.Context()

		// Start the retry function
		go svc.retryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Send network error for unknown network
		inputError := NetworkError{
			Network: "unknown-network",
			Height:  types.Hex("0x456"),
			Err:     errors.New("original fetch error"),
		}
		retryCh <- inputError
		close(retryCh)

		// Wait for final error
		select {
		case finalErr := <-finalErrorCh:
			assert.Equal(t, "unknown-network", finalErr.Network)
			assert.Equal(t, types.Hex("0x456"), finalErr.Height)
			assert.Contains(t, finalErr.Err.Error(), "original fetch error")
			assert.Contains(t, finalErr.Err.Error(), "network not registered")
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
			}).Return(persistentErr)

		// Mock blockchain client that returns error
		blockchainMock.EXPECT().FetchBlockByHeight(mock.Anything, types.Hex("0x789")).
			Return(Block{}, persistentErr)

		// Create channels
		retryCh := make(chan NetworkError, 1)
		recoveredCh := make(chan NetworkBlock, 1)
		finalErrorCh := make(chan NetworkError, 1)

		// Use test context
		ctx := t.Context()

		// Start the retry function
		go svc.retryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Send network error
		inputError := NetworkError{
			Network: "ethereum",
			Height:  types.Hex("0x789"),
			Err:     errors.New("original fetch error"),
		}
		retryCh <- inputError
		close(retryCh)

		// Wait for final error
		select {
		case finalErr := <-finalErrorCh:
			assert.Equal(t, "ethereum", finalErr.Network)
			assert.Equal(t, types.Hex("0x789"), finalErr.Height)
			assert.Contains(t, finalErr.Err.Error(), "original fetch error")
			assert.Contains(t, finalErr.Err.Error(), "persistent blockchain error")
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
			Return(context.Canceled)

		// Create channels
		retryCh := make(chan NetworkError, 1)
		recoveredCh := make(chan NetworkBlock, 1)
		finalErrorCh := make(chan NetworkError, 1)

		// Use test context
		ctx := t.Context()

		// Start the retry function
		go svc.retryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Send network error
		inputError := NetworkError{
			Network: "ethereum",
			Height:  types.Hex("0xabc"),
			Err:     errors.New("original fetch error"),
		}
		retryCh <- inputError
		close(retryCh)

		// Wait for final error
		select {
		case finalErr := <-finalErrorCh:
			assert.Equal(t, "ethereum", finalErr.Network)
			assert.Equal(t, types.Hex("0xabc"), finalErr.Height)
			assert.Contains(t, finalErr.Err.Error(), "original fetch error")
			assert.Contains(t, finalErr.Err.Error(), "context canceled")
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
			}).Return(errors.New("persistent error")).Once()

		blockchainMock.EXPECT().FetchBlockByHeight(mock.Anything, types.Hex("0x200")).
			Return(Block{}, errors.New("persistent error")).Once()

		// Create channels
		retryCh := make(chan NetworkError, 2)
		recoveredCh := make(chan NetworkBlock, 2)
		finalErrorCh := make(chan NetworkError, 2)

		ctx := t.Context()

		// Start the retry function
		go svc.retryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Send two network errors
		retryCh <- NetworkError{
			Network: "ethereum",
			Height:  types.Hex("0x100"),
			Err:     errors.New("error 1"),
		}
		retryCh <- NetworkError{
			Network: "ethereum",
			Height:  types.Hex("0x200"),
			Err:     errors.New("error 2"),
		}
		close(retryCh)

		// Collect results
		var recoveredBlocks []NetworkBlock
		var finalErrors []NetworkError

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
		assert.Equal(t, types.Hex("0x100"), recoveredBlocks[0].Block.Height)
		assert.Equal(t, "hash100", recoveredBlocks[0].Block.Hash)

		// Check final error
		assert.Equal(t, "ethereum", finalErrors[0].Network)
		assert.Equal(t, types.Hex("0x200"), finalErrors[0].Height)
		assert.Contains(t, finalErrors[0].Err.Error(), "error 2")
		assert.Contains(t, finalErrors[0].Err.Error(), "persistent error")
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
		retryCh := make(chan NetworkError)
		recoveredCh := make(chan NetworkBlock, 1)
		finalErrorCh := make(chan NetworkError, 1)

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

		// Mock retry that returns an error
		retryMock.EXPECT().Execute(mock.Anything, mock.AnythingOfType("func() error")).
			Return(errors.New("retry failed"))

		// Create channels with no buffer to simulate blocking
		retryCh := make(chan NetworkError, 1)
		recoveredCh := make(chan NetworkBlock)
		finalErrorCh := make(chan NetworkError) // No buffer - will block

		// Create context that will be canceled
		ctx, cancel := context.WithCancel(t.Context())

		// Start the retry function
		done := make(chan struct{})
		go func() {
			svc.retryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)
			close(done)
		}()

		// Send network error
		retryCh <- NetworkError{
			Network: "unknown",
			Height:  types.Hex("0x123"),
			Err:     errors.New("original error"),
		}

		// Cancel context immediately to simulate context done before final error can be sent
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
		retryCh := make(chan NetworkError, 1)
		recoveredCh := make(chan NetworkBlock, 1)
		finalErrorCh := make(chan NetworkError, 1)

		ctx := t.Context()

		// Start the retry function using the wrapper
		svc.startRetryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Send network error
		inputError := NetworkError{
			Network: "ethereum",
			Height:  types.Hex("0x123"),
			Err:     errors.New("original error"),
		}
		retryCh <- inputError
		close(retryCh)

		// Verify that the goroutine processes the error
		select {
		case recovered := <-recoveredCh:
			assert.Equal(t, "ethereum", recovered.Network)
			assert.Equal(t, types.Hex("0x123"), recovered.Block.Height)
			assert.Equal(t, "test-hash", recovered.Block.Hash)
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
		retryCh := make(chan NetworkError)
		recoveredCh := make(chan NetworkBlock)
		finalErrorCh := make(chan NetworkError)

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
			Return(errors.New("retry failed"))

		// Create channels
		retryCh := make(chan NetworkError, 1)
		recoveredCh := make(chan NetworkBlock)
		finalErrorCh := make(chan NetworkError, 1)

		// Create context that will be canceled
		ctx, cancel := context.WithCancel(t.Context())

		// Start the retry function
		svc.startRetryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)

		// Send network error
		retryCh <- NetworkError{
			Network: "unknown",
			Height:  types.Hex("0x123"),
			Err:     errors.New("original error"),
		}

		// Wait for final error to be processed
		select {
		case finalErr := <-finalErrorCh:
			assert.Equal(t, "unknown", finalErr.Network)
			assert.Equal(t, types.Hex("0x123"), finalErr.Height)
			assert.Contains(t, finalErr.Err.Error(), "original error")
			assert.Contains(t, finalErr.Err.Error(), "retry failed")
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
		blocksCh := make(chan NetworkBlock, 2)
		errorsCh := make(chan NetworkError, 2)

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
		var receivedBlocks []NetworkBlock
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
		assert.Equal(t, types.Hex("0x100"), receivedBlocks[0].Block.Height)
		assert.Equal(t, "hash100", receivedBlocks[0].Block.Hash)
		assert.Equal(t, []Transaction{{Hash: "tx1", From: "addr1", To: "addr2"}}, receivedBlocks[0].Block.Transactions)

		// Check second block
		assert.Equal(t, "ethereum", receivedBlocks[1].Network)
		assert.Equal(t, types.Hex("0x101"), receivedBlocks[1].Block.Height)
		assert.Equal(t, "hash101", receivedBlocks[1].Block.Hash)
		assert.Equal(t, []Transaction{{Hash: "tx2", From: "addr3", To: "addr4"}}, receivedBlocks[1].Block.Transactions)

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
		blocksCh := make(chan NetworkBlock, 2)
		errorsCh := make(chan NetworkError, 2)

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
		var receivedErrors []NetworkError
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
		assert.Equal(t, "fetch error 1", receivedErrors[0].Err.Error())

		// Check second error
		assert.Equal(t, "ethereum", receivedErrors[1].Network)
		assert.Equal(t, types.Hex("0x201"), receivedErrors[1].Height)
		assert.Equal(t, "fetch error 2", receivedErrors[1].Err.Error())

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
		blocksCh := make(chan NetworkBlock, 3)
		errorsCh := make(chan NetworkError, 3)

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
		var receivedBlocks []NetworkBlock
		var receivedErrors []NetworkError
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
		assert.Equal(t, types.Hex("0x300"), receivedBlocks[0].Block.Height)
		assert.Equal(t, "hash300", receivedBlocks[0].Block.Hash)

		assert.Equal(t, "bitcoin", receivedBlocks[1].Network)
		assert.Equal(t, types.Hex("0x302"), receivedBlocks[1].Block.Height)
		assert.Equal(t, "hash302", receivedBlocks[1].Block.Hash)

		// Check error
		assert.Equal(t, "bitcoin", receivedErrors[0].Network)
		assert.Equal(t, types.Hex("0x301"), receivedErrors[0].Height)
		assert.Equal(t, "network error", receivedErrors[0].Err.Error())
	})

	t.Run("context canceled while processing", func(t *testing.T) {
		// Create service
		svc := &service{}

		// Create channels
		eventsCh := make(chan BlockchainEvent, 2)
		blocksCh := make(chan NetworkBlock) // No buffer to simulate blocking
		errorsCh := make(chan NetworkError, 2)

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
		blocksCh := make(chan NetworkBlock, 1)
		errorsCh := make(chan NetworkError, 1)

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
		blocksCh := make(chan NetworkBlock, 2)
		errorsCh := make(chan NetworkError, 2)

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
		var receivedBlocks []NetworkBlock
		var receivedErrors []NetworkError
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
		blocksCh := make(chan NetworkBlock, 1)
		errorsCh := make(chan NetworkError, 1)

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
		blocksCh := make(chan NetworkBlock, 1)
		errorsCh := make(chan NetworkError, 1)

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
		blocksCh := make(chan NetworkBlock, 2)
		errorsCh := make(chan NetworkError, 2)

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
		blocksCh := make(chan NetworkBlock, 1)
		errorsCh := make(chan NetworkError, 1)

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
		blocksCh := make(chan NetworkBlock, 1)
		errorsCh := make(chan NetworkError, 1)

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
		blocksCh := make(chan NetworkBlock, 1)
		errorsCh := make(chan NetworkError, 1)

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
		blocksCh := make(chan NetworkBlock, 1)
		errorsCh := make(chan NetworkError, 1)

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
			assert.Equal(t, types.Hex("0x123"), receivedBlock.Block.Height)
			assert.Equal(t, "test-hash", receivedBlock.Block.Hash)
		case <-time.After(1 * time.Second):
			t.Fatal("Expected block to be received from dispatchSubscriptionEvents goroutine")
		}
	})
}
