package postgresql

import (
	"context"
	"testing"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/pkg/x/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveCheckpoint(t *testing.T) {
	client, cleanup := setupPostgreSQLContainer(t)
	defer cleanup()

	network := "ethereum"

	t.Run("basic save and load workflow", func(t *testing.T) {
		height := types.HexFromInt(100)

		err := client.SaveCheckpoint(t.Context(), network, height)
		require.NoError(t, err)

		loaded, err := client.LoadLatestCheckpoint(t.Context(), network)
		require.NoError(t, err)
		assert.Equal(t, height, loaded)
	})

	t.Run("duplicate and higher checkpoints", func(t *testing.T) {
		height1 := types.HexFromInt(200)
		height2 := types.HexFromInt(300)

		// Save initial checkpoint
		err := client.SaveCheckpoint(t.Context(), network, height1)
		require.NoError(t, err)

		// Save duplicate (should not error)
		err = client.SaveCheckpoint(t.Context(), network, height1)
		require.NoError(t, err)

		// Save higher checkpoint
		err = client.SaveCheckpoint(t.Context(), network, height2)
		require.NoError(t, err)

		// Should return the highest
		loaded, err := client.LoadLatestCheckpoint(t.Context(), network)
		require.NoError(t, err)
		assert.Equal(t, height2, loaded)
	})

	t.Run("network normalization and isolation", func(t *testing.T) {
		ethereumHeight := types.HexFromInt(400)
		solanaHeight := types.HexFromInt(500)

		// Save with lowercase, load with uppercase
		err := client.SaveCheckpoint(t.Context(), "ethereum", ethereumHeight)
		require.NoError(t, err)

		loaded, err := client.LoadLatestCheckpoint(t.Context(), "ETHEREUM")
		require.NoError(t, err)
		assert.Equal(t, ethereumHeight, loaded)

		// Different networks are isolated
		err = client.SaveCheckpoint(t.Context(), "solana", solanaHeight)
		require.NoError(t, err)

		loaded, err = client.LoadLatestCheckpoint(t.Context(), "solana")
		require.NoError(t, err)
		assert.Equal(t, solanaHeight, loaded)

		// Original network unchanged
		loaded, err = client.LoadLatestCheckpoint(t.Context(), "ethereum")
		require.NoError(t, err)
		assert.Equal(t, ethereumHeight, loaded)
	})

	t.Run("edge cases and errors", func(t *testing.T) {
		// No checkpoint found
		_, err := client.LoadLatestCheckpoint(t.Context(), "nonexistent")
		assert.ErrorIs(t, err, chainstream.ErrNoCheckpointFound)

		// Zero height
		zeroHeight := types.HexFromInt(0)
		err = client.SaveCheckpoint(t.Context(), "zerotest", zeroHeight)
		require.NoError(t, err)

		loaded, err := client.LoadLatestCheckpoint(t.Context(), "zerotest")
		require.NoError(t, err)
		assert.Equal(t, zeroHeight, loaded)

		// Context cancellation
		cancelCtx, cancel := context.WithCancel(t.Context())
		cancel()

		err = client.SaveCheckpoint(cancelCtx, network, types.HexFromInt(999))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")

		_, err = client.LoadLatestCheckpoint(cancelCtx, network)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("concurrent operations and latest selection", func(t *testing.T) {
		heights := []int64{1000, 1100, 900, 1200, 1050}
		expectedLatest := types.HexFromInt(1200)

		// Save multiple checkpoints concurrently
		results := make(chan error, len(heights))
		for _, h := range heights {
			go func(height int64) {
				err := client.SaveCheckpoint(t.Context(), network, types.HexFromInt(height))
				results <- err
			}(h)
		}

		// All saves should succeed
		for i := 0; i < len(heights); i++ {
			err := <-results
			assert.NoError(t, err)
		}

		// Should return the highest checkpoint
		loaded, err := client.LoadLatestCheckpoint(t.Context(), network)
		require.NoError(t, err)
		assert.Equal(t, expectedLatest, loaded)
	})
}

func TestLoadLatestCheckpoint(t *testing.T) {
	client, cleanup := setupPostgreSQLContainer(t)
	defer cleanup()

	network := "ethereum"

	t.Run("no checkpoint found", func(t *testing.T) {
		_, err := client.LoadLatestCheckpoint(t.Context(), "nonexistent")
		assert.ErrorIs(t, err, chainstream.ErrNoCheckpointFound)
	})

	t.Run("load after save", func(t *testing.T) {
		height := types.HexFromInt(1000)

		err := client.SaveCheckpoint(t.Context(), network, height)
		require.NoError(t, err)

		loaded, err := client.LoadLatestCheckpoint(t.Context(), network)
		require.NoError(t, err)
		assert.Equal(t, height, loaded)
	})

	t.Run("returns latest checkpoint from multiple", func(t *testing.T) {
		heights := []int64{2000, 2100, 1900, 2200, 2050}
		expectedLatest := types.HexFromInt(2200)

		// Save multiple checkpoints in random order
		for _, h := range heights {
			height := types.HexFromInt(h)
			err := client.SaveCheckpoint(t.Context(), network, height)
			require.NoError(t, err)
		}

		// Should return the highest one
		loaded, err := client.LoadLatestCheckpoint(t.Context(), network)
		require.NoError(t, err)
		assert.Equal(t, expectedLatest, loaded)
	})

	t.Run("network case insensitive", func(t *testing.T) {
		height := types.HexFromInt(3000)

		// Save with lowercase
		err := client.SaveCheckpoint(t.Context(), "polygon", height)
		require.NoError(t, err)

		// Load with different cases
		loaded, err := client.LoadLatestCheckpoint(t.Context(), "POLYGON")
		require.NoError(t, err)
		assert.Equal(t, height, loaded)

		loaded, err = client.LoadLatestCheckpoint(t.Context(), "Polygon")
		require.NoError(t, err)
		assert.Equal(t, height, loaded)

		loaded, err = client.LoadLatestCheckpoint(t.Context(), "pOlYgOn")
		require.NoError(t, err)
		assert.Equal(t, height, loaded)
	})

	t.Run("network isolation", func(t *testing.T) {
		ethereumHeight := types.HexFromInt(4000)
		solanaHeight := types.HexFromInt(5000)
		avalancheHeight := types.HexFromInt(6000)

		// Save checkpoints for different networks
		err := client.SaveCheckpoint(t.Context(), "ethereum", ethereumHeight)
		require.NoError(t, err)

		err = client.SaveCheckpoint(t.Context(), "solana", solanaHeight)
		require.NoError(t, err)

		err = client.SaveCheckpoint(t.Context(), "avalanche", avalancheHeight)
		require.NoError(t, err)

		// Verify each network returns its own checkpoint
		loaded, err := client.LoadLatestCheckpoint(t.Context(), "ethereum")
		require.NoError(t, err)
		assert.Equal(t, ethereumHeight, loaded)

		loaded, err = client.LoadLatestCheckpoint(t.Context(), "solana")
		require.NoError(t, err)
		assert.Equal(t, solanaHeight, loaded)

		loaded, err = client.LoadLatestCheckpoint(t.Context(), "avalanche")
		require.NoError(t, err)
		assert.Equal(t, avalancheHeight, loaded)
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := client.LoadLatestCheckpoint(cancelCtx, network)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("empty network name", func(t *testing.T) {
		height := types.HexFromInt(7000)

		// Save with empty network
		err := client.SaveCheckpoint(t.Context(), "", height)
		require.NoError(t, err)

		// Load with empty network
		loaded, err := client.LoadLatestCheckpoint(t.Context(), "")
		require.NoError(t, err)
		assert.Equal(t, height, loaded)
	})

	t.Run("zero height checkpoint", func(t *testing.T) {
		zeroHeight := types.HexFromInt(0)

		err := client.SaveCheckpoint(t.Context(), "zeronet", zeroHeight)
		require.NoError(t, err)

		loaded, err := client.LoadLatestCheckpoint(t.Context(), "zeronet")
		require.NoError(t, err)
		assert.Equal(t, zeroHeight, loaded)
	})

	t.Run("large height values", func(t *testing.T) {
		largeHeight := types.HexFromInt(9223372036854775807) // max int64

		err := client.SaveCheckpoint(t.Context(), "largenet", largeHeight)
		require.NoError(t, err)

		loaded, err := client.LoadLatestCheckpoint(t.Context(), "largenet")
		require.NoError(t, err)
		assert.Equal(t, largeHeight, loaded)
	})
}
