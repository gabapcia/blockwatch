package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gabapcia/blockwatch/internal/walletwatch"

	"github.com/redis/go-redis/v9"
)

// walletwatchKeyPrefix is the Redis key namespace used to store idempotency entries
// related to wallet transaction watching. All keys will be prefixed with this value.
const walletwatchKeyPrefix = "walletwatch"

// notifyTransactionsStreamKey returns the Redis key used for publishing
// new transactions detected for a wallet. The format is:
//
//	"walletwatch:streams:notify-transactions"
func notifyTransactionsStreamKey() string {
	return fmt.Sprintf("%s:streams:notify-transactions", walletwatchKeyPrefix)
}

// makeNotifyTransactionsMessage constructs a message payload for the notify-transactions
// Redis stream. It includes the network name, wallet address, and a JSON-encoded list of transactions.
//
// Each transaction includes hash, to, and from fields.
//
// Returns:
//   - A map[string]any representing the stream entry fields.
//   - An error if the JSON marshaling fails.
func makeNotifyTransactionsMessage(network, wallet string, txs []walletwatch.Transaction) (map[string]any, error) {
	transactions := make([]map[string]any, len(txs))
	for i, tx := range txs {
		transactions[i] = map[string]any{
			"hash": tx.Hash,
			"to":   tx.To,
			"from": tx.From,
		}
	}

	data, err := json.Marshal(transactions)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"network":      network,
		"wallet":       wallet,
		"transactions": string(data),
	}, nil
}

// NotifyTransactions sends a new wallet transaction event to the notify-transactions Redis stream.
//
// It serializes the list of transactions into JSON and includes it in the message along with the
// network name and wallet address. The Redis stream will be created automatically if it does not exist.
//
// Parameters:
//   - ctx: context for timeout and cancellation.
//   - network: the blockchain network name.
//   - wallet: the wallet address being monitored.
//   - txs: the list of transactions detected for the wallet.
//
// Returns:
//   - An error if the message could not be added to the Redis stream.
func (c *client) NotifyTransactions(ctx context.Context, network, wallet string, txs []walletwatch.Transaction) error {
	values, err := makeNotifyTransactionsMessage(network, wallet, txs)
	if err != nil {
		return err
	}

	stream := notifyTransactionsStreamKey()
	cmd := c.conn.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		ID:     "*",
		Values: values,
	})

	return cmd.Err()
}

// Ensure the client satisfies the walletwatch.TransactionNotifier interface at compile time.
var _ walletwatch.TransactionNotifier = new(client)

// walletwatchIdempotencyDone is the terminal value stored in Redis to indicate that
// a block has already been fully processed and should not be processed again.
const walletwatchIdempotencyDone = "done"

// walletwatchIdempotencyKey returns the Redis key used to track idempotency for a specific block
// in a given blockchain network. The format is:
//
//	"walletwatch:idempotency:<network>:<blockHash>"
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

// Ensure the client satisfies the walletwatch.IdempotencyGuard interface at compile time.
var _ walletwatch.IdempotencyGuard = new(client)
