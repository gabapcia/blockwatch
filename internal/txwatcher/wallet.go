package txwatcher

import "context"

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
