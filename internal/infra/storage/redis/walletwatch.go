package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gabapcia/blockwatch/internal/walletwatch"

	"github.com/redis/go-redis/v9"
)

const (
	// walletwatchKeyPrefix is the Redis key namespace used to store idempotency entries
	// related to wallet transaction watching. All keys will be prefixed with this value.
	walletwatchKeyPrefix = "walletwatch"

	// walletwatchIdempotencyDone is the terminal value stored in Redis to indicate that
	// a block has already been fully processed and should not be processed again.
	walletwatchIdempotencyDone = "done"
)

// walletwatchIdempotencyKey builds the Redis key used to track idempotency
// for a given block in a specific network.
func walletwatchIdempotencyKey(network, blockHash string) string {
	return fmt.Sprintf("%s:idempotency:%s:%s", walletwatchKeyPrefix, network, blockHash)
}

// ClaimBlockForTxWatch attempts to claim exclusive rights to process a block for transaction watching.
//
// Behavior:
//   - If the key is already marked as "done", it returns ErrAlreadyFinished.
//   - If the key exists but is not "done", it returns ErrStillInProgress.
//   - Otherwise, it sets an empty string value with TTL to reserve the claim.
//
// This function guarantees that only one process can scan the block at a time.
//
// Returns:
//   - nil if the claim is successful.
//   - walletwatch.ErrAlreadyFinished if the block was already processed.
//   - walletwatch.ErrStillInProgress if another process is handling it.
//   - any other error if the Redis operation fails.
func (s *client) ClaimBlockForTxWatch(ctx context.Context, network, blockHash string, ttl time.Duration) error {
	key := walletwatchIdempotencyKey(network, blockHash)

	val, err := s.conn.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return err
	}

	if val == walletwatchIdempotencyDone {
		return walletwatch.ErrAlreadyFinished
	}

	ok, err := s.conn.SetNX(ctx, key, "", ttl).Result()
	if err != nil {
		return err
	}

	if !ok {
		return walletwatch.ErrStillInProgress
	}

	return nil
}

// MarkBlockTxWatchComplete marks the given block as successfully processed by setting
// the Redis key value to "done" with no expiration.
//
// This prevents any future attempts to reprocess the same block.
func (s *client) MarkBlockTxWatchComplete(ctx context.Context, network, blockHash string) error {
	key := walletwatchIdempotencyKey(network, blockHash)
	return s.conn.Set(ctx, key, walletwatchIdempotencyDone, 0).Err()
}

// Ensure the client satisfies the IdempotencyGuard interface at compile time.
var _ walletwatch.IdempotencyGuard = new(client)
