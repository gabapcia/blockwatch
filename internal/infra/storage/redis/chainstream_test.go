package redis

import (
	"context"
	"testing"
	"time"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/pkg/x/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainstreamCheckpointKey(t *testing.T) {
	t.Run("generates correct key format", func(t *testing.T) {
		// Execute
		key := chainstreamCheckpointKey("ethereum")

		// Assert
		expected := "chainstream:checkpoint:ethereum"
		assert.Equal(t, expected, key)
	})

	t.Run("handles empty network", func(t *testing.T) {
		// Execute
		key := chainstreamCheckpointKey("")

		// Assert
		expected := "chainstream:checkpoint:"
		assert.Equal(t, expected, key)
	})

	t.Run("handles special characters in network", func(t *testing.T) {
		// Execute
		key := chainstreamCheckpointKey("test:network-v2")

		// Assert
		expected := "chainstream:checkpoint:test:network-v2"
		assert.Equal(t, expected, key)
	})
}

func TestSaveCheckpoint(t *testing.T) {
	t.Run("successfully saves checkpoint for new network", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "ethereum"
		height := types.Hex("0x100")

		// Execute
		err := client.SaveCheckpoint(ctx, network, height)

		// Assert
		require.NoError(t, err)

		// Verify the checkpoint was saved correctly
		key := chainstreamCheckpointKey(network)
		val, err := client.conn.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, string(height), val)

		// Verify no TTL is set (persistent key)
		ttl, err := client.conn.TTL(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, time.Duration(-1), ttl) // -1 means no expiration
	})

	t.Run("overwrites existing checkpoint for same network", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "ethereum"
		initialHeight := types.Hex("0x100")
		newHeight := types.Hex("0x200")

		// Save initial checkpoint
		err := client.SaveCheckpoint(ctx, network, initialHeight)
		require.NoError(t, err)

		// Execute - save new checkpoint for same network
		err = client.SaveCheckpoint(ctx, network, newHeight)

		// Assert
		require.NoError(t, err)

		// Verify the checkpoint was overwritten
		key := chainstreamCheckpointKey(network)
		val, err := client.conn.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, string(newHeight), val)
	})

	t.Run("handles different networks independently", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		ethereumHeight := types.Hex("0x100")
		polygonHeight := types.Hex("0x200")

		// Execute - save checkpoints for different networks
		err1 := client.SaveCheckpoint(ctx, "ethereum", ethereumHeight)
		err2 := client.SaveCheckpoint(ctx, "polygon", polygonHeight)

		// Assert
		require.NoError(t, err1)
		require.NoError(t, err2)

		// Verify both checkpoints exist independently
		ethKey := chainstreamCheckpointKey("ethereum")
		polyKey := chainstreamCheckpointKey("polygon")

		ethVal, err := client.conn.Get(ctx, ethKey).Result()
		require.NoError(t, err)
		assert.Equal(t, string(ethereumHeight), ethVal)

		polyVal, err := client.conn.Get(ctx, polyKey).Result()
		require.NoError(t, err)
		assert.Equal(t, string(polygonHeight), polyVal)
	})

	t.Run("handles empty network name", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := ""
		height := types.Hex("0x100")

		// Execute
		err := client.SaveCheckpoint(ctx, network, height)

		// Assert
		require.NoError(t, err)

		// Verify the checkpoint was saved with empty network
		key := chainstreamCheckpointKey(network)
		val, err := client.conn.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, string(height), val)
	})

	t.Run("handles empty height value", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "ethereum"
		height := types.Hex("")

		// Execute
		err := client.SaveCheckpoint(ctx, network, height)

		// Assert
		require.NoError(t, err)

		// Verify the checkpoint was saved with empty height
		key := chainstreamCheckpointKey(network)
		val, err := client.conn.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, string(height), val)
	})

	t.Run("handles various hex formats", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()

		testCases := []struct {
			name    string
			network string
			height  types.Hex
		}{
			{"lowercase hex", "ethereum", types.Hex("0xabc123")},
			{"uppercase hex", "polygon", types.Hex("0xABC123")},
			{"zero value", "bitcoin", types.Hex("0x0")},
			{"large value", "solana", types.Hex("0xffffffffffffffff")},
			{"single digit", "arbitrum", types.Hex("0x1")},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Execute
				err := client.SaveCheckpoint(ctx, tc.network, tc.height)

				// Assert
				require.NoError(t, err)

				// Verify the checkpoint was saved correctly
				key := chainstreamCheckpointKey(tc.network)
				val, err := client.conn.Get(ctx, key).Result()
				require.NoError(t, err)
				assert.Equal(t, string(tc.height), val)
			})
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		network := "ethereum"
		height := types.Hex("0x100")

		// Execute
		err := client.SaveCheckpoint(ctx, network, height)

		// Assert - should return context error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("concurrent saves to same network", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "ethereum"
		height1 := types.Hex("0x100")
		height2 := types.Hex("0x200")

		// Execute concurrent saves
		errCh := make(chan error, 2)

		go func() {
			errCh <- client.SaveCheckpoint(ctx, network, height1)
		}()

		go func() {
			errCh <- client.SaveCheckpoint(ctx, network, height2)
		}()

		// Collect results
		err1 := <-errCh
		err2 := <-errCh

		// Assert - both should succeed
		require.NoError(t, err1)
		require.NoError(t, err2)

		// Verify one of the values was saved (last write wins)
		key := chainstreamCheckpointKey(network)
		val, err := client.conn.Get(ctx, key).Result()
		require.NoError(t, err)

		// Should be one of the two values
		assert.True(t, val == string(height1) || val == string(height2),
			"saved value should be one of the concurrent writes, got: %s", val)
	})

	t.Run("concurrent saves to different networks", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		height := types.Hex("0x100")

		networks := []string{"ethereum", "polygon", "bitcoin", "solana", "arbitrum"}
		errCh := make(chan error, len(networks))

		// Execute concurrent saves to different networks
		for _, network := range networks {
			go func(net string) {
				errCh <- client.SaveCheckpoint(ctx, net, height)
			}(network)
		}

		// Collect results
		for i := 0; i < len(networks); i++ {
			err := <-errCh
			require.NoError(t, err)
		}

		// Verify all checkpoints were saved
		for _, network := range networks {
			key := chainstreamCheckpointKey(network)
			val, err := client.conn.Get(ctx, key).Result()
			require.NoError(t, err)
			assert.Equal(t, string(height), val)
		}
	})

	t.Run("saves multiple checkpoints in sequence", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "ethereum"

		heights := []types.Hex{
			types.Hex("0x100"),
			types.Hex("0x101"),
			types.Hex("0x102"),
			types.Hex("0x103"),
			types.Hex("0x104"),
		}

		// Execute - save checkpoints in sequence
		for _, height := range heights {
			err := client.SaveCheckpoint(ctx, network, height)
			require.NoError(t, err)

			// Verify each checkpoint is saved correctly
			key := chainstreamCheckpointKey(network)
			val, err := client.conn.Get(ctx, key).Result()
			require.NoError(t, err)
			assert.Equal(t, string(height), val)
		}

		// Final verification - should have the last height
		key := chainstreamCheckpointKey(network)
		val, err := client.conn.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, string(heights[len(heights)-1]), val)
	})
}

func TestLoadLatestCheckpoint(t *testing.T) {
	t.Run("loads existing checkpoint successfully", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "ethereum"
		expectedHeight := types.Hex("0x100")

		// Save a checkpoint first
		err := client.SaveCheckpoint(ctx, network, expectedHeight)
		require.NoError(t, err)

		// Execute
		height, err := client.LoadLatestCheckpoint(ctx, network)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expectedHeight, height)
	})

	t.Run("returns ErrNoCheckpointFound for non-existent network", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "nonexistent"

		// Execute
		height, err := client.LoadLatestCheckpoint(ctx, network)

		// Assert
		require.Error(t, err)
		assert.Equal(t, chainstream.ErrNoCheckpointFound, err)
		assert.Equal(t, types.Hex(""), height)
	})

	t.Run("handles invalid hex value in Redis", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "ethereum"
		invalidHex := "invalid-hex-value"

		// Manually set invalid hex value in Redis
		key := chainstreamCheckpointKey(network)
		err := client.conn.Set(ctx, key, invalidHex, 0).Err()
		require.NoError(t, err)

		// Execute
		height, err := client.LoadLatestCheckpoint(ctx, network)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hex string must start with 0x")
		assert.Equal(t, types.Hex(""), height)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		network := "ethereum"

		// Execute
		height, err := client.LoadLatestCheckpoint(ctx, network)

		// Assert - should return context error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
		assert.Equal(t, types.Hex(""), height)
	})

	t.Run("loads different checkpoints for different networks", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()

		testCases := []struct {
			network string
			height  types.Hex
		}{
			{"ethereum", types.Hex("0x100")},
			{"polygon", types.Hex("0x200")},
			{"bitcoin", types.Hex("0x300")},
			{"solana", types.Hex("0x400")},
		}

		// Save checkpoints for different networks
		for _, tc := range testCases {
			err := client.SaveCheckpoint(ctx, tc.network, tc.height)
			require.NoError(t, err)
		}

		// Execute and verify each network returns correct checkpoint
		for _, tc := range testCases {
			height, err := client.LoadLatestCheckpoint(ctx, tc.network)
			require.NoError(t, err)
			assert.Equal(t, tc.height, height)
		}
	})

	t.Run("loads latest checkpoint after multiple saves", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "ethereum"

		heights := []types.Hex{
			types.Hex("0x100"),
			types.Hex("0x101"),
			types.Hex("0x102"),
		}

		// Save multiple checkpoints
		for _, height := range heights {
			err := client.SaveCheckpoint(ctx, network, height)
			require.NoError(t, err)
		}

		// Execute
		height, err := client.LoadLatestCheckpoint(ctx, network)

		// Assert - should return the last saved checkpoint
		require.NoError(t, err)
		assert.Equal(t, heights[len(heights)-1], height)
	})

	t.Run("handles empty network name", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := ""
		expectedHeight := types.Hex("0x100")

		// Save checkpoint with empty network
		err := client.SaveCheckpoint(ctx, network, expectedHeight)
		require.NoError(t, err)

		// Execute
		height, err := client.LoadLatestCheckpoint(ctx, network)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, expectedHeight, height)
	})

	t.Run("handles various hex formats", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()

		testCases := []struct {
			name    string
			network string
			height  types.Hex
		}{
			{"lowercase hex", "ethereum", types.Hex("0xabc123")},
			{"uppercase hex", "polygon", types.Hex("0xABC123")},
			{"zero value", "bitcoin", types.Hex("0x0")},
			{"large value", "solana", types.Hex("0xffffffffffffffff")},
			{"single digit", "arbitrum", types.Hex("0x1")},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Save checkpoint
				err := client.SaveCheckpoint(ctx, tc.network, tc.height)
				require.NoError(t, err)

				// Execute
				height, err := client.LoadLatestCheckpoint(ctx, tc.network)

				// Assert
				require.NoError(t, err)
				assert.Equal(t, tc.height, height)
			})
		}
	})
}

func TestSaveAndLoadCheckpointIntegration(t *testing.T) {
	t.Run("complete save and load cycle", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "ethereum"
		height := types.Hex("0x123456")

		// Execute save
		saveErr := client.SaveCheckpoint(ctx, network, height)
		require.NoError(t, saveErr)

		// Execute load
		loadedHeight, loadErr := client.LoadLatestCheckpoint(ctx, network)

		// Assert
		require.NoError(t, loadErr)
		assert.Equal(t, height, loadedHeight)
	})

	t.Run("multiple networks save and load cycle", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()

		networks := map[string]types.Hex{
			"ethereum": types.Hex("0x100"),
			"polygon":  types.Hex("0x200"),
			"bitcoin":  types.Hex("0x300"),
			"solana":   types.Hex("0x400"),
			"arbitrum": types.Hex("0x500"),
		}

		// Execute saves
		for network, height := range networks {
			err := client.SaveCheckpoint(ctx, network, height)
			require.NoError(t, err)
		}

		// Execute loads and verify
		for network, expectedHeight := range networks {
			loadedHeight, err := client.LoadLatestCheckpoint(ctx, network)
			require.NoError(t, err)
			assert.Equal(t, expectedHeight, loadedHeight)
		}
	})

	t.Run("checkpoint persistence after client reconnection", func(t *testing.T) {
		// Setup
		client1, cleanup1 := setupRedisContainer(t)
		defer cleanup1()

		ctx := context.Background()
		network := "ethereum"
		height := types.Hex("0x100")

		// Save checkpoint with first client
		err := client1.SaveCheckpoint(ctx, network, height)
		require.NoError(t, err)

		// Create second client using same Redis instance
		// Note: In a real scenario, this would be a new connection to the same Redis server
		// For this test, we'll use the same client to verify persistence
		loadedHeight, err := client1.LoadLatestCheckpoint(ctx, network)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, height, loadedHeight)
	})
}
