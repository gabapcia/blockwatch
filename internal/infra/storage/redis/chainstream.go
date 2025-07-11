package redis

import (
	"context"
	"errors"
	"fmt"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/pkg/types"

	"github.com/redis/go-redis/v9"
)

// chainstreamKeyPrefix is the namespace prefix for all keys related to the chainstream checkpointing system.
const chainstreamKeyPrefix = "chainstream"

// chainstreamCheckpointKey constructs the Redis key used to store the latest processed block height
// for a specific blockchain network. The format is:
//
//	"chainstream:checkpoint:<network>"
func chainstreamCheckpointKey(network string) string {
	return fmt.Sprintf("%s:checkpoint:%s", chainstreamKeyPrefix, network)
}

// SaveCheckpoint persists the most recent block height processed for a given network.
//
// This allows the chainstream system to resume from the correct position after restarts.
// The checkpoint is stored as a Redis key with no expiration.
//
// Parameters:
//   - ctx: context for timeout and cancellation.
//   - network: the blockchain network name (e.g., "ethereum", "solana").
//   - height: the latest processed block height, encoded as a types.Hex value.
//
// Returns:
//   - An error if the Redis operation fails.
func (c *client) SaveCheckpoint(ctx context.Context, network string, height types.Hex) error {
	key := chainstreamCheckpointKey(network)
	return c.conn.Set(ctx, key, height, 0).Err()
}

// LoadLatestCheckpoint retrieves the most recently saved checkpoint for the given network.
//
// If no checkpoint exists yet, it returns chainstream.ErrNoCheckpointFound.
// Otherwise, the value is parsed into a types.Hex.
//
// Parameters:
//   - ctx: context for timeout and cancellation.
//   - network: the blockchain network name.
//
// Returns:
//   - The last known block height (types.Hex) or an error if retrieval or parsing fails.
func (c *client) LoadLatestCheckpoint(ctx context.Context, network string) (types.Hex, error) {
	key := chainstreamCheckpointKey(network)

	val, err := c.conn.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = chainstream.ErrNoCheckpointFound
		}

		return "", err
	}

	return types.HexFromString(val)
}

// Compile-time assertion to ensure client implements the CheckpointStorage interface.
var _ chainstream.CheckpointStorage = new(client)
