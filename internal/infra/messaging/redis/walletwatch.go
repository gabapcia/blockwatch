package redis

import (
	"context"
	"encoding/json"

	"github.com/gabapcia/blockwatch/internal/walletwatch"

	"github.com/redis/go-redis/v9"
)

type WalletwatchTransactionNotifier = walletwatch.TransactionNotifier

// walletwatchTransactionNotifier implements walletwatch.TransactionNotifier by
// publishing transaction notifications to a Redis Stream.
type walletwatchTransactionNotifier struct {
	conn   *redis.Client // Redis client connection
	stream string        // Name of the Redis stream to publish transaction events
}

// AsWalletwatchTransactionNotifier returns a new instance of walletwatchTransactionNotifier,
// configured to publish to the given Redis Stream.
//
// Parameters:
//   - stream: the name of the Redis Stream to which notifications will be sent.
func (c *client) AsWalletwatchTransactionNotifier(stream string) walletwatch.TransactionNotifier {
	return &walletwatchTransactionNotifier{
		conn:   c.conn,
		stream: stream,
	}
}

// makeNotifyTransactionsMessage serializes the list of transactions into a
// JSON-compatible structure suitable for insertion into a Redis Stream.
//
// It returns a flat map[string]any representing the message fields,
// or an error if the serialization fails.
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

// NotifyTransactions sends a list of transactions related to a specific wallet and network
// to the configured Redis Stream. The message includes the wallet address, network name,
// and the list of transactions as a JSON array.
//
// This method implements walletwatch.TransactionNotifier.
func (c *walletwatchTransactionNotifier) NotifyTransactions(ctx context.Context, network, wallet string, txs []walletwatch.Transaction) error {
	values, err := makeNotifyTransactionsMessage(network, wallet, txs)
	if err != nil {
		return err
	}

	return c.conn.XAdd(ctx, &redis.XAddArgs{
		Stream: c.stream,
		ID:     "*",
		Values: values,
	}).Err()
}

// Compile-time check to ensure walletwatchTransactionNotifier implements walletwatch.TransactionNotifier.
var _ walletwatch.TransactionNotifier = (*walletwatchTransactionNotifier)(nil)
