package chainwatch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gabapcia/blockwatch/internal/pkg/logger"
	retrytest "github.com/gabapcia/blockwatch/internal/pkg/resilience/retry/mocks"
	"github.com/gabapcia/blockwatch/internal/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func init() {
	// Initialize logger for tests to prevent nil pointer dereference
	_ = logger.Init("error") // Use error level to reduce test output
}

func TestService_Start(t *testing.T) {
	t.Run("successful start with no retry", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := New(map[string]Blockchain{
			"ethereum": blockchainMock,
		}, WithCheckpointStorage(checkpointMock))

		// Mock checkpoint load
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Mock successful subscription
		eventsCh := make(chan BlockchainEvent)
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(eventsCh), nil)

		ctx := t.Context()

		// Start service
		observedBlockCh, err := svc.Start(ctx)

		// Verify success
		assert.NoError(t, err)
		assert.NotNil(t, observedBlockCh)

		// Verify service is marked as started
		assert.True(t, svc.isStarted)

		// Clean up
		svc.Close()
		close(eventsCh)
	})

	t.Run("successful start with retry", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)
		retryMock := retrytest.NewRetry(t)

		// Create service with retry
		svc := New(map[string]Blockchain{
			"ethereum": blockchainMock,
		}, WithCheckpointStorage(checkpointMock), WithRetry(retryMock))

		// Mock checkpoint load
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Mock successful subscription
		eventsCh := make(chan BlockchainEvent)
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(eventsCh), nil)

		ctx := t.Context()

		// Start service
		observedBlockCh, err := svc.Start(ctx)

		// Verify success
		assert.NoError(t, err)
		assert.NotNil(t, observedBlockCh)
		assert.True(t, svc.isStarted)

		// Clean up
		svc.Close()
		close(eventsCh)
	})

	t.Run("start with custom dispatch failure handler", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Track handler calls
		var handlerCalls []BlockDispatchFailure
		var mu sync.Mutex
		customHandler := func(ctx context.Context, failure BlockDispatchFailure) {
			mu.Lock()
			defer mu.Unlock()
			handlerCalls = append(handlerCalls, failure)
		}

		// Create service with custom handler
		svc := New(map[string]Blockchain{
			"ethereum": blockchainMock,
		}, WithCheckpointStorage(checkpointMock), WithDispatchFailureHandler(customHandler))

		// Mock checkpoint load
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Mock successful subscription
		eventsCh := make(chan BlockchainEvent)
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(eventsCh), nil)

		ctx := t.Context()

		// Start service
		observedBlockCh, err := svc.Start(ctx)

		// Verify success
		assert.NoError(t, err)
		assert.NotNil(t, observedBlockCh)
		assert.True(t, svc.isStarted)

		// Clean up
		svc.Close()
		close(eventsCh)
	})

	t.Run("already started error", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := New(map[string]Blockchain{
			"ethereum": blockchainMock,
		}, WithCheckpointStorage(checkpointMock))

		// Mock checkpoint load for first start
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Mock successful subscription for first start
		eventsCh := make(chan BlockchainEvent)
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(eventsCh), nil)

		ctx := t.Context()

		// Start service first time
		observedBlockCh1, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, observedBlockCh1)

		// Try to start again
		observedBlockCh2, err := svc.Start(ctx)

		// Verify error
		assert.Error(t, err)
		assert.Equal(t, ErrServiceAlreadyStarted, err)
		assert.Nil(t, observedBlockCh2)

		// Clean up
		svc.Close()
		close(eventsCh)
	})

	t.Run("checkpoint load error", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := New(map[string]Blockchain{
			"ethereum": blockchainMock,
		}, WithCheckpointStorage(checkpointMock))

		// Mock checkpoint load error
		checkpointError := errors.New("database connection failed")
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), checkpointError)

		ctx := t.Context()

		// Start service
		observedBlockCh, err := svc.Start(ctx)

		// Verify error
		assert.Error(t, err)
		assert.Equal(t, checkpointError, err)
		assert.Nil(t, observedBlockCh)
		assert.False(t, svc.isStarted)
	})

	t.Run("subscription error", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := New(map[string]Blockchain{
			"ethereum": blockchainMock,
		}, WithCheckpointStorage(checkpointMock))

		// Mock successful checkpoint load
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Mock subscription error
		subscriptionError := errors.New("failed to connect to blockchain node")
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(nil), subscriptionError)

		ctx := t.Context()

		// Start service
		observedBlockCh, err := svc.Start(ctx)

		// Verify error
		assert.Error(t, err)
		assert.Equal(t, subscriptionError, err)
		assert.Nil(t, observedBlockCh)
		assert.False(t, svc.isStarted)
	})

	t.Run("multiple networks", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		ethereumMock := NewBlockchainMock(t)
		bitcoinMock := NewBlockchainMock(t)

		// Create service with multiple networks
		svc := New(map[string]Blockchain{
			"ethereum": ethereumMock,
			"bitcoin":  bitcoinMock,
		}, WithCheckpointStorage(checkpointMock))

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

		ctx := t.Context()

		// Start service
		observedBlockCh, err := svc.Start(ctx)

		// Verify success
		assert.NoError(t, err)
		assert.NotNil(t, observedBlockCh)
		assert.True(t, svc.isStarted)

		// Clean up
		svc.Close()
		close(ethereumEventsCh)
		close(bitcoinEventsCh)
	})

	t.Run("no networks configured", func(t *testing.T) {
		// Create service with no networks
		svc := New(map[string]Blockchain{})

		ctx := t.Context()

		// Start service
		observedBlockCh, err := svc.Start(ctx)

		// Verify success (empty service should start successfully)
		assert.NoError(t, err)
		assert.NotNil(t, observedBlockCh)
		assert.True(t, svc.isStarted)

		// Clean up
		svc.Close()
	})

	t.Run("integration test - end to end block processing", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := New(map[string]Blockchain{
			"ethereum": blockchainMock,
		}, WithCheckpointStorage(checkpointMock))

		// Mock checkpoint load
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Create events channel that we can control
		eventsCh := make(chan BlockchainEvent, 2)
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(eventsCh), nil)

		// Mock successful checkpoint save for the integration test
		checkpointMock.EXPECT().SaveCheckpoint(mock.Anything, "ethereum", types.Hex("0x123")).
			Return(nil)

		ctx := t.Context()

		// Start service
		observedBlockCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, observedBlockCh)

		// Send test events
		testBlock := Block{
			Height: types.Hex("0x123"),
			Hash:   "test-hash",
			Transactions: []Transaction{
				{Hash: "tx1", From: "addr1", To: "addr2"},
			},
		}

		eventsCh <- BlockchainEvent{
			Height: types.Hex("0x123"),
			Block:  testBlock,
			Err:    nil,
		}
		close(eventsCh)

		// Verify the block is received
		select {
		case receivedBlock := <-observedBlockCh:
			assert.Equal(t, "ethereum", receivedBlock.Network)
			assert.Equal(t, types.Hex("0x123"), receivedBlock.Height)
			assert.Equal(t, "test-hash", receivedBlock.Hash)
			assert.Len(t, receivedBlock.Transactions, 1)
			assert.Equal(t, "tx1", receivedBlock.Transactions[0].Hash)
		case <-time.After(1 * time.Second):
			t.Fatal("Expected block to be received")
		}

		// Clean up
		svc.Close()
	})
}

func TestService_Close(t *testing.T) {
	t.Run("close unstarted service", func(t *testing.T) {
		// Create service
		svc := New(map[string]Blockchain{})

		// Close should not panic
		svc.Close()

		// Verify state
		assert.False(t, svc.isStarted)
		assert.Nil(t, svc.closeFunc)
	})

	t.Run("close started service", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := New(map[string]Blockchain{
			"ethereum": blockchainMock,
		}, WithCheckpointStorage(checkpointMock))

		// Mock checkpoint load
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Mock successful subscription
		eventsCh := make(chan BlockchainEvent)
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(eventsCh), nil)

		ctx := t.Context()

		// Start service
		observedBlockCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, observedBlockCh)
		assert.True(t, svc.isStarted)

		// Close service
		svc.Close()

		// Verify state
		assert.False(t, svc.isStarted)
		assert.Nil(t, svc.closeFunc)

		// Verify channels are closed
		select {
		case _, ok := <-observedBlockCh:
			assert.False(t, ok, "observedBlockCh should be closed")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("observedBlockCh should be closed")
		}

		// Clean up
		close(eventsCh)
	})

	t.Run("multiple close calls", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := New(map[string]Blockchain{
			"ethereum": blockchainMock,
		}, WithCheckpointStorage(checkpointMock))

		// Mock checkpoint load
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Mock successful subscription
		eventsCh := make(chan BlockchainEvent)
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(eventsCh), nil)

		ctx := t.Context()

		// Start service
		observedBlockCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, observedBlockCh)

		// Close multiple times - should not panic
		svc.Close()
		svc.Close()
		svc.Close()

		// Verify state
		assert.False(t, svc.isStarted)
		assert.Nil(t, svc.closeFunc)

		// Clean up
		close(eventsCh)
	})

	t.Run("close with retry enabled", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)
		retryMock := retrytest.NewRetry(t)

		// Create service with retry
		svc := New(map[string]Blockchain{
			"ethereum": blockchainMock,
		}, WithCheckpointStorage(checkpointMock), WithRetry(retryMock))

		// Mock checkpoint load
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Mock successful subscription
		eventsCh := make(chan BlockchainEvent)
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(eventsCh), nil)

		ctx := t.Context()

		// Start service
		observedBlockCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, observedBlockCh)

		// Close service
		svc.Close()

		// Verify state
		assert.False(t, svc.isStarted)
		assert.Nil(t, svc.closeFunc)

		// Clean up
		close(eventsCh)
	})

	t.Run("context cancellation stops dispatch", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := New(map[string]Blockchain{
			"ethereum": blockchainMock,
		}, WithCheckpointStorage(checkpointMock))

		// Mock checkpoint load
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Mock successful subscription
		eventsCh := make(chan BlockchainEvent)
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(eventsCh), nil)

		ctx, cancel := context.WithCancel(t.Context())

		// Start service
		observedBlockCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, observedBlockCh)

		// Cancel the context
		cancel()

		// Send an event - it should not be processed
		eventsCh <- BlockchainEvent{
			Height: types.Hex("0x123"),
			Block:  Block{Height: types.Hex("0x123")},
		}

		// Verify no block is received
		select {
		case <-observedBlockCh:
			t.Fatal("Received unexpected block after context cancellation")
		case <-time.After(100 * time.Millisecond):
			// Expected behavior
		}

		// Clean up
		svc.Close()
		close(eventsCh)
	})
}

func TestNew(t *testing.T) {
	t.Run("create service with default options", func(t *testing.T) {
		// Create networks map
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
			"bitcoin":  NewBlockchainMock(t),
		}

		// Create service
		svc := New(networks)

		// Verify service configuration
		assert.NotNil(t, svc)
		assert.Equal(t, networks, svc.networks)
		assert.NotNil(t, svc.checkpointStorage)
		assert.Nil(t, svc.retry)
		assert.NotNil(t, svc.dispatchFailureHandler)
		assert.False(t, svc.isStarted)
		assert.Nil(t, svc.closeFunc)

		// Verify default checkpoint storage is nopCheckpoint
		_, ok := svc.checkpointStorage.(nopCheckpoint)
		assert.True(t, ok, "Default checkpoint storage should be nopCheckpoint")
	})

	t.Run("create service with all options", func(t *testing.T) {
		// Setup dependencies
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}
		checkpointMock := NewCheckpointStorageMock(t)
		retryMock := retrytest.NewRetry(t)

		// Track handler calls
		var handlerCalls []BlockDispatchFailure
		var mu sync.Mutex
		customHandler := func(ctx context.Context, failure BlockDispatchFailure) {
			mu.Lock()
			defer mu.Unlock()
			handlerCalls = append(handlerCalls, failure)
		}

		// Create service with all options
		svc := New(networks,
			WithCheckpointStorage(checkpointMock),
			WithRetry(retryMock),
			WithDispatchFailureHandler(customHandler))

		// Verify service configuration
		assert.NotNil(t, svc)
		assert.Equal(t, networks, svc.networks)
		assert.Equal(t, checkpointMock, svc.checkpointStorage)
		assert.Equal(t, retryMock, svc.retry)
		assert.NotNil(t, svc.dispatchFailureHandler)
		assert.False(t, svc.isStarted)
		assert.Nil(t, svc.closeFunc)
	})

	t.Run("create service with empty networks", func(t *testing.T) {
		// Create service with empty networks
		svc := New(map[string]Blockchain{})

		// Verify service configuration
		assert.NotNil(t, svc)
		assert.Empty(t, svc.networks)
		assert.NotNil(t, svc.checkpointStorage)
		assert.Nil(t, svc.retry)
		assert.NotNil(t, svc.dispatchFailureHandler)
	})

	t.Run("create service with nil networks", func(t *testing.T) {
		// Create service with nil networks
		svc := New(nil)

		// Verify service configuration
		assert.NotNil(t, svc)
		assert.Nil(t, svc.networks)
		assert.NotNil(t, svc.checkpointStorage)
		assert.Nil(t, svc.retry)
		assert.NotNil(t, svc.dispatchFailureHandler)
	})
}

func TestNewWithTransform(t *testing.T) {
	t.Run("create service with transform function", func(t *testing.T) {
		// Create networks map
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Define a transform function
		transformFunc := func(ob ObservedBlock) string {
			return fmt.Sprintf("transformed-%s", ob.Height)
		}

		// Create service with transform function
		svc := NewWithTransform(networks, transformFunc)

		// Verify service configuration
		assert.NotNil(t, svc)
		assert.Equal(t, networks, svc.networks)
		assert.NotNil(t, svc.checkpointStorage)
		assert.Nil(t, svc.retry)
		assert.NotNil(t, svc.dispatchFailureHandler)
		assert.NotNil(t, svc.transformFunc)

		// Test the transform function
		testBlock := ObservedBlock{Block: Block{Height: types.Hex("0x123")}}
		transformed := svc.transformFunc(testBlock)
		assert.Equal(t, "transformed-0x123", transformed)
	})

	t.Run("create service with all options and transform", func(t *testing.T) {
		// Setup dependencies
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}
		checkpointMock := NewCheckpointStorageMock(t)
		retryMock := retrytest.NewRetry(t)
		customHandler := func(ctx context.Context, failure BlockDispatchFailure) {}
		transformFunc := func(ob ObservedBlock) int {
			return 123
		}

		// Create service with all options
		svc := NewWithTransform(networks, transformFunc,
			WithCheckpointStorage(checkpointMock),
			WithRetry(retryMock),
			WithDispatchFailureHandler(customHandler))

		// Verify service configuration
		assert.NotNil(t, svc)
		assert.Equal(t, networks, svc.networks)
		assert.Equal(t, checkpointMock, svc.checkpointStorage)
		assert.Equal(t, retryMock, svc.retry)
		assert.NotNil(t, svc.dispatchFailureHandler)
		assert.NotNil(t, svc.transformFunc)
	})
}

func TestDefaultOnDispatchFailure(t *testing.T) {
	t.Run("logs dispatch failure", func(t *testing.T) {
		// Create test failure
		failure := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x123"),
			Errors: []error{
				errors.New("network error"),
				errors.New("retry error"),
			},
		}

		ctx := t.Context()

		// Call default handler - should not panic
		defaultOnDispatchFailure(ctx, failure)

		// Note: We can't easily test the actual logging output without
		// modifying the logger package, but we can verify it doesn't panic
	})

	t.Run("handles empty errors", func(t *testing.T) {
		// Create test failure with no errors
		failure := BlockDispatchFailure{
			Network: "bitcoin",
			Height:  types.Hex("0x456"),
			Errors:  []error{},
		}

		ctx := t.Context()

		// Call default handler - should not panic
		defaultOnDispatchFailure(ctx, failure)
	})

	t.Run("handles nil errors", func(t *testing.T) {
		// Create test failure with nil errors
		failure := BlockDispatchFailure{
			Network: "polygon",
			Height:  types.Hex("0x789"),
			Errors:  nil,
		}

		ctx := t.Context()

		// Call default handler - should not panic
		defaultOnDispatchFailure(ctx, failure)
	})

	t.Run("handles context with values", func(t *testing.T) {
		// Create test failure
		failure := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0xabc"),
			Errors:  []error{errors.New("test error")},
		}

		// Create context with values
		type testKeyType struct{}
		var testKey testKeyType
		ctx := context.WithValue(t.Context(), testKey, "test-value")

		// Call default handler - should not panic
		defaultOnDispatchFailure(ctx, failure)
	})
}

func TestWithDispatchFailureHandler(t *testing.T) {
	t.Run("sets custom handler", func(t *testing.T) {
		// Track handler calls
		var handlerCalls []BlockDispatchFailure
		var mu sync.Mutex
		customHandler := func(ctx context.Context, failure BlockDispatchFailure) {
			mu.Lock()
			defer mu.Unlock()
			handlerCalls = append(handlerCalls, failure)
		}

		// Create service with custom handler
		svc := New(map[string]Blockchain{}, WithDispatchFailureHandler(customHandler))

		// Verify handler is set (we can't directly compare function pointers,
		// but we can verify it's not the default by testing behavior)
		assert.NotNil(t, svc.dispatchFailureHandler)

		// Test the handler works
		testFailure := BlockDispatchFailure{
			Network: "test",
			Height:  types.Hex("0x123"),
			Errors:  []error{errors.New("test error")},
		}

		svc.dispatchFailureHandler(t.Context(), testFailure)

		// Verify custom handler was called
		mu.Lock()
		defer mu.Unlock()
		require.Len(t, handlerCalls, 1)
		assert.Equal(t, "test", handlerCalls[0].Network)
		assert.Equal(t, types.Hex("0x123"), handlerCalls[0].Height)
	})

	t.Run("nil handler", func(t *testing.T) {
		// Create service with nil handler
		svc := New(map[string]Blockchain{}, WithDispatchFailureHandler(nil))

		// Verify handler is set to nil
		assert.Nil(t, svc.dispatchFailureHandler)
	})
}

func TestWithRetry(t *testing.T) {
	t.Run("sets retry", func(t *testing.T) {
		// Create retry mock
		retryMock := retrytest.NewRetry(t)

		// Create service with retry
		svc := New(map[string]Blockchain{}, WithRetry(retryMock))

		// Verify retry is set
		assert.Equal(t, retryMock, svc.retry)
	})

	t.Run("nil retry", func(t *testing.T) {
		// Create service with nil retry
		svc := New(map[string]Blockchain{}, WithRetry(nil))

		// Verify retry is nil
		assert.Nil(t, svc.retry)
	})
}

func TestWithCheckpointStorage(t *testing.T) {
	t.Run("sets checkpoint storage", func(t *testing.T) {
		// Create checkpoint storage mock
		checkpointMock := NewCheckpointStorageMock(t)

		// Create service with checkpoint storage
		svc := New(map[string]Blockchain{}, WithCheckpointStorage(checkpointMock))

		// Verify checkpoint storage is set
		assert.Equal(t, checkpointMock, svc.checkpointStorage)
	})

	t.Run("nil checkpoint storage", func(t *testing.T) {
		// Create service with nil checkpoint storage
		svc := New(map[string]Blockchain{}, WithCheckpointStorage(nil))

		// Verify checkpoint storage is nil
		assert.Nil(t, svc.checkpointStorage)
	})
}

func TestCheckpointAndForward(t *testing.T) {
	t.Run("successful block processing with checkpoint save", func(t *testing.T) {
		checkpointMock := NewCheckpointStorageMock(t)
		transformFunc := func(ob ObservedBlock) string {
			return fmt.Sprintf("transformed-%s-%s", ob.Network, ob.Height)
		}
		svc := &service[string]{
			checkpointStorage: checkpointMock,
			transformFunc:     transformFunc,
		}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		blockIn := make(chan ObservedBlock, 1)
		transformedOut := make(chan string, 1)

		checkpointMock.EXPECT().SaveCheckpoint(mock.Anything, "ethereum", types.Hex("0x123")).Return(nil)

		go svc.checkpointAndForward(ctx, blockIn, transformedOut)

		blockIn <- ObservedBlock{Network: "ethereum", Block: Block{Height: types.Hex("0x123"), Hash: "test-hash"}}
		close(blockIn)

		select {
		case result := <-transformedOut:
			assert.Equal(t, "transformed-ethereum-0x123", result)
		case <-time.After(1 * time.Second):
			t.Fatal("Expected transformed output, but none received")
		}
	})

	t.Run("checkpoint save failure does not stop processing", func(t *testing.T) {
		checkpointMock := NewCheckpointStorageMock(t)
		transformFunc := func(ob ObservedBlock) string {
			return fmt.Sprintf("transformed-%s-%s", ob.Network, ob.Height)
		}
		svc := &service[string]{
			checkpointStorage: checkpointMock,
			transformFunc:     transformFunc,
		}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		blockIn := make(chan ObservedBlock, 1)
		transformedOut := make(chan string, 1)

		checkpointMock.EXPECT().SaveCheckpoint(mock.Anything, "ethereum", types.Hex("0x123")).Return(errors.New("save failed"))

		go svc.checkpointAndForward(ctx, blockIn, transformedOut)

		blockIn <- ObservedBlock{Network: "ethereum", Block: Block{Height: types.Hex("0x123"), Hash: "test-hash"}}
		close(blockIn)

		select {
		case result := <-transformedOut:
			assert.Equal(t, "transformed-ethereum-0x123", result)
		case <-time.After(1 * time.Second):
			t.Fatal("Expected transformed output, but none received")
		}
	})

	t.Run("context cancellation stops processing", func(t *testing.T) {
		checkpointMock := NewCheckpointStorageMock(t)
		transformFunc := func(ob ObservedBlock) string {
			return fmt.Sprintf("transformed-%s-%s", ob.Network, ob.Height)
		}
		svc := &service[string]{
			checkpointStorage: checkpointMock,
			transformFunc:     transformFunc,
		}
		ctx, cancel := context.WithCancel(t.Context())
		blockIn := make(chan ObservedBlock, 1)
		transformedOut := make(chan string, 1)

		checkpointMock.EXPECT().SaveCheckpoint(mock.Anything, "ethereum", types.Hex("0x123")).Return(nil).Maybe()

		go svc.checkpointAndForward(ctx, blockIn, transformedOut)

		cancel()                          // Cancel the context immediately
		time.Sleep(50 * time.Millisecond) // Allow time for cancellation to propagate

		blockIn <- ObservedBlock{Network: "ethereum", Block: Block{Height: types.Hex("0x123"), Hash: "test-hash"}}
		close(blockIn)

		select {
		case <-transformedOut:
			t.Fatal("Received unexpected output after context cancellation")
		case <-time.After(100 * time.Millisecond):
			// Expected behavior: no output
		}
	})

	t.Run("input channel close stops processing", func(t *testing.T) {
		checkpointMock := NewCheckpointStorageMock(t)
		transformFunc := func(ob ObservedBlock) string {
			return fmt.Sprintf("transformed-%s-%s", ob.Network, ob.Height)
		}
		svc := &service[string]{
			checkpointStorage: checkpointMock,
			transformFunc:     transformFunc,
		}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		blockIn := make(chan ObservedBlock, 1)
		transformedOut := make(chan string, 1)

		go svc.checkpointAndForward(ctx, blockIn, transformedOut)

		close(blockIn) // Close the input channel

		select {
		case <-transformedOut:
			t.Fatal("Received unexpected output from empty input channel")
		case <-time.After(100 * time.Millisecond):
			// Expected behavior: no output
		}
	})

	t.Run("output channel send failure stops processing", func(t *testing.T) {
		checkpointMock := NewCheckpointStorageMock(t)
		transformFunc := func(ob ObservedBlock) string {
			return fmt.Sprintf("transformed-%s-%s", ob.Network, ob.Height)
		}
		svc := &service[string]{
			checkpointStorage: checkpointMock,
			transformFunc:     transformFunc,
		}
		ctx, cancel := context.WithCancel(t.Context())
		blockIn := make(chan ObservedBlock, 1)
		transformedOut := make(chan string) // Unbuffered channel

		checkpointMock.EXPECT().SaveCheckpoint(mock.Anything, "ethereum", types.Hex("0x123")).Return(nil)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.checkpointAndForward(ctx, blockIn, transformedOut)
		}()

		blockIn <- ObservedBlock{Network: "ethereum", Block: Block{Height: types.Hex("0x123"), Hash: "test-hash"}}

		// Wait for the checkpoint to be saved, then cancel
		time.Sleep(50 * time.Millisecond)
		cancel()

		wg.Wait() // Ensure the goroutine has exited

		select {
		case <-transformedOut:
			t.Fatal("Received unexpected output after send failure")
		default:
			// Expected behavior: no output
		}
	})
}

func TestStartCheckpointAndForward(t *testing.T) {
	t.Run("starts checkpointAndForward in a goroutine", func(t *testing.T) {
		startedCh := make(chan struct{})
		transformFunc := func(ob ObservedBlock) string {
			close(startedCh)
			return ""
		}

		svc := &service[string]{
			checkpointStorage: NewCheckpointStorageMock(t),
			transformFunc:     transformFunc,
		}

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		blockIn := make(chan ObservedBlock, 1)
		transformedOut := make(chan string, 1)

		// Mock the checkpoint save to prevent unexpected calls
		svc.checkpointStorage.(*CheckpointStorageMock).EXPECT().SaveCheckpoint(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

		svc.startCheckpointAndForward(ctx, blockIn, transformedOut)

		// Send a block to trigger the transform function
		blockIn <- ObservedBlock{}

		select {
		case <-startedCh:
			// Test passed, the goroutine was started
		case <-time.After(1 * time.Second):
			t.Fatal("checkpointAndForward was not started in a goroutine")
		}
	})
}

func TestService_ConcurrentOperations(t *testing.T) {
	t.Run("concurrent start and close", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := New(map[string]Blockchain{
			"ethereum": blockchainMock,
		}, WithCheckpointStorage(checkpointMock))

		// Mock checkpoint load
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Mock successful subscription
		eventsCh := make(chan BlockchainEvent)
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(eventsCh), nil)

		ctx := t.Context()

		// Start service
		observedBlockCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, observedBlockCh)

		// Concurrent operations
		var wg sync.WaitGroup
		wg.Add(2)

		// Goroutine 1: Try to start again (should fail)
		go func() {
			defer wg.Done()
			_, err := svc.Start(ctx)
			assert.Error(t, err)
			assert.Equal(t, ErrServiceAlreadyStarted, err)
		}()

		// Goroutine 2: Close service
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond) // Small delay to ensure start attempt happens first
			svc.Close()
		}()

		wg.Wait()

		// Clean up
		close(eventsCh)
	})

	t.Run("multiple concurrent close calls", func(t *testing.T) {
		// Setup mocks
		checkpointMock := NewCheckpointStorageMock(t)
		blockchainMock := NewBlockchainMock(t)

		// Create service
		svc := New(map[string]Blockchain{
			"ethereum": blockchainMock,
		}, WithCheckpointStorage(checkpointMock))

		// Mock checkpoint load
		checkpointMock.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").
			Return(types.Hex(""), ErrNoCheckpointFound)

		// Mock successful subscription
		eventsCh := make(chan BlockchainEvent)
		blockchainMock.EXPECT().Subscribe(mock.Anything, types.Hex("")).
			Return((<-chan BlockchainEvent)(eventsCh), nil)

		ctx := t.Context()

		// Start service
		observedBlockCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, observedBlockCh)

		// Multiple concurrent close calls
		var wg sync.WaitGroup
		numGoroutines := 5
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				svc.Close() // Should not panic
			}()
		}

		wg.Wait()

		// Verify final state
		assert.False(t, svc.isStarted)
		assert.Nil(t, svc.closeFunc)

		// Clean up
		close(eventsCh)
	})
}
