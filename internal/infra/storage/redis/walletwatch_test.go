package redis

import (
	"context"
	"testing"
	"time"

	"github.com/gabapcia/blockwatch/internal/walletwatch"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalletwatchIdempotencyKey(t *testing.T) {
	t.Run("generates correct key format", func(t *testing.T) {
		// Execute
		key := walletwatchIdempotencyKey("ethereum", "0xabc123")

		// Assert
		expected := "walletwatch:idempotency:ethereum:0xabc123"
		assert.Equal(t, expected, key)
	})

	t.Run("handles empty network", func(t *testing.T) {
		// Execute
		key := walletwatchIdempotencyKey("", "0xabc123")

		// Assert
		expected := "walletwatch:idempotency::0xabc123"
		assert.Equal(t, expected, key)
	})

	t.Run("handles empty block hash", func(t *testing.T) {
		// Execute
		key := walletwatchIdempotencyKey("ethereum", "")

		// Assert
		expected := "walletwatch:idempotency:ethereum:"
		assert.Equal(t, expected, key)
	})

	t.Run("handles both empty", func(t *testing.T) {
		// Execute
		key := walletwatchIdempotencyKey("", "")

		// Assert
		expected := "walletwatch:idempotency::"
		assert.Equal(t, expected, key)
	})

	t.Run("handles special characters", func(t *testing.T) {
		// Execute
		key := walletwatchIdempotencyKey("test:network", "0x123:456")

		// Assert
		expected := "walletwatch:idempotency:test:network:0x123:456"
		assert.Equal(t, expected, key)
	})
}

func TestClaimBlockForTxWatch(t *testing.T) {
	t.Run("successful claim on new block", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := t.Context()
		network := "ethereum"
		blockHash := "0xabc123"
		ttl := 5 * time.Minute

		// Execute
		err := client.ClaimBlockForTxWatch(ctx, network, blockHash, ttl)

		// Assert
		require.NoError(t, err)

		// Verify the key was set with correct TTL
		key := walletwatchIdempotencyKey(network, blockHash)
		val, err := client.conn.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, "", val) // Should be empty string for in-progress

		// Verify TTL is set
		ttlResult, err := client.conn.TTL(ctx, key).Result()
		require.NoError(t, err)
		assert.Greater(t, ttlResult, time.Duration(0))
		assert.LessOrEqual(t, ttlResult, ttl)
	})

	t.Run("returns ErrStillInProgress when block is already claimed", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := t.Context()
		network := "ethereum"
		blockHash := "0xdef456"
		ttl := 5 * time.Minute

		// First claim should succeed
		err := client.ClaimBlockForTxWatch(ctx, network, blockHash, ttl)
		require.NoError(t, err)

		// Execute - second claim should fail
		err = client.ClaimBlockForTxWatch(ctx, network, blockHash, ttl)

		// Assert
		require.Error(t, err)
		assert.Equal(t, walletwatch.ErrStillInProgress, err)
	})

	t.Run("returns ErrAlreadyFinished when block is marked as done", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := t.Context()
		network := "ethereum"
		blockHash := "0xghi789"

		// Mark block as completed first
		err := client.MarkBlockTxWatchComplete(ctx, network, blockHash)
		require.NoError(t, err)

		// Execute - attempt to claim should fail
		err = client.ClaimBlockForTxWatch(ctx, network, blockHash, 5*time.Minute)

		// Assert
		require.Error(t, err)
		assert.Equal(t, walletwatch.ErrAlreadyFinished, err)
	})

	t.Run("successful claim after TTL expires", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := t.Context()
		network := "ethereum"
		blockHash := "0xjkl012"
		shortTTL := 100 * time.Millisecond

		// First claim with short TTL
		err := client.ClaimBlockForTxWatch(ctx, network, blockHash, shortTTL)
		require.NoError(t, err)

		// Wait for TTL to expire
		time.Sleep(150 * time.Millisecond)

		// Execute - second claim should succeed after TTL expires
		err = client.ClaimBlockForTxWatch(ctx, network, blockHash, 5*time.Minute)

		// Assert
		require.NoError(t, err)
	})

	t.Run("handles different networks independently", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := t.Context()
		blockHash := "0xsame123"
		ttl := 5 * time.Minute

		// Execute - claim same block hash on different networks
		err1 := client.ClaimBlockForTxWatch(ctx, "ethereum", blockHash, ttl)
		err2 := client.ClaimBlockForTxWatch(ctx, "polygon", blockHash, ttl)

		// Assert - both should succeed as they're on different networks
		require.NoError(t, err1)
		require.NoError(t, err2)

		// Verify both keys exist
		key1 := walletwatchIdempotencyKey("ethereum", blockHash)
		key2 := walletwatchIdempotencyKey("polygon", blockHash)

		val1, err := client.conn.Get(ctx, key1).Result()
		require.NoError(t, err)
		assert.Equal(t, "", val1)

		val2, err := client.conn.Get(ctx, key2).Result()
		require.NoError(t, err)
		assert.Equal(t, "", val2)
	})

	t.Run("handles different block hashes independently", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := t.Context()
		network := "ethereum"
		ttl := 5 * time.Minute

		// Execute - claim different block hashes on same network
		err1 := client.ClaimBlockForTxWatch(ctx, network, "0xblock1", ttl)
		err2 := client.ClaimBlockForTxWatch(ctx, network, "0xblock2", ttl)

		// Assert - both should succeed as they're different blocks
		require.NoError(t, err1)
		require.NoError(t, err2)

		// Verify both keys exist
		key1 := walletwatchIdempotencyKey(network, "0xblock1")
		key2 := walletwatchIdempotencyKey(network, "0xblock2")

		val1, err := client.conn.Get(ctx, key1).Result()
		require.NoError(t, err)
		assert.Equal(t, "", val1)

		val2, err := client.conn.Get(ctx, key2).Result()
		require.NoError(t, err)
		assert.Equal(t, "", val2)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		network := "ethereum"
		blockHash := "0xcancelled"
		ttl := 5 * time.Minute

		// Execute
		err := client.ClaimBlockForTxWatch(ctx, network, blockHash, ttl)

		// Assert - should return context error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("handles zero TTL", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := t.Context()
		network := "ethereum"
		blockHash := "0xzero_ttl"
		ttl := time.Duration(0)

		// Execute
		err := client.ClaimBlockForTxWatch(ctx, network, blockHash, ttl)

		// Assert
		require.NoError(t, err)

		// Verify the key was set but with no TTL (persistent)
		key := walletwatchIdempotencyKey(network, blockHash)
		val, err := client.conn.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, "", val)

		// Verify no TTL is set (returns -1 for persistent keys)
		ttlResult, err := client.conn.TTL(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, time.Duration(-1), ttlResult)
	})

	t.Run("handles empty network name", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := t.Context()
		network := ""
		blockHash := "0xempty_network"
		ttl := 5 * time.Minute

		// Execute
		err := client.ClaimBlockForTxWatch(ctx, network, blockHash, ttl)

		// Assert - should still work with empty network
		require.NoError(t, err)

		// Verify the key was created with empty network
		key := walletwatchIdempotencyKey(network, blockHash)
		val, err := client.conn.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, "", val)
	})

	t.Run("handles empty block hash", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := t.Context()
		network := "ethereum"
		blockHash := ""
		ttl := 5 * time.Minute

		// Execute
		err := client.ClaimBlockForTxWatch(ctx, network, blockHash, ttl)

		// Assert - should still work with empty block hash
		require.NoError(t, err)

		// Verify the key was created with empty block hash
		key := walletwatchIdempotencyKey(network, blockHash)
		val, err := client.conn.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, "", val)
	})

	t.Run("concurrent claims on same block", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := t.Context()
		network := "ethereum"
		blockHash := "0xconcurrent"
		ttl := 5 * time.Minute

		// Execute concurrent claims
		errCh := make(chan error, 2)

		go func() {
			errCh <- client.ClaimBlockForTxWatch(ctx, network, blockHash, ttl)
		}()

		go func() {
			errCh <- client.ClaimBlockForTxWatch(ctx, network, blockHash, ttl)
		}()

		// Collect results
		err1 := <-errCh
		err2 := <-errCh

		// Assert - one should succeed, one should fail with ErrStillInProgress
		var successCount, inProgressCount int
		for _, err := range []error{err1, err2} {
			if err == nil {
				successCount++
			} else if err == walletwatch.ErrStillInProgress {
				inProgressCount++
			} else {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		assert.Equal(t, 1, successCount, "exactly one claim should succeed")
		assert.Equal(t, 1, inProgressCount, "exactly one claim should fail with ErrStillInProgress")
	})

	t.Run("handles SetNX error with negative TTL", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := t.Context()
		network := "ethereum"
		blockHash := "0xsetnx_negative_ttl"

		// Use a negative TTL which should cause SetNX to fail
		// Negative durations are invalid for Redis operations
		invalidTTL := -1 * time.Second

		// Execute - this should trigger the SetNX error path
		err := client.ClaimBlockForTxWatch(ctx, network, blockHash, invalidTTL)

		// Assert - should return an error (either from SetNX or validation)
		require.Error(t, err)
		// Should not be our custom errors
		assert.NotEqual(t, walletwatch.ErrAlreadyFinished, err)
		assert.NotEqual(t, walletwatch.ErrStillInProgress, err)
	})
}

func TestMarkBlockTxWatchComplete(t *testing.T) {
	t.Run("marks block as complete", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := t.Context()
		network := "ethereum"
		blockHash := "0xcomplete"

		// Execute
		err := client.MarkBlockTxWatchComplete(ctx, network, blockHash)

		// Assert
		require.NoError(t, err)

		// Verify the key was set to "done" with no TTL
		key := walletwatchIdempotencyKey(network, blockHash)
		val, err := client.conn.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, walletwatchIdempotencyDone, val)

		// Verify no TTL is set (returns -1 for persistent keys)
		ttlResult, err := client.conn.TTL(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, time.Duration(-1), ttlResult)
	})

	t.Run("overwrites existing in-progress claim", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := t.Context()
		network := "ethereum"
		blockHash := "0xoverwrite"
		ttl := 5 * time.Minute

		// First claim the block
		err := client.ClaimBlockForTxWatch(ctx, network, blockHash, ttl)
		require.NoError(t, err)

		// Execute - mark as complete
		err = client.MarkBlockTxWatchComplete(ctx, network, blockHash)

		// Assert
		require.NoError(t, err)

		// Verify the key was overwritten to "done"
		key := walletwatchIdempotencyKey(network, blockHash)
		val, err := client.conn.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, walletwatchIdempotencyDone, val)

		// Verify TTL was removed (returns -1 for persistent keys)
		ttlResult, err := client.conn.TTL(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, time.Duration(-1), ttlResult)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		network := "ethereum"
		blockHash := "0xcancelled_complete"

		// Execute
		err := client.MarkBlockTxWatchComplete(ctx, network, blockHash)

		// Assert - should return context error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}
