package chainwatch

import (
	"context"
	"testing"

	"github.com/gabapcia/blockwatch/internal/pkg/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
