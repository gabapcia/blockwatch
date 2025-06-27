package watcher

import "context"

type BlockNotifier interface {
	NotifyBlockProcessed(ctx context.Context, network string, block Block) error
}

type TransactionNotifier interface {
	NotifyTransaction(ctx context.Context, network, wallet string, tx Transaction) error
}
