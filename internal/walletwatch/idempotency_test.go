package walletwatch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNopIdempotencyGuard_ClaimBlockForTxWatch(t *testing.T) {
	t.Run("always returns nil", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx := context.Background()

		err := guard.ClaimBlockForTxWatch(ctx, "ethereum", "0xabc123", 5*time.Minute)
		require.NoError(t, err)
	})

	t.Run("works with different networks", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx := context.Background()

		networks := []string{"ethereum", "polygon", "solana", "bitcoin", ""}
		for _, network := range networks {
			err := guard.ClaimBlockForTxWatch(ctx, network, "0xhash", time.Hour)
			require.NoError(t, err)
		}
	})

	t.Run("works with different block hashes", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx := context.Background()

		hashes := []string{"0xabc123", "0xdef456", "0x789xyz", "", "invalid-hash"}
		for _, hash := range hashes {
			err := guard.ClaimBlockForTxWatch(ctx, "ethereum", hash, time.Minute)
			require.NoError(t, err)
		}
	})

	t.Run("works with different TTL values", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx := context.Background()

		ttls := []time.Duration{
			0,
			time.Nanosecond,
			time.Millisecond,
			time.Second,
			time.Minute,
			time.Hour,
			24 * time.Hour,
			-time.Minute, // negative duration
		}

		for _, ttl := range ttls {
			err := guard.ClaimBlockForTxWatch(ctx, "ethereum", "0xhash", ttl)
			require.NoError(t, err)
		}
	})

	t.Run("works with cancelled context", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := guard.ClaimBlockForTxWatch(ctx, "ethereum", "0xabc123", time.Minute)
		require.NoError(t, err)
	})

	t.Run("works with context with timeout", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()

		// Wait for context to timeout
		<-ctx.Done()

		err := guard.ClaimBlockForTxWatch(ctx, "ethereum", "0xabc123", time.Minute)
		require.NoError(t, err)
	})

	t.Run("multiple calls with same parameters", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx := context.Background()

		// Multiple calls with identical parameters should all succeed
		for i := 0; i < 10; i++ {
			err := guard.ClaimBlockForTxWatch(ctx, "ethereum", "0xsame", time.Minute)
			require.NoError(t, err)
		}
	})

	t.Run("concurrent calls", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx := context.Background()

		// Run multiple goroutines concurrently
		done := make(chan error, 10)
		for i := 0; i < 10; i++ {
			go func(id int) {
				err := guard.ClaimBlockForTxWatch(ctx, "ethereum", "0xconcurrent", time.Minute)
				done <- err
			}(i)
		}

		// All should succeed
		for i := 0; i < 10; i++ {
			err := <-done
			require.NoError(t, err)
		}
	})
}

func TestNopIdempotencyGuard_MarkBlockTxWatchComplete(t *testing.T) {
	t.Run("always returns nil", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx := context.Background()

		err := guard.MarkBlockTxWatchComplete(ctx, "ethereum", "0xabc123")
		require.NoError(t, err)
	})

	t.Run("works with different networks", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx := context.Background()

		networks := []string{"ethereum", "polygon", "solana", "bitcoin", ""}
		for _, network := range networks {
			err := guard.MarkBlockTxWatchComplete(ctx, network, "0xhash")
			require.NoError(t, err)
		}
	})

	t.Run("works with different block hashes", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx := context.Background()

		hashes := []string{"0xabc123", "0xdef456", "0x789xyz", "", "invalid-hash"}
		for _, hash := range hashes {
			err := guard.MarkBlockTxWatchComplete(ctx, "ethereum", hash)
			require.NoError(t, err)
		}
	})

	t.Run("works with cancelled context", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := guard.MarkBlockTxWatchComplete(ctx, "ethereum", "0xabc123")
		require.NoError(t, err)
	})

	t.Run("works with context with timeout", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()

		// Wait for context to timeout
		<-ctx.Done()

		err := guard.MarkBlockTxWatchComplete(ctx, "ethereum", "0xabc123")
		require.NoError(t, err)
	})

	t.Run("multiple calls with same parameters", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx := context.Background()

		// Multiple calls with identical parameters should all succeed
		for i := 0; i < 10; i++ {
			err := guard.MarkBlockTxWatchComplete(ctx, "ethereum", "0xsame")
			require.NoError(t, err)
		}
	})

	t.Run("concurrent calls", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx := context.Background()

		// Run multiple goroutines concurrently
		done := make(chan error, 10)
		for i := 0; i < 10; i++ {
			go func(id int) {
				err := guard.MarkBlockTxWatchComplete(ctx, "ethereum", "0xconcurrent")
				done <- err
			}(i)
		}

		// All should succeed
		for i := 0; i < 10; i++ {
			err := <-done
			require.NoError(t, err)
		}
	})

	t.Run("can be called after claim", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx := context.Background()

		// Simulate normal flow: claim then mark complete
		err := guard.ClaimBlockForTxWatch(ctx, "ethereum", "0xflow", time.Minute)
		require.NoError(t, err)

		err = guard.MarkBlockTxWatchComplete(ctx, "ethereum", "0xflow")
		require.NoError(t, err)
	})

	t.Run("can be called without prior claim", func(t *testing.T) {
		guard := nopIdempotencyGuard{}
		ctx := context.Background()

		// Mark complete without claiming first - should still work
		err := guard.MarkBlockTxWatchComplete(ctx, "ethereum", "0xnoclaim")
		require.NoError(t, err)
	})
}
