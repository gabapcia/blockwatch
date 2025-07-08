package walletwatch

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/pkg/logger"
)

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

// WalletStorage defines the contract for identifying watched wallet addresses
// involved in a given set of blockchain transactions.
//
// Implementations are responsible for determining which wallet addresses are
// being actively watched (opted-in) and returning only the subset of those
// that appear in the provided transactions.
type WalletStorage interface {
	// GetTransactionsByWallet scans the given slice of transactions (txs) for the specified
	// blockchain network (e.g., "ethereum", "solana") and returns a mapping where:
	//
	//   - Each key is a wallet address that is actively watched.
	//   - Each value is a list of Transaction objects where that wallet was involved,
	//     either as sender (From) or recipient (To).
	//
	// Only transactions involving watched wallets should be included in the result.
	//
	// The context parameter controls cancellation and timeouts for any underlying operations
	// such as database queries or remote lookups.
	GetTransactionsByWallet(ctx context.Context, network string, txs []Transaction) (map[string][]Transaction, error)
}

// notifyWatchedWalletTransactions identifies transactions involving watched wallet addresses
// and triggers notifications for each match.
//
// It queries the WalletStorage to determine which wallet addresses from the provided transactions
// are currently being watched for the given blockchain network. For each transaction involving a
// watched wallet, it calls the TransactionNotifier to emit a notification.
//
// Parameters:
//   - ctx: controls cancellation and timeout.
//   - network: the blockchain network identifier (e.g., "ethereum", "solana").
//   - txs: the full list of transactions from a newly observed block.
//
// Returns:
//   - An error if either the WalletStorage lookup or any notification fails.
//
// Behavior:
//   - If GetTransactionsByWallet returns an error, the function returns early with that error.
//   - If any call to NotifyTransaction fails, processing stops and the error is returned.
//   - Otherwise, all relevant transactions are notified successfully.
func (s *service) notifyWatchedWalletTransactions(ctx context.Context, network string, txs []Transaction) error {
	matchingTxsByWallet, err := s.walletStorage.GetTransactionsByWallet(ctx, network, txs)
	if err != nil {
		return err
	}

	for wallet, txs := range matchingTxsByWallet {
		if err := s.transactionNotifier.NotifyTransactions(ctx, network, wallet, txs); err != nil {
			return err
		}
	}

	return nil
}

// NotifyWatchedTransactions processes a blockchain block to detect and notify
// transactions involving watched wallet addresses.
//
// This function enforces idempotent processing through the configured IdempotencyGuard,
// ensuring that each block is only scanned once per TTL window, even in distributed environments.
//
// The flow is as follows:
//  1. Attempts to claim the block for processing using ClaimBlockForTxWatch.
//     - If the claim fails with ErrStillInProgress, it indicates that another process is currently handling it.
//     - If the claim fails with ErrAlreadyFinished, it indicates the block has already been processed.
//     - In both cases, the error is returned as-is so the caller can decide how to handle it.
//  2. If claimed successfully, it identifies relevant transactions via WalletStorage
//     and notifies them through the TransactionNotifier.
//  3. Upon successful notification, it marks the block as completed via MarkBlockTxWatchComplete.
//
// Parameters:
//   - ctx: context used for cancellation and timeout.
//   - block: the blockchain block containing transactions to be checked.
//
// Returns:
//   - nil on successful processing and completion.
//   - ErrStillInProgress or ErrAlreadyFinished from the IdempotencyGuard if applicable.
//   - Any other error encountered during processing or finalization.
//
// Note: Errors returned from step 3 (MarkBlockTxWatchComplete) are logged but do not prevent
// NotifyWatchedTransactions from returning nil if the main processing was successful.
func (s *service) NotifyWatchedTransactions(ctx context.Context, block Block) error {
	if err := s.idempotencyGuard.ClaimBlockForTxWatch(ctx, block.Network, block.Hash, s.maxProcessingTime); err != nil {
		return err
	}

	if err := s.notifyWatchedWalletTransactions(ctx, block.Network, block.Transactions); err != nil {
		return err
	}

	if err := s.idempotencyGuard.MarkBlockTxWatchComplete(ctx, block.Network, block.Hash); err != nil {
		logger.Error(ctx, "error marking block tx watch as complete",
			"block.network", block.Network,
			"block.hash", block.Hash,
			"block.height", block.Height,
			"error", err,
		)
	}

	return nil
}
