package chainstream

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gabapcia/blockwatch/internal/pkg/logger"
	retrytest "github.com/gabapcia/blockwatch/internal/pkg/resilience/retry/mocks"
	"github.com/gabapcia/blockwatch/internal/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	// Initialize logger for tests to prevent nil pointer dereference
	_ = logger.Init("error")
}

func TestService_Start(t *testing.T) {
	t.Run("successful start with no networks", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := New(map[string]Blockchain{}, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// Start the service
		outputCh, err := svc.Start(ctx)

		// Verify success
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Clean up
		svc.Close()
	})

	t.Run("successful start with single network", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockBlockchain := NewBlockchainMock(t)
		networks := map[string]Blockchain{
			"ethereum": mockBlockchain,
		}
		svc := New(networks, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// Setup expectations
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").Return(types.Hex(""), ErrNoCheckpointFound).Once()

		eventsCh := make(chan BlockchainEvent, 1)
		mockBlockchain.EXPECT().Subscribe(mock.Anything, types.Hex("")).Return((<-chan BlockchainEvent)(eventsCh), nil).Once()

		// Start the service
		outputCh, err := svc.Start(ctx)

		// Verify success
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Clean up
		close(eventsCh)
		svc.Close()
	})

	t.Run("successful start with existing checkpoint", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockBlockchain := NewBlockchainMock(t)
		networks := map[string]Blockchain{
			"ethereum": mockBlockchain,
		}
		svc := New(networks, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// Setup expectations - checkpoint exists, so should start from next height
		lastCheckpoint := types.Hex("0x100")
		expectedStartHeight := types.Hex("0x101")
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").Return(lastCheckpoint, nil).Once()

		eventsCh := make(chan BlockchainEvent, 1)
		mockBlockchain.EXPECT().Subscribe(mock.Anything, expectedStartHeight).Return((<-chan BlockchainEvent)(eventsCh), nil).Once()

		// Start the service
		outputCh, err := svc.Start(ctx)

		// Verify success
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Clean up
		close(eventsCh)
		svc.Close()
	})

	t.Run("returns error when already started", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := New(map[string]Blockchain{}, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// Start the service first time
		outputCh1, err1 := svc.Start(ctx)
		assert.NoError(t, err1)
		assert.NotNil(t, outputCh1)

		// Try to start again
		outputCh2, err2 := svc.Start(ctx)

		// Verify error
		assert.Error(t, err2)
		assert.Equal(t, ErrServiceAlreadyStarted, err2)
		assert.Nil(t, outputCh2)

		// Clean up
		svc.Close()
	})

	t.Run("returns error when checkpoint load fails", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockBlockchain := NewBlockchainMock(t)
		networks := map[string]Blockchain{
			"ethereum": mockBlockchain,
		}
		svc := New(networks, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// Setup expectations - checkpoint load fails
		checkpointError := errors.New("database connection failed")
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").Return(types.Hex(""), checkpointError).Once()

		// Start the service
		outputCh, err := svc.Start(ctx)

		// Verify error
		assert.Error(t, err)
		assert.Equal(t, checkpointError, err)
		assert.Nil(t, outputCh)
	})

	t.Run("returns error when subscription fails", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockBlockchain := NewBlockchainMock(t)
		networks := map[string]Blockchain{
			"ethereum": mockBlockchain,
		}
		svc := New(networks, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// Setup expectations
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").Return(types.Hex(""), ErrNoCheckpointFound).Once()

		subscriptionError := errors.New("network connection failed")
		mockBlockchain.EXPECT().Subscribe(mock.Anything, types.Hex("")).Return((<-chan BlockchainEvent)(nil), subscriptionError).Once()

		// Start the service
		outputCh, err := svc.Start(ctx)

		// Verify error
		assert.Error(t, err)
		assert.Equal(t, subscriptionError, err)
		assert.Nil(t, outputCh)
	})

	t.Run("successful start with multiple networks", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockEthereum := NewBlockchainMock(t)
		mockPolygon := NewBlockchainMock(t)
		networks := map[string]Blockchain{
			"ethereum": mockEthereum,
			"polygon":  mockPolygon,
		}
		svc := New(networks, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// Setup expectations for both networks
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").Return(types.Hex(""), ErrNoCheckpointFound).Once()
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "polygon").Return(types.Hex(""), ErrNoCheckpointFound).Once()

		ethEventsCh := make(chan BlockchainEvent, 1)
		polyEventsCh := make(chan BlockchainEvent, 1)
		mockEthereum.EXPECT().Subscribe(mock.Anything, types.Hex("")).Return((<-chan BlockchainEvent)(ethEventsCh), nil).Once()
		mockPolygon.EXPECT().Subscribe(mock.Anything, types.Hex("")).Return((<-chan BlockchainEvent)(polyEventsCh), nil).Once()

		// Start the service
		outputCh, err := svc.Start(ctx)

		// Verify success
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Clean up
		close(ethEventsCh)
		close(polyEventsCh)
		svc.Close()
	})

	t.Run("processes blocks through the pipeline", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockBlockchain := NewBlockchainMock(t)
		networks := map[string]Blockchain{
			"ethereum": mockBlockchain,
		}
		svc := New(networks, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// Setup expectations
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").Return(types.Hex(""), ErrNoCheckpointFound).Once()

		eventsCh := make(chan BlockchainEvent, 1)
		mockBlockchain.EXPECT().Subscribe(mock.Anything, types.Hex("")).Return((<-chan BlockchainEvent)(eventsCh), nil).Once()

		// Test block
		testBlock := Block{
			Height: types.Hex("0x100"),
			Hash:   "hash1",
			Transactions: []Transaction{
				{Hash: "tx1", From: "addr1", To: "addr2"},
			},
		}

		// Expect checkpoint to be saved
		mockStorage.EXPECT().SaveCheckpoint(mock.Anything, "ethereum", testBlock.Height).Return(nil).Once()

		// Start the service
		outputCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Send a block event
		eventsCh <- BlockchainEvent{
			Height: testBlock.Height,
			Block:  testBlock,
			Err:    nil,
		}

		// Verify block is received on output channel
		select {
		case observedBlock := <-outputCh:
			assert.Equal(t, "ethereum", observedBlock.Network)
			assert.Equal(t, testBlock, observedBlock.Block)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected to receive block on output channel")
		}

		// Clean up
		close(eventsCh)
		svc.Close()
	})

	t.Run("close cleans up properly after start", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := New(map[string]Blockchain{}, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// Start the service
		outputCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Close the service
		svc.Close()

		// Verify output channel is closed
		select {
		case _, ok := <-outputCh:
			assert.False(t, ok, "Output channel should be closed")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected output channel to be closed")
		}

		// Verify service can be started again after close
		outputCh2, err2 := svc.Start(ctx)
		assert.NoError(t, err2)
		assert.NotNil(t, outputCh2)

		// Clean up
		svc.Close()
	})
}

func TestService_Close(t *testing.T) {
	t.Run("close without start is safe", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := New(map[string]Blockchain{}, WithCheckpointStorage(mockStorage))

		// Close without starting should not panic or cause issues
		assert.NotPanics(t, func() {
			svc.Close()
		})

		// Should be able to start after close
		ctx := t.Context()
		outputCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Clean up
		svc.Close()
	})

	t.Run("close after start cleans up resources", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockBlockchain := NewBlockchainMock(t)
		networks := map[string]Blockchain{
			"ethereum": mockBlockchain,
		}
		svc := New(networks, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// Setup expectations
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").Return(types.Hex(""), ErrNoCheckpointFound).Once()

		eventsCh := make(chan BlockchainEvent, 1)
		mockBlockchain.EXPECT().Subscribe(mock.Anything, types.Hex("")).Return((<-chan BlockchainEvent)(eventsCh), nil).Once()

		// Start the service
		outputCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Close the service
		svc.Close()

		// Verify output channel is closed
		select {
		case _, ok := <-outputCh:
			assert.False(t, ok, "Output channel should be closed")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected output channel to be closed")
		}

		// Clean up
		close(eventsCh)
	})

	t.Run("close cancels context and stops goroutines", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockBlockchain := NewBlockchainMock(t)
		networks := map[string]Blockchain{
			"ethereum": mockBlockchain,
		}
		svc := New(networks, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// Setup expectations
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").Return(types.Hex(""), ErrNoCheckpointFound).Once()

		eventsCh := make(chan BlockchainEvent, 1)
		mockBlockchain.EXPECT().Subscribe(mock.Anything, types.Hex("")).Return((<-chan BlockchainEvent)(eventsCh), nil).Once()

		// Start the service
		outputCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Send a test block to verify processing is working
		testBlock := Block{
			Height: types.Hex("0x100"),
			Hash:   "hash1",
		}

		// Expect checkpoint to be saved
		mockStorage.EXPECT().SaveCheckpoint(mock.Anything, "ethereum", testBlock.Height).Return(nil).Once()

		// Send block event
		eventsCh <- BlockchainEvent{
			Height: testBlock.Height,
			Block:  testBlock,
			Err:    nil,
		}

		// Verify block is processed
		select {
		case observedBlock := <-outputCh:
			assert.Equal(t, "ethereum", observedBlock.Network)
			assert.Equal(t, testBlock, observedBlock.Block)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected to receive block on output channel")
		}

		// Close the service
		svc.Close()

		// Send another block after close - should not be processed
		testBlock2 := Block{
			Height: types.Hex("0x101"),
			Hash:   "hash2",
		}

		eventsCh <- BlockchainEvent{
			Height: testBlock2.Height,
			Block:  testBlock2,
			Err:    nil,
		}

		// Verify no more blocks are processed after close
		select {
		case _, ok := <-outputCh:
			if ok {
				t.Fatal("Should not receive blocks after close")
			}
			// Channel closed is expected
		case <-time.After(50 * time.Millisecond):
			// Timeout is also acceptable - no more processing
		}

		// Clean up
		close(eventsCh)
	})

	t.Run("multiple close calls are safe", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := New(map[string]Blockchain{}, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// Start the service
		outputCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Multiple close calls should not panic
		assert.NotPanics(t, func() {
			svc.Close()
			svc.Close()
			svc.Close()
		})

		// Verify output channel is closed
		select {
		case _, ok := <-outputCh:
			assert.False(t, ok, "Output channel should be closed")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected output channel to be closed")
		}
	})

	t.Run("close allows service restart", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockBlockchain := NewBlockchainMock(t)
		networks := map[string]Blockchain{
			"ethereum": mockBlockchain,
		}
		svc := New(networks, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// First start
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").Return(types.Hex(""), ErrNoCheckpointFound).Once()
		eventsCh1 := make(chan BlockchainEvent, 1)
		mockBlockchain.EXPECT().Subscribe(mock.Anything, types.Hex("")).Return((<-chan BlockchainEvent)(eventsCh1), nil).Once()

		outputCh1, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, outputCh1)

		// Close
		svc.Close()

		// Verify first channel is closed
		select {
		case _, ok := <-outputCh1:
			assert.False(t, ok, "First output channel should be closed")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected first output channel to be closed")
		}

		// Second start should work
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").Return(types.Hex(""), ErrNoCheckpointFound).Once()
		eventsCh2 := make(chan BlockchainEvent, 1)
		mockBlockchain.EXPECT().Subscribe(mock.Anything, types.Hex("")).Return((<-chan BlockchainEvent)(eventsCh2), nil).Once()

		outputCh2, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, outputCh2)

		// Verify second instance works
		testBlock := Block{
			Height: types.Hex("0x100"),
			Hash:   "hash1",
		}

		mockStorage.EXPECT().SaveCheckpoint(mock.Anything, "ethereum", testBlock.Height).Return(nil).Once()

		eventsCh2 <- BlockchainEvent{
			Height: testBlock.Height,
			Block:  testBlock,
			Err:    nil,
		}

		select {
		case observedBlock := <-outputCh2:
			assert.Equal(t, "ethereum", observedBlock.Network)
			assert.Equal(t, testBlock, observedBlock.Block)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected to receive block on second output channel")
		}

		// Clean up
		close(eventsCh1)
		close(eventsCh2)
		svc.Close()
	})

	t.Run("close with multiple networks", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockEthereum := NewBlockchainMock(t)
		mockPolygon := NewBlockchainMock(t)
		networks := map[string]Blockchain{
			"ethereum": mockEthereum,
			"polygon":  mockPolygon,
		}
		svc := New(networks, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// Setup expectations for both networks
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").Return(types.Hex(""), ErrNoCheckpointFound).Once()
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "polygon").Return(types.Hex(""), ErrNoCheckpointFound).Once()

		ethEventsCh := make(chan BlockchainEvent, 1)
		polyEventsCh := make(chan BlockchainEvent, 1)
		mockEthereum.EXPECT().Subscribe(mock.Anything, types.Hex("")).Return((<-chan BlockchainEvent)(ethEventsCh), nil).Once()
		mockPolygon.EXPECT().Subscribe(mock.Anything, types.Hex("")).Return((<-chan BlockchainEvent)(polyEventsCh), nil).Once()

		// Start the service
		outputCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Send blocks from both networks
		ethBlock := Block{Height: types.Hex("0x100"), Hash: "eth_hash"}
		polyBlock := Block{Height: types.Hex("0x200"), Hash: "poly_hash"}

		mockStorage.EXPECT().SaveCheckpoint(mock.Anything, "ethereum", ethBlock.Height).Return(nil).Once()
		mockStorage.EXPECT().SaveCheckpoint(mock.Anything, "polygon", polyBlock.Height).Return(nil).Once()

		ethEventsCh <- BlockchainEvent{Height: ethBlock.Height, Block: ethBlock, Err: nil}
		polyEventsCh <- BlockchainEvent{Height: polyBlock.Height, Block: polyBlock, Err: nil}

		// Verify both blocks are processed
		receivedBlocks := make(map[string]Block)
		for i := 0; i < 2; i++ {
			select {
			case observedBlock := <-outputCh:
				receivedBlocks[observedBlock.Network] = observedBlock.Block
			case <-time.After(100 * time.Millisecond):
				t.Fatal("Expected to receive blocks from both networks")
			}
		}

		assert.Equal(t, ethBlock, receivedBlocks["ethereum"])
		assert.Equal(t, polyBlock, receivedBlocks["polygon"])

		// Close the service
		svc.Close()

		// Verify output channel is closed
		select {
		case _, ok := <-outputCh:
			assert.False(t, ok, "Output channel should be closed")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected output channel to be closed")
		}

		// Clean up
		close(ethEventsCh)
		close(polyEventsCh)
	})

	t.Run("close is thread-safe", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := New(map[string]Blockchain{}, WithCheckpointStorage(mockStorage))

		ctx := t.Context()

		// Start the service
		outputCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Call close from multiple goroutines concurrently
		const numGoroutines = 10
		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				svc.Close()
			}()
		}

		// Wait for all goroutines to complete
		wg.Wait()

		// Verify output channel is closed
		select {
		case _, ok := <-outputCh:
			assert.False(t, ok, "Output channel should be closed")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected output channel to be closed")
		}

	})

	t.Run("close with retry enabled", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockRetry := retrytest.NewRetry(t)
		mockBlockchain := NewBlockchainMock(t)
		networks := map[string]Blockchain{
			"ethereum": mockBlockchain,
		}
		svc := New(networks, WithCheckpointStorage(mockStorage), WithRetry(mockRetry))

		ctx := t.Context()

		// Setup expectations
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").Return(types.Hex(""), ErrNoCheckpointFound).Once()

		eventsCh := make(chan BlockchainEvent, 1)
		mockBlockchain.EXPECT().Subscribe(mock.Anything, types.Hex("")).Return((<-chan BlockchainEvent)(eventsCh), nil).Once()

		// Start the service
		outputCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Close the service
		svc.Close()

		// Verify output channel is closed
		select {
		case _, ok := <-outputCh:
			assert.False(t, ok, "Output channel should be closed")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected output channel to be closed")
		}

		// Clean up
		close(eventsCh)
	})

	t.Run("close with custom dispatch failure handler", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		handlerCalled := false
		customHandler := func(ctx context.Context, failure BlockDispatchFailure) {
			handlerCalled = true
		}

		svc := New(map[string]Blockchain{}, WithCheckpointStorage(mockStorage), WithDispatchFailureHandler(customHandler))

		ctx := t.Context()

		// Start the service
		outputCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Close the service
		svc.Close()

		// Verify output channel is closed
		select {
		case _, ok := <-outputCh:
			assert.False(t, ok, "Output channel should be closed")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Expected output channel to be closed")
		}

		// Handler should not have been called during normal close
		assert.False(t, handlerCalled)
	})
}

func TestNew(t *testing.T) {
	t.Run("creates service with default configuration", func(t *testing.T) {
		// Setup
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
			"polygon":  NewBlockchainMock(t),
		}

		// Create service with no options
		svc := New(networks)

		// Verify service is created
		assert.NotNil(t, svc)
		assert.IsType(t, &service{}, svc)

		// Verify internal state
		assert.Equal(t, networks, svc.networks)
		assert.NotNil(t, svc.checkpointStorage)
		assert.IsType(t, nopCheckpoint{}, svc.checkpointStorage)
		assert.Nil(t, svc.retry)
		assert.NotNil(t, svc.dispatchFailureHandler)
		assert.False(t, svc.isStarted)
		assert.Nil(t, svc.closeFunc)
	})

	t.Run("creates service with empty networks map", func(t *testing.T) {
		// Create service with empty networks
		svc := New(map[string]Blockchain{})

		// Verify service is created
		assert.NotNil(t, svc)
		assert.Equal(t, map[string]Blockchain{}, svc.networks)
		assert.NotNil(t, svc.checkpointStorage)
		assert.IsType(t, nopCheckpoint{}, svc.checkpointStorage)
	})

	t.Run("creates service with nil networks map", func(t *testing.T) {
		// Create service with nil networks
		svc := New(nil)

		// Verify service is created
		assert.NotNil(t, svc)
		assert.Nil(t, svc.networks)
		assert.NotNil(t, svc.checkpointStorage)
		assert.IsType(t, nopCheckpoint{}, svc.checkpointStorage)
	})

	t.Run("creates service with single network", func(t *testing.T) {
		// Setup
		mockBlockchain := NewBlockchainMock(t)
		networks := map[string]Blockchain{
			"ethereum": mockBlockchain,
		}

		// Create service
		svc := New(networks)

		// Verify service is created with correct network
		assert.NotNil(t, svc)
		assert.Equal(t, networks, svc.networks)
		assert.Equal(t, mockBlockchain, svc.networks["ethereum"])
	})

	t.Run("creates service with multiple networks", func(t *testing.T) {
		// Setup
		mockEthereum := NewBlockchainMock(t)
		mockPolygon := NewBlockchainMock(t)
		mockArbitrum := NewBlockchainMock(t)
		networks := map[string]Blockchain{
			"ethereum": mockEthereum,
			"polygon":  mockPolygon,
			"arbitrum": mockArbitrum,
		}

		// Create service
		svc := New(networks)

		// Verify service is created with all networks
		assert.NotNil(t, svc)
		assert.Equal(t, networks, svc.networks)
		assert.Len(t, svc.networks, 3)
		assert.Equal(t, mockEthereum, svc.networks["ethereum"])
		assert.Equal(t, mockPolygon, svc.networks["polygon"])
		assert.Equal(t, mockArbitrum, svc.networks["arbitrum"])
	})

	t.Run("default dispatch failure handler logs errors", func(t *testing.T) {
		// Setup
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with default handler
		svc := New(networks)

		// Test the default handler (should not panic)
		ctx := context.Background()
		failure := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x100"),
			Errors:  []error{errors.New("test error")},
		}

		// Should not panic when called
		assert.NotPanics(t, func() {
			svc.dispatchFailureHandler(ctx, failure)
		})
	})

	t.Run("service is not started initially", func(t *testing.T) {
		// Setup
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service
		svc := New(networks)

		// Verify initial state
		assert.False(t, svc.isStarted)
		assert.Nil(t, svc.closeFunc)
	})

	t.Run("service can be started after creation", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockBlockchain := NewBlockchainMock(t)
		networks := map[string]Blockchain{
			"ethereum": mockBlockchain,
		}

		// Create service
		svc := New(networks, WithCheckpointStorage(mockStorage))

		// Setup expectations
		mockStorage.EXPECT().LoadLatestCheckpoint(mock.Anything, "ethereum").Return(types.Hex(""), ErrNoCheckpointFound).Once()
		eventsCh := make(chan BlockchainEvent, 1)
		mockBlockchain.EXPECT().Subscribe(mock.Anything, types.Hex("")).Return((<-chan BlockchainEvent)(eventsCh), nil).Once()

		// Start should work
		ctx := context.Background()
		outputCh, err := svc.Start(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, outputCh)

		// Clean up
		close(eventsCh)
		svc.Close()
	})
}

func TestWithCheckpointStorage(t *testing.T) {
	t.Run("sets custom checkpoint storage", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with custom checkpoint storage
		svc := New(networks, WithCheckpointStorage(mockStorage))

		// Verify checkpoint storage is set
		assert.Equal(t, mockStorage, svc.checkpointStorage)
	})

	t.Run("overrides default nop checkpoint storage", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with custom checkpoint storage
		svc := New(networks, WithCheckpointStorage(mockStorage))

		// Verify it's not the default nop storage
		assert.NotEqual(t, nopCheckpoint{}, svc.checkpointStorage)
		assert.Equal(t, mockStorage, svc.checkpointStorage)
	})

	t.Run("can be set to nil", func(t *testing.T) {
		// Setup
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with nil checkpoint storage
		svc := New(networks, WithCheckpointStorage(nil))

		// Verify checkpoint storage is nil
		assert.Nil(t, svc.checkpointStorage)
	})

	t.Run("last option wins when multiple checkpoint storages provided", func(t *testing.T) {
		// Setup
		mockStorage1 := NewCheckpointStorageMock(t)
		mockStorage2 := NewCheckpointStorageMock(t)
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with multiple checkpoint storage options
		svc := New(networks,
			WithCheckpointStorage(mockStorage1),
			WithCheckpointStorage(mockStorage2),
		)

		// Verify last option is used
		assert.Equal(t, mockStorage2, svc.checkpointStorage)
		// Verify it's not the first storage by checking they're different pointers
		assert.True(t, mockStorage1 != mockStorage2, "Should use the last provided storage")
	})

	t.Run("works with other options", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockRetry := retrytest.NewRetry(t)
		customHandler := func(ctx context.Context, failure BlockDispatchFailure) {}
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with multiple options
		svc := New(networks,
			WithCheckpointStorage(mockStorage),
			WithRetry(mockRetry),
			WithDispatchFailureHandler(customHandler),
		)

		// Verify all options are applied
		assert.Equal(t, mockStorage, svc.checkpointStorage)
		assert.Equal(t, mockRetry, svc.retry)
		assert.NotNil(t, svc.dispatchFailureHandler)
	})
}

func TestWithRetry(t *testing.T) {
	t.Run("sets custom retry mechanism", func(t *testing.T) {
		// Setup
		mockRetry := retrytest.NewRetry(t)
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with custom retry
		svc := New(networks, WithRetry(mockRetry))

		// Verify retry is set
		assert.Equal(t, mockRetry, svc.retry)
	})

	t.Run("overrides default nil retry", func(t *testing.T) {
		// Setup
		mockRetry := retrytest.NewRetry(t)
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with custom retry
		svc := New(networks, WithRetry(mockRetry))

		// Verify it's not nil
		assert.NotNil(t, svc.retry)
		assert.Equal(t, mockRetry, svc.retry)
	})

	t.Run("can be set to nil", func(t *testing.T) {
		// Setup
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with nil retry
		svc := New(networks, WithRetry(nil))

		// Verify retry is nil
		assert.Nil(t, svc.retry)
	})

	t.Run("last option wins when multiple retries provided", func(t *testing.T) {
		// Setup
		mockRetry1 := retrytest.NewRetry(t)
		mockRetry2 := retrytest.NewRetry(t)
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with multiple retry options
		svc := New(networks,
			WithRetry(mockRetry1),
			WithRetry(mockRetry2),
		)

		// Verify last option is used
		assert.Equal(t, mockRetry2, svc.retry)
		// Verify it's not the first retry by checking they're different pointers
		assert.True(t, mockRetry1 != mockRetry2, "Should use the last provided retry")
	})

	t.Run("works with other options", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockRetry := retrytest.NewRetry(t)
		customHandler := func(ctx context.Context, failure BlockDispatchFailure) {}
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with multiple options
		svc := New(networks,
			WithRetry(mockRetry),
			WithCheckpointStorage(mockStorage),
			WithDispatchFailureHandler(customHandler),
		)

		// Verify all options are applied
		assert.Equal(t, mockRetry, svc.retry)
		assert.Equal(t, mockStorage, svc.checkpointStorage)
		assert.NotNil(t, svc.dispatchFailureHandler)
	})
}

func TestWithDispatchFailureHandler(t *testing.T) {
	t.Run("sets custom dispatch failure handler", func(t *testing.T) {
		// Setup
		handlerCalled := false
		customHandler := func(ctx context.Context, failure BlockDispatchFailure) {
			handlerCalled = true
		}
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with custom handler
		svc := New(networks, WithDispatchFailureHandler(customHandler))

		// Verify handler is set and works
		ctx := context.Background()
		failure := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x100"),
			Errors:  []error{errors.New("test error")},
		}

		svc.dispatchFailureHandler(ctx, failure)
		assert.True(t, handlerCalled)
	})

	t.Run("overrides default handler", func(t *testing.T) {
		// Setup
		customHandlerCalled := false
		customHandler := func(ctx context.Context, failure BlockDispatchFailure) {
			customHandlerCalled = true
		}
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with custom handler
		svc := New(networks, WithDispatchFailureHandler(customHandler))

		// Test that custom handler is called, not default
		ctx := context.Background()
		failure := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x100"),
			Errors:  []error{errors.New("test error")},
		}

		svc.dispatchFailureHandler(ctx, failure)
		assert.True(t, customHandlerCalled)
	})

	t.Run("can be set to nil", func(t *testing.T) {
		// Setup
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with nil handler
		svc := New(networks, WithDispatchFailureHandler(nil))

		// Verify handler is nil
		assert.Nil(t, svc.dispatchFailureHandler)
	})

	t.Run("handler receives correct failure information", func(t *testing.T) {
		// Setup
		var receivedFailure BlockDispatchFailure
		customHandler := func(ctx context.Context, failure BlockDispatchFailure) {
			receivedFailure = failure
		}
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with custom handler
		svc := New(networks, WithDispatchFailureHandler(customHandler))

		// Test handler receives correct data
		ctx := context.Background()
		expectedFailure := BlockDispatchFailure{
			Network: "polygon",
			Height:  types.Hex("0x200"),
			Errors:  []error{errors.New("network error"), errors.New("timeout error")},
		}

		svc.dispatchFailureHandler(ctx, expectedFailure)
		assert.Equal(t, expectedFailure, receivedFailure)
	})

	t.Run("last option wins when multiple handlers provided", func(t *testing.T) {
		// Setup
		handler1Called := false
		handler2Called := false
		handler1 := func(ctx context.Context, failure BlockDispatchFailure) {
			handler1Called = true
		}
		handler2 := func(ctx context.Context, failure BlockDispatchFailure) {
			handler2Called = true
		}
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with multiple handlers
		svc := New(networks,
			WithDispatchFailureHandler(handler1),
			WithDispatchFailureHandler(handler2),
		)

		// Test that only the last handler is called
		ctx := context.Background()
		failure := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x100"),
			Errors:  []error{errors.New("test error")},
		}

		svc.dispatchFailureHandler(ctx, failure)
		assert.False(t, handler1Called)
		assert.True(t, handler2Called)
	})

	t.Run("works with other options", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		mockRetry := retrytest.NewRetry(t)
		handlerCalled := false
		customHandler := func(ctx context.Context, failure BlockDispatchFailure) {
			handlerCalled = true
		}
		networks := map[string]Blockchain{
			"ethereum": NewBlockchainMock(t),
		}

		// Create service with multiple options
		svc := New(networks,
			WithDispatchFailureHandler(customHandler),
			WithCheckpointStorage(mockStorage),
			WithRetry(mockRetry),
		)

		// Verify all options are applied
		assert.Equal(t, mockStorage, svc.checkpointStorage)
		assert.Equal(t, mockRetry, svc.retry)

		// Test handler works
		ctx := context.Background()
		failure := BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x100"),
			Errors:  []error{errors.New("test error")},
		}

		svc.dispatchFailureHandler(ctx, failure)
		assert.True(t, handlerCalled)
	})
}
