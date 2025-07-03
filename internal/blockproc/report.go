package blockproc

import "context"

// BlockProcessedNotifier defines the contract for components interested in being notified
// when a block has been successfully processed.
//
// This interface is commonly used to trigger side effects (e.g., persisting audit logs,
// updating metrics, publishing domain events) after a block has been fully processed
// without errors.
type BlockProcessedNotifier interface {
	// NotifyBlockProcessed is called after a block has been successfully processed.
	// It receives the full BlockProcessingSuccess data, including metadata and timestamps.
	//
	// Implementations should return a non-nil error only if the notification itself fails
	// (e.g., network I/O, persistence failure).
	NotifyBlockProcessed(ctx context.Context, result BlockProcessingSuccess) error
}

// BlockProcessingFailureNotifier defines the contract for components that should be notified
// when a block processing operation ends in a terminal (non-retryable) failure.
//
// This interface is typically used to persist failure reports, trigger alerts,
// or update monitoring dashboards.
type BlockProcessingFailureNotifier interface {
	// NotifyBlockProcessingFailure is called when a block processing operation
	// has failed irrecoverably and will no longer be retried.
	//
	// The result includes contextual information like number of attempts, error history,
	// and the last error that caused the failure.
	NotifyBlockProcessingFailure(ctx context.Context, result BlockProcessingFailure) error
}

// TransactionNotifier defines a mechanism for notifying external components
// when relevant transactions have been observed involving opted-in wallets.
//
// This interface is useful for triggering downstream processing, alerting users,
// or emitting events based on wallet activity detected across different blockchain networks.
type TransactionNotifier interface {
	// NotifyTransactions is called whenever one or more transactions involving a
	// wallet that has opted in for monitoring are detected.
	//
	// Parameters:
	//   - ctx: context for cancellation and timeout control.
	//   - network: the blockchain network name (e.g., "ethereum", "solana").
	//   - wallet: the wallet address that matched the opt-in criteria.
	//   - txs: a slice of transactions associated with the wallet.
	NotifyTransactions(ctx context.Context, network, wallet string, txs []Transaction) error
}
