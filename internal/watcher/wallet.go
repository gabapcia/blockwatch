package watcher

import "context"

// WalletStorage defines methods to index transactions by wallet address.
//
// Implementations receive a slice of Transaction for a specific network and
// return a map where each key is a watched wallet address and the value is
// the list of transaction IDs in which that address appears.
type WalletStorage interface {
	// GetTransactionsByWallet scans the provided transactions (txs) for the given
	// network (e.g., "ethereum", "solana") and returns a mapping from each watched
	// wallet address to the slice of transaction IDs where that address occurs.
	//
	// ctx controls cancellation and timeouts for any underlying I/O operations.
	GetTransactionsByWallet(ctx context.Context, network string, txs []Transaction) (map[string][]string, error)
}
