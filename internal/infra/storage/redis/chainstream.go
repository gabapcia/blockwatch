package redis

import (
	"context"
	"errors"
	"fmt"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/pkg/logger"
	"github.com/gabapcia/blockwatch/internal/pkg/types"

	"github.com/redis/go-redis/v9"
)

// chainstreamKeyPrefix is the namespace prefix for all keys related to the chainstream checkpointing system.
const chainstreamKeyPrefix = "chainstream"

// makeBlockDispatchFailureMessage converts a BlockDispatchFailure into a map[string]any
// suitable for sending to Redis streams.
func makeBlockDispatchFailureMessage(dispatchFailure chainstream.BlockDispatchFailure) map[string]any {
	return map[string]any{
		"network": dispatchFailure.Network, // name of the blockchain network (e.g., "ethereum")
		"height":  dispatchFailure.Height,  // block height that failed to be dispatched
		"errors":  dispatchFailure.Errors,  // slice of all errors encountered during dispatch and retry attempts
	}
}

// BuildChainstreamDispatchFailureHandler returns a DispatchFailureHandler that logs block dispatch
// failures to a Redis stream.
//
// Each failure is added as an entry to the given stream, with fields for network, height, and errors.
// This function does not create the stream and expects it to already exist in Redis.
// If the Redis operation fails, the error is logged.
//
// Parameters:
//   - stream: the Redis stream name to write failures to.
//
// Returns:
//   - A function matching the chainstream.DispatchFailureHandler signature.
func (c *client) BuildChainstreamDispatchFailureHandler(stream string) chainstream.DispatchFailureHandler {
	return func(ctx context.Context, dispatchFailure chainstream.BlockDispatchFailure) {
		cmd := c.conn.XAdd(ctx, &redis.XAddArgs{
			Stream:     stream,
			ID:         "*",
			NoMkStream: true,
			Values:     makeBlockDispatchFailureMessage(dispatchFailure),
		})
		if err := cmd.Err(); err != nil {
			logger.Error(ctx, "stream xadd failed",
				"redis.stream", stream,
				"error", err,
			)
		}
	}
}

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
