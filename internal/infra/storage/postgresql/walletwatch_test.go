package postgresql

import (
	"context"
	"testing"
	"time"

	"github.com/gabapcia/blockwatch/internal/walletwatch"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaimBlockForTxWatch(t *testing.T) {
	client, cleanup := setupPostgreSQLContainer(t)
	defer cleanup()

	network := "ethereum"
	ttl := 5 * time.Minute

	t.Run("successful first claim", func(t *testing.T) {
		blockHash := "0x1234567890abcdef"
		err := client.ClaimBlockForTxWatch(t.Context(), network, blockHash, ttl)
		require.NoError(t, err)
	})

	t.Run("claim already in progress", func(t *testing.T) {
		blockHash := "0xabcdef1234567890"

		err := client.ClaimBlockForTxWatch(t.Context(), network, blockHash, ttl)
		require.NoError(t, err)

		err = client.ClaimBlockForTxWatch(t.Context(), network, blockHash, ttl)
		assert.ErrorIs(t, err, walletwatch.ErrStillInProgress)
	})

	t.Run("claim already finished", func(t *testing.T) {
		blockHash := "0xfedcba0987654321"

		err := client.ClaimBlockForTxWatch(t.Context(), network, blockHash, ttl)
		require.NoError(t, err)

		err = client.MarkBlockTxWatchComplete(t.Context(), network, blockHash)
		require.NoError(t, err)

		err = client.ClaimBlockForTxWatch(t.Context(), network, blockHash, ttl)
		assert.ErrorIs(t, err, walletwatch.ErrAlreadyFinished)
	})

	t.Run("claim expired lock", func(t *testing.T) {
		blockHash := "0x1111222233334444"
		shortTTL := 100 * time.Millisecond

		err := client.ClaimBlockForTxWatch(t.Context(), network, blockHash, shortTTL)
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		err = client.ClaimBlockForTxWatch(t.Context(), network, blockHash, ttl)
		require.NoError(t, err)
	})

	t.Run("network normalization", func(t *testing.T) {
		blockHash := "0x5555666677778888"

		err := client.ClaimBlockForTxWatch(t.Context(), "ethereum", blockHash, ttl)
		require.NoError(t, err)

		err = client.ClaimBlockForTxWatch(t.Context(), "ETHEREUM", blockHash, ttl)
		assert.ErrorIs(t, err, walletwatch.ErrStillInProgress)

		err = client.ClaimBlockForTxWatch(t.Context(), "Ethereum", blockHash, ttl)
		assert.ErrorIs(t, err, walletwatch.ErrStillInProgress)
	})

	t.Run("different networks same block hash", func(t *testing.T) {
		blockHash := "0x9999aaaabbbbcccc"

		err := client.ClaimBlockForTxWatch(t.Context(), "ethereum", blockHash, ttl)
		require.NoError(t, err)

		err = client.ClaimBlockForTxWatch(t.Context(), "solana", blockHash, ttl)
		require.NoError(t, err)
	})

	t.Run("concurrent claims", func(t *testing.T) {
		blockHash := "0xddddeeeeffffaaaa"

		results := make(chan error, 2)

		go func() {
			err := client.ClaimBlockForTxWatch(t.Context(), network, blockHash, ttl)
			results <- err
		}()

		go func() {
			err := client.ClaimBlockForTxWatch(t.Context(), network, blockHash, ttl)
			results <- err
		}()

		err1 := <-results
		err2 := <-results

		if err1 == nil {
			assert.ErrorIs(t, err2, walletwatch.ErrStillInProgress)
		} else if err2 == nil {
			assert.ErrorIs(t, err1, walletwatch.ErrStillInProgress)
		} else {
			t.Fatal("Both concurrent claims failed, expected one to succeed")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		blockHash := "0xbbbbccccddddeeee"

		cancelCtx, cancel := context.WithCancel(t.Context())
		cancel()

		err := client.ClaimBlockForTxWatch(cancelCtx, network, blockHash, ttl)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("database error scenarios", func(t *testing.T) {
		blockHash := "0xdberrortest1234"

		// Test with invalid context to trigger database errors
		invalidCtx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-1*time.Hour))
		defer cancel()

		err := client.ClaimBlockForTxWatch(invalidCtx, network, blockHash, ttl)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("full lifecycle", func(t *testing.T) {
		blockHash := "0xintegrationtest123"

		err := client.ClaimBlockForTxWatch(t.Context(), network, blockHash, ttl)
		require.NoError(t, err)

		err = client.ClaimBlockForTxWatch(t.Context(), network, blockHash, ttl)
		assert.ErrorIs(t, err, walletwatch.ErrStillInProgress)

		err = client.MarkBlockTxWatchComplete(t.Context(), network, blockHash)
		require.NoError(t, err)

		err = client.ClaimBlockForTxWatch(t.Context(), network, blockHash, ttl)
		assert.ErrorIs(t, err, walletwatch.ErrAlreadyFinished)

		err = client.MarkBlockTxWatchComplete(t.Context(), network, blockHash)
		require.NoError(t, err)

		err = client.ClaimBlockForTxWatch(t.Context(), network, blockHash, ttl)
		assert.ErrorIs(t, err, walletwatch.ErrAlreadyFinished)
	})
}

func TestMarkBlockTxWatchComplete(t *testing.T) {
	client, cleanup := setupPostgreSQLContainer(t)
	defer cleanup()

	network := "ethereum"
	ttl := 5 * time.Minute

	t.Run("basic completion workflow", func(t *testing.T) {
		blockHash := "0x1234567890abcdef"

		// Claim, complete, and verify
		err := client.ClaimBlockForTxWatch(t.Context(), network, blockHash, ttl)
		require.NoError(t, err)

		err = client.MarkBlockTxWatchComplete(t.Context(), network, blockHash)
		require.NoError(t, err)

		err = client.ClaimBlockForTxWatch(t.Context(), network, blockHash, ttl)
		assert.ErrorIs(t, err, walletwatch.ErrAlreadyFinished)
	})

	t.Run("idempotent and unclaimed completion", func(t *testing.T) {
		blockHash1 := "0xabcdef1234567890"
		blockHash2 := "0xfedcba0987654321"

		// Test idempotent completion
		err := client.ClaimBlockForTxWatch(t.Context(), network, blockHash1, ttl)
		require.NoError(t, err)

		err = client.MarkBlockTxWatchComplete(t.Context(), network, blockHash1)
		require.NoError(t, err)

		err = client.MarkBlockTxWatchComplete(t.Context(), network, blockHash1)
		require.NoError(t, err) // Should succeed (idempotent)

		err = client.ClaimBlockForTxWatch(t.Context(), network, blockHash1, ttl)
		assert.ErrorIs(t, err, walletwatch.ErrAlreadyFinished)

		// Test completing unclaimed block (no-op - doesn't create record)
		err = client.MarkBlockTxWatchComplete(t.Context(), network, blockHash2)
		require.NoError(t, err)

		// Since no record was created, claim should succeed
		err = client.ClaimBlockForTxWatch(t.Context(), network, blockHash2, ttl)
		require.NoError(t, err)
	})

	t.Run("network normalization and isolation", func(t *testing.T) {
		blockHash := "0x1111222233334444"

		// Claim with lowercase, complete with uppercase
		err := client.ClaimBlockForTxWatch(t.Context(), "ethereum", blockHash, ttl)
		require.NoError(t, err)

		err = client.MarkBlockTxWatchComplete(t.Context(), "ETHEREUM", blockHash)
		require.NoError(t, err)

		// Verify with mixed case
		err = client.ClaimBlockForTxWatch(t.Context(), "Ethereum", blockHash, ttl)
		assert.ErrorIs(t, err, walletwatch.ErrAlreadyFinished)

		// Different network should be independent
		err = client.ClaimBlockForTxWatch(t.Context(), "solana", blockHash, ttl)
		require.NoError(t, err)
	})

	t.Run("concurrent completion and edge cases", func(t *testing.T) {
		blockHash1 := "0x5555666677778888"
		blockHash2 := "0x9999aaaabbbbcccc"
		shortTTL := 100 * time.Millisecond

		// Test concurrent completions
		err := client.ClaimBlockForTxWatch(t.Context(), network, blockHash1, ttl)
		require.NoError(t, err)

		results := make(chan error, 2)
		for i := 0; i < 2; i++ {
			go func() {
				results <- client.MarkBlockTxWatchComplete(t.Context(), network, blockHash1)
			}()
		}

		// Both should succeed (idempotent)
		for i := 0; i < 2; i++ {
			err := <-results
			assert.NoError(t, err)
		}

		// Test completing expired claim
		err = client.ClaimBlockForTxWatch(t.Context(), network, blockHash2, shortTTL)
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		err = client.MarkBlockTxWatchComplete(t.Context(), network, blockHash2)
		require.NoError(t, err)

		err = client.ClaimBlockForTxWatch(t.Context(), network, blockHash2, ttl)
		assert.ErrorIs(t, err, walletwatch.ErrAlreadyFinished)
	})

	t.Run("context cancellation and empty values", func(t *testing.T) {
		// Test context cancellation
		cancelCtx, cancel := context.WithCancel(t.Context())
		cancel()

		err := client.MarkBlockTxWatchComplete(cancelCtx, network, "0xtest")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")

		// Test empty values
		err = client.ClaimBlockForTxWatch(t.Context(), "", "", ttl)
		require.NoError(t, err)

		err = client.MarkBlockTxWatchComplete(t.Context(), "", "")
		require.NoError(t, err)

		err = client.ClaimBlockForTxWatch(t.Context(), "", "", ttl)
		assert.ErrorIs(t, err, walletwatch.ErrAlreadyFinished)
	})
}
