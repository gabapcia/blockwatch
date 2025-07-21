package redis

import (
	"context"
	"encoding/json"

	"github.com/gabapcia/blockwatch/internal/walletwatch"

	"github.com/redis/go-redis/v9"
)

type walletwatchTransactionNotifier struct {
	conn   *redis.Client
	stream string
}

func (c *client) AsWalletwatchTransactionNotifier(stream string) *walletwatchTransactionNotifier {
	return &walletwatchTransactionNotifier{
		conn:   c.conn,
		stream: stream,
	}
}

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

var _ walletwatch.TransactionNotifier = new(walletwatchTransactionNotifier)
