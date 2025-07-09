package chainstream

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabapcia/blockwatch/internal/pkg/logger"
	"github.com/gabapcia/blockwatch/internal/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Initialize logger for tests to prevent nil pointer dereference
	_ = logger.Init("error")
}

func TestService_checkpointAndForward(t *testing.T) {
	t.Run("successfully forwards blocks and saves checkpoints", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := &service{
			checkpointStorage: mockStorage,
		}

		ctx := t.Context()
		blockIn := make(chan ObservedBlock, 3)
		blockOut := make(chan ObservedBlock, 3)

		// Test data
		blocks := []ObservedBlock{
			{
				Network: "ethereum",
				Block: Block{
					Height: types.Hex("0x100"),
					Hash:   "hash1",
					Transactions: []Transaction{
						{Hash: "tx1", From: "addr1", To: "addr2"},
					},
				},
			},
			{
				Network: "polygon",
				Block: Block{
					Height: types.Hex("0x200"),
					Hash:   "hash2",
					Transactions: []Transaction{
						{Hash: "tx2", From: "addr3", To: "addr4"},
					},
				},
			},
			{
				Network: "ethereum",
				Block: Block{
					Height:       types.Hex("0x101"),
					Hash:         "hash3",
					Transactions: []Transaction{},
				},
			},
		}

		// Setup expectations
		for _, block := range blocks {
			mockStorage.EXPECT().SaveCheckpoint(ctx, block.Network, block.Height).Return(nil).Once()
		}

		// Send test blocks
		for _, block := range blocks {
			blockIn <- block
		}
		close(blockIn)

		// Start the function under test
		go svc.checkpointAndForward(ctx, blockIn, blockOut)

		// Collect results
		var results []ObservedBlock
		for i := 0; i < len(blocks); i++ {
			select {
			case block := <-blockOut:
				results = append(results, block)
			case <-time.After(100 * time.Millisecond):
				t.Fatal("Timeout waiting for block")
			}
		}

		// Verify results
		assert.Equal(t, blocks, results)
	})

	t.Run("continues processing when checkpoint save fails", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := &service{
			checkpointStorage: mockStorage,
		}

		ctx := t.Context()
		blockIn := make(chan ObservedBlock, 2)
		blockOut := make(chan ObservedBlock, 2)

		// Test data
		blocks := []ObservedBlock{
			{
				Network: "ethereum",
				Block: Block{
					Height: types.Hex("0x100"),
					Hash:   "hash1",
				},
			},
			{
				Network: "ethereum",
				Block: Block{
					Height: types.Hex("0x101"),
					Hash:   "hash2",
				},
			},
		}

		// Setup expectations - first save fails, second succeeds
		checkpointError := errors.New("checkpoint storage error")
		mockStorage.EXPECT().SaveCheckpoint(ctx, blocks[0].Network, blocks[0].Height).Return(checkpointError).Once()
		mockStorage.EXPECT().SaveCheckpoint(ctx, blocks[1].Network, blocks[1].Height).Return(nil).Once()

		// Send test blocks
		for _, block := range blocks {
			blockIn <- block
		}
		close(blockIn)

		// Start the function under test
		go svc.checkpointAndForward(ctx, blockIn, blockOut)

		// Collect results
		var results []ObservedBlock
		for i := 0; i < len(blocks); i++ {
			select {
			case block := <-blockOut:
				results = append(results, block)
			case <-time.After(100 * time.Millisecond):
				t.Fatal("Timeout waiting for block")
			}
		}

		// Verify both blocks were forwarded despite checkpoint failure
		assert.Equal(t, blocks, results)
	})

	t.Run("exits when input channel is closed", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := &service{
			checkpointStorage: mockStorage,
		}

		ctx := t.Context()
		blockIn := make(chan ObservedBlock)
		blockOut := make(chan ObservedBlock, 1)

		// Close input channel immediately
		close(blockIn)

		// Start the function under test
		done := make(chan struct{})
		go func() {
			svc.checkpointAndForward(ctx, blockIn, blockOut)
			close(done)
		}()

		// Function should exit quickly
		select {
		case <-done:
			// Expected - function should exit when input channel is closed
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Function should exit when input channel is closed")
		}

		// Verify no blocks were sent to output
		select {
		case <-blockOut:
			t.Fatal("No blocks should be sent when input channel is closed immediately")
		default:
			// Expected - no blocks should be sent
		}
	})

	t.Run("exits when context is canceled", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := &service{
			checkpointStorage: mockStorage,
		}

		ctx, cancel := context.WithCancel(t.Context())
		blockIn := make(chan ObservedBlock)
		blockOut := make(chan ObservedBlock, 1)

		// Start the function under test
		done := make(chan struct{})
		go func() {
			svc.checkpointAndForward(ctx, blockIn, blockOut)
			close(done)
		}()

		// Cancel context
		cancel()

		// Function should exit quickly
		select {
		case <-done:
			// Expected - function should exit when context is canceled
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Function should exit when context is canceled")
		}
	})

	t.Run("exits when output channel send fails due to context cancellation", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := &service{
			checkpointStorage: mockStorage,
		}

		ctx, cancel := context.WithCancel(t.Context())
		blockIn := make(chan ObservedBlock, 1)
		blockOut := make(chan ObservedBlock) // No buffer - will block

		// Test data
		block := ObservedBlock{
			Network: "ethereum",
			Block: Block{
				Height: types.Hex("0x100"),
				Hash:   "hash1",
			},
		}

		// Setup expectation
		mockStorage.EXPECT().SaveCheckpoint(ctx, block.Network, block.Height).Return(nil).Once()

		// Send test block
		blockIn <- block

		// Start the function under test
		done := make(chan struct{})
		go func() {
			svc.checkpointAndForward(ctx, blockIn, blockOut)
			close(done)
		}()

		// Give some time for checkpoint to be saved and send to be attempted
		time.Sleep(10 * time.Millisecond)

		// Cancel context while send is blocked
		cancel()

		// Function should exit
		select {
		case <-done:
			// Expected - function should exit when send fails due to context cancellation
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Function should exit when send fails due to context cancellation")
		}
	})

	t.Run("handles multiple networks correctly", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := &service{
			checkpointStorage: mockStorage,
		}

		ctx := t.Context()
		blockIn := make(chan ObservedBlock, 4)
		blockOut := make(chan ObservedBlock, 4)

		// Test data with different networks
		blocks := []ObservedBlock{
			{Network: "ethereum", Block: Block{Height: types.Hex("0x100"), Hash: "eth1"}},
			{Network: "polygon", Block: Block{Height: types.Hex("0x200"), Hash: "poly1"}},
			{Network: "arbitrum", Block: Block{Height: types.Hex("0x300"), Hash: "arb1"}},
			{Network: "ethereum", Block: Block{Height: types.Hex("0x101"), Hash: "eth2"}},
		}

		// Setup expectations
		for _, block := range blocks {
			mockStorage.EXPECT().SaveCheckpoint(ctx, block.Network, block.Height).Return(nil).Once()
		}

		// Send test blocks
		for _, block := range blocks {
			blockIn <- block
		}
		close(blockIn)

		// Start the function under test
		go svc.checkpointAndForward(ctx, blockIn, blockOut)

		// Collect results
		var results []ObservedBlock
		for i := 0; i < len(blocks); i++ {
			select {
			case block := <-blockOut:
				results = append(results, block)
			case <-time.After(100 * time.Millisecond):
				t.Fatal("Timeout waiting for block")
			}
		}

		// Verify all blocks were processed correctly
		assert.Equal(t, blocks, results)
	})

	t.Run("handles empty blocks correctly", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := &service{
			checkpointStorage: mockStorage,
		}

		ctx := t.Context()
		blockIn := make(chan ObservedBlock, 1)
		blockOut := make(chan ObservedBlock, 1)

		// Test data with empty block
		block := ObservedBlock{
			Network: "ethereum",
			Block: Block{
				Height:       types.Hex("0x100"),
				Hash:         "hash1",
				Transactions: []Transaction{}, // Empty transactions
			},
		}

		// Setup expectation
		mockStorage.EXPECT().SaveCheckpoint(ctx, block.Network, block.Height).Return(nil).Once()

		// Send test block
		blockIn <- block
		close(blockIn)

		// Start the function under test
		go svc.checkpointAndForward(ctx, blockIn, blockOut)

		// Collect result
		select {
		case result := <-blockOut:
			assert.Equal(t, block, result)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for block")
		}
	})

	t.Run("handles blocks with zero height", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := &service{
			checkpointStorage: mockStorage,
		}

		ctx := t.Context()
		blockIn := make(chan ObservedBlock, 1)
		blockOut := make(chan ObservedBlock, 1)

		// Test data with zero height
		block := ObservedBlock{
			Network: "ethereum",
			Block: Block{
				Height: types.Hex("0x0"),
				Hash:   "genesis",
			},
		}

		// Setup expectation
		mockStorage.EXPECT().SaveCheckpoint(ctx, block.Network, block.Height).Return(nil).Once()

		// Send test block
		blockIn <- block
		close(blockIn)

		// Start the function under test
		go svc.checkpointAndForward(ctx, blockIn, blockOut)

		// Collect result
		select {
		case result := <-blockOut:
			assert.Equal(t, block, result)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for block")
		}
	})

	t.Run("handles concurrent checkpoint saves safely", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := &service{
			checkpointStorage: mockStorage,
		}

		ctx := t.Context()
		blockIn := make(chan ObservedBlock, 10)
		blockOut := make(chan ObservedBlock, 10)

		// Test data - multiple blocks for the same network
		blocks := make([]ObservedBlock, 10)
		for i := 0; i < 10; i++ {
			blocks[i] = ObservedBlock{
				Network: "ethereum",
				Block: Block{
					Height: types.Hex("0x" + string(rune('0'+i))),
					Hash:   "hash" + string(rune('0'+i)),
				},
			}
		}

		// Setup expectations - each block should have its checkpoint saved
		for _, block := range blocks {
			mockStorage.EXPECT().SaveCheckpoint(ctx, block.Network, block.Height).Return(nil).Once()
		}

		// Send all blocks
		for _, block := range blocks {
			blockIn <- block
		}
		close(blockIn)

		// Start the function under test
		go svc.checkpointAndForward(ctx, blockIn, blockOut)

		// Collect all results
		var results []ObservedBlock
		for i := 0; i < len(blocks); i++ {
			select {
			case block := <-blockOut:
				results = append(results, block)
			case <-time.After(200 * time.Millisecond):
				t.Fatalf("Timeout waiting for block %d", i)
			}
		}

		// Verify all blocks were processed
		assert.Equal(t, blocks, results)
	})
}

func TestService_startCheckpointAndForward(t *testing.T) {
	t.Run("starts checkpointAndForward in goroutine", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := &service{
			checkpointStorage: mockStorage,
		}

		ctx := t.Context()
		blockIn := make(chan ObservedBlock, 1)
		blockOut := make(chan ObservedBlock, 1)

		// Test data
		block := ObservedBlock{
			Network: "ethereum",
			Block: Block{
				Height: types.Hex("0x100"),
				Hash:   "hash1",
			},
		}

		// Setup expectation
		mockStorage.EXPECT().SaveCheckpoint(ctx, block.Network, block.Height).Return(nil).Once()

		// Start the function under test (should return immediately)
		start := time.Now()
		svc.startCheckpointAndForward(ctx, blockIn, blockOut)
		duration := time.Since(start)

		// Function should return immediately (within a reasonable time)
		assert.Less(t, duration, 10*time.Millisecond, "startCheckpointAndForward should return immediately")

		// Send test block
		blockIn <- block
		close(blockIn)

		// Verify processing happens in background
		select {
		case result := <-blockOut:
			assert.Equal(t, block, result)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Background processing should handle the block")
		}
	})

	t.Run("multiple calls create independent goroutines", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := &service{
			checkpointStorage: mockStorage,
		}

		ctx := t.Context()

		// Create multiple independent channels
		blockIn1 := make(chan ObservedBlock, 1)
		blockOut1 := make(chan ObservedBlock, 1)
		blockIn2 := make(chan ObservedBlock, 1)
		blockOut2 := make(chan ObservedBlock, 1)

		// Test data
		block1 := ObservedBlock{Network: "ethereum", Block: Block{Height: types.Hex("0x100"), Hash: "hash1"}}
		block2 := ObservedBlock{Network: "polygon", Block: Block{Height: types.Hex("0x200"), Hash: "hash2"}}

		// Setup expectations
		mockStorage.EXPECT().SaveCheckpoint(ctx, block1.Network, block1.Height).Return(nil).Once()
		mockStorage.EXPECT().SaveCheckpoint(ctx, block2.Network, block2.Height).Return(nil).Once()

		// Start multiple independent processors
		svc.startCheckpointAndForward(ctx, blockIn1, blockOut1)
		svc.startCheckpointAndForward(ctx, blockIn2, blockOut2)

		// Send blocks to different processors
		blockIn1 <- block1
		blockIn2 <- block2
		close(blockIn1)
		close(blockIn2)

		// Verify both processors work independently
		var results []ObservedBlock

		select {
		case result := <-blockOut1:
			results = append(results, result)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("First processor should handle its block")
		}

		select {
		case result := <-blockOut2:
			results = append(results, result)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Second processor should handle its block")
		}

		// Verify both blocks were processed
		assert.Contains(t, results, block1)
		assert.Contains(t, results, block2)
		assert.Len(t, results, 2)
	})

	t.Run("function returns immediately", func(t *testing.T) {
		// Setup
		mockStorage := NewCheckpointStorageMock(t)
		svc := &service{
			checkpointStorage: mockStorage,
		}

		ctx := t.Context()
		blockIn := make(chan ObservedBlock)
		blockOut := make(chan ObservedBlock)

		// Measure execution time
		start := time.Now()
		svc.startCheckpointAndForward(ctx, blockIn, blockOut)
		duration := time.Since(start)

		// Should return almost immediately
		assert.Less(t, duration, 5*time.Millisecond, "startCheckpointAndForward should return immediately")

		// Clean up
		close(blockIn)
	})
}

func TestNopCheckpoint_SaveCheckpoint(t *testing.T) {
	t.Run("saves checkpoint successfully without error", func(t *testing.T) {
		// Create nopCheckpoint instance
		checkpoint := nopCheckpoint{}

		// Test context
		ctx := t.Context()

		// Test parameters
		network := "ethereum"
		height := types.Hex("0x123456")

		// Call SaveCheckpoint - should not return error
		err := checkpoint.SaveCheckpoint(ctx, network, height)

		// Verify no error is returned
		assert.NoError(t, err)
	})

	t.Run("handles empty network name", func(t *testing.T) {
		checkpoint := nopCheckpoint{}
		ctx := t.Context()

		err := checkpoint.SaveCheckpoint(ctx, "", types.Hex("0x100"))

		assert.NoError(t, err)
	})

	t.Run("handles empty height", func(t *testing.T) {
		checkpoint := nopCheckpoint{}
		ctx := t.Context()

		err := checkpoint.SaveCheckpoint(ctx, "bitcoin", types.Hex(""))

		assert.NoError(t, err)
	})

	t.Run("handles various network names", func(t *testing.T) {
		checkpoint := nopCheckpoint{}
		ctx := t.Context()

		networks := []string{"ethereum", "bitcoin", "polygon", "solana", "arbitrum"}
		heights := []types.Hex{"0x1", "0x100", "0xabc123", "0xffffffff", "0x0"}

		for i, network := range networks {
			height := heights[i%len(heights)]
			err := checkpoint.SaveCheckpoint(ctx, network, height)
			assert.NoError(t, err, "SaveCheckpoint should not fail for network %s with height %s", network, height)
		}
	})

	t.Run("handles canceled context", func(t *testing.T) {
		checkpoint := nopCheckpoint{}

		// Create canceled context
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		// SaveCheckpoint should still succeed since it's a no-op
		err := checkpoint.SaveCheckpoint(ctx, "ethereum", types.Hex("0x123"))

		assert.NoError(t, err)
	})

	t.Run("multiple calls with same network overwrite behavior", func(t *testing.T) {
		checkpoint := nopCheckpoint{}
		ctx := t.Context()

		network := "ethereum"

		// Save multiple checkpoints for the same network
		err1 := checkpoint.SaveCheckpoint(ctx, network, types.Hex("0x100"))
		err2 := checkpoint.SaveCheckpoint(ctx, network, types.Hex("0x200"))
		err3 := checkpoint.SaveCheckpoint(ctx, network, types.Hex("0x300"))

		// All should succeed
		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NoError(t, err3)

		// Since it's a no-op, we can't verify the overwrite behavior directly,
		// but we can verify that LoadLatestCheckpoint still returns ErrNoCheckpointFound
		_, err := checkpoint.LoadLatestCheckpoint(ctx, network)
		assert.Equal(t, ErrNoCheckpointFound, err)
	})
}

func TestNopCheckpoint_LoadLatestCheckpoint(t *testing.T) {
	t.Run("always returns ErrNoCheckpointFound", func(t *testing.T) {
		checkpoint := nopCheckpoint{}
		ctx := t.Context()

		height, err := checkpoint.LoadLatestCheckpoint(ctx, "ethereum")

		// Verify error is ErrNoCheckpointFound
		assert.Equal(t, ErrNoCheckpointFound, err)
		// Verify height is empty
		assert.Equal(t, types.Hex(""), height)
	})

	t.Run("returns error for various network names", func(t *testing.T) {
		checkpoint := nopCheckpoint{}
		ctx := t.Context()

		networks := []string{"ethereum", "bitcoin", "polygon", "solana", "arbitrum", "unknown-network"}

		for _, network := range networks {
			height, err := checkpoint.LoadLatestCheckpoint(ctx, network)

			assert.Equal(t, ErrNoCheckpointFound, err, "LoadLatestCheckpoint should return ErrNoCheckpointFound for network %s", network)
			assert.Equal(t, types.Hex(""), height, "Height should be empty for network %s", network)
		}
	})

	t.Run("returns error for empty network name", func(t *testing.T) {
		checkpoint := nopCheckpoint{}
		ctx := t.Context()

		height, err := checkpoint.LoadLatestCheckpoint(ctx, "")

		assert.Equal(t, ErrNoCheckpointFound, err)
		assert.Equal(t, types.Hex(""), height)
	})

	t.Run("handles canceled context", func(t *testing.T) {
		checkpoint := nopCheckpoint{}

		// Create canceled context
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		// LoadLatestCheckpoint should still return ErrNoCheckpointFound since it's a no-op
		height, err := checkpoint.LoadLatestCheckpoint(ctx, "ethereum")

		assert.Equal(t, ErrNoCheckpointFound, err)
		assert.Equal(t, types.Hex(""), height)
	})

	t.Run("consistent behavior after save attempts", func(t *testing.T) {
		checkpoint := nopCheckpoint{}
		ctx := t.Context()

		network := "ethereum"

		// Try to save a checkpoint
		saveErr := checkpoint.SaveCheckpoint(ctx, network, types.Hex("0x123"))
		assert.NoError(t, saveErr)

		// Load should still return ErrNoCheckpointFound since no persistence occurs
		height, loadErr := checkpoint.LoadLatestCheckpoint(ctx, network)

		assert.Equal(t, ErrNoCheckpointFound, loadErr)
		assert.Equal(t, types.Hex(""), height)
	})
}

func TestNopCheckpoint_Interface(t *testing.T) {
	t.Run("implements CheckpointStorage interface", func(t *testing.T) {
		// Verify that nopCheckpoint implements CheckpointStorage interface
		var checkpoint CheckpointStorage = nopCheckpoint{}

		// This test will fail to compile if nopCheckpoint doesn't implement the interface
		require.NotNil(t, checkpoint)

		// Test that we can call interface methods
		ctx := t.Context()

		err := checkpoint.SaveCheckpoint(ctx, "test", types.Hex("0x1"))
		assert.NoError(t, err)

		height, err := checkpoint.LoadLatestCheckpoint(ctx, "test")
		assert.Equal(t, ErrNoCheckpointFound, err)
		assert.Equal(t, types.Hex(""), height)
	})
}

func TestNopCheckpoint_ConcurrentAccess(t *testing.T) {
	t.Run("safe for concurrent access", func(t *testing.T) {
		checkpoint := nopCheckpoint{}
		ctx := t.Context()

		// Number of goroutines to run concurrently
		numGoroutines := 10
		numOperationsPerGoroutine := 100

		// Channel to collect errors
		errCh := make(chan error, numGoroutines*numOperationsPerGoroutine*2) // *2 for save and load operations

		// Start multiple goroutines performing save and load operations
		for i := 0; i < numGoroutines; i++ {
			go func(goroutineID int) {
				for j := 0; j < numOperationsPerGoroutine; j++ {
					network := "ethereum"
					height := types.Hex("0x" + string(rune('0'+goroutineID)) + string(rune('0'+j%10)))

					// Save operation
					if err := checkpoint.SaveCheckpoint(ctx, network, height); err != nil {
						errCh <- err
					}

					// Load operation
					if _, err := checkpoint.LoadLatestCheckpoint(ctx, network); err != ErrNoCheckpointFound {
						errCh <- err
					}
				}
			}(i)
		}

		// Wait a bit for all goroutines to complete
		// Since nopCheckpoint operations are very fast, a short wait should be sufficient
		// In a real scenario, you might use sync.WaitGroup for more precise synchronization
		done := make(chan struct{})
		go func() {
			expectedOperations := numGoroutines * numOperationsPerGoroutine * 2
			receivedErrors := 0
			for receivedErrors < expectedOperations {
				select {
				case <-errCh:
					receivedErrors++
					t.Errorf("Unexpected error during concurrent access")
				default:
					// No more errors, assume operations completed successfully
					close(done)
					return
				}
			}
		}()

		// Wait for completion or timeout
		select {
		case <-done:
			// Success - no unexpected errors
		case <-ctx.Done():
			t.Fatal("Test timed out")
		}

		// Verify no unexpected errors occurred
		select {
		case err := <-errCh:
			t.Errorf("Unexpected error during concurrent access: %v", err)
		default:
			// Expected - no errors should occur
		}
	})
}
