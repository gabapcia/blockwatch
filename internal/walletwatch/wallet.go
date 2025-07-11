package walletwatch

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/pkg/logger"
	"github.com/gabapcia/blockwatch/internal/pkg/types"
)

// TransactionNotifier defines a mechanism for notifying external systems
// when transactions involving opted-in wallet addresses are detected.
//
// This interface is useful for triggering downstream processing,
// alerting users, or publishing events based on wallet activity.
type TransactionNotifier interface {
	// NotifyTransactions is invoked whenever transactions involving a watched
	// wallet address are detected.
	//
	// Parameters:
	//   - ctx: context for cancellation and timeout control.
	//   - network: blockchain network name (e.g., "ethereum", "solana").
	//   - wallet: the watched wallet address.
	//   - txs: a list of transactions associated with the wallet.
	//
	// Returns:
	//   - An error if the notification could not be delivered.
	NotifyTransactions(ctx context.Context, network, wallet string, txs []Transaction) error
}

// WalletStorage defines the contract for determining which wallet addresses
// are currently opted-in for transaction monitoring.
//
// It allows filtering a list of addresses to return only those
// actively being watched on a given blockchain network.
type WalletStorage interface {
	// FilterWatchedWallets filters the provided list of wallet addresses and returns
	// only those that are currently under watch for the specified network.
	//
	// Parameters:
	//   - ctx: context for cancellation and timeout.
	//   - network: blockchain network identifier.
	//   - addresses: list of wallet addresses to check.
	//
	// Returns:
	//   - A slice of addresses currently being monitored.
	//   - An error if the lookup fails.
	FilterWatchedWallets(ctx context.Context, network string, addresses []string) ([]string, error)
}

// getTransactionsByWallet identifies and groups transactions that involve wallets
// currently being watched (i.e., opted-in) for a specific blockchain network.
//
// This function follows a multi-step process to efficiently extract only the relevant
// transactions for watched wallets from a complete set of transactions in a block:
//
//  1. It builds a unique set of all wallet addresses that appear as senders (From)
//     or recipients (To) across all provided transactions.
//  2. It creates an index mapping each wallet address to the set of transaction hashes
//     in which it appears.
//  3. It queries WalletStorage to filter out only those wallet addresses from step 1
//     that are currently under watch for the specified blockchain network.
//  4. It constructs a final mapping where each watched wallet address is associated
//     with the full transaction objects it was involved in.
//
// This function is optimized for fast lookup, avoids redundant scanning, and guarantees
// that only watched wallets are returned.
//
// Parameters:
//   - ctx: a context that manages cancellation and deadlines.
//   - network: the blockchain network identifier (e.g., "ethereum", "solana").
//   - txs: a slice of all transactions from a newly observed block.
//
// Returns:
//   - A map where each key is a watched wallet address and the value is a slice of
//     Transaction objects involving that address.
//   - An error if the filtering of watched wallets fails.
func (s *service) getTransactionsByWallet(ctx context.Context, network string, txs []Transaction) (map[string][]Transaction, error) {
	var (
		// Set of all wallet addresses that appear as sender or recipient in the block
		walletsSet = types.NewSet[string]()

		// Mapping of transaction hash -> full Transaction struct for fast access later
		transactionsMap = make(map[string]Transaction)

		// Mapping of wallet address -> set of transaction hashes it is involved in
		transactionsByWallet = types.NewDefaultMap[string](func() types.Set[string] {
			return types.NewSet[string]()
		})
	)

	// Step 1: Index all transactions to extract involved wallets and build reverse mapping
	for _, tx := range txs {
		// Track both sender and recipient addresses
		walletsSet.Add(tx.From, tx.To)

		// Store transaction by its hash
		transactionsMap[tx.Hash] = tx

		// Associate sender with this transaction
		transactionsByWallet.Get(tx.From).Add(tx.Hash)

		// Associate recipient with this transaction
		transactionsByWallet.Get(tx.To).Add(tx.Hash)
	}

	// Step 2: Query WalletStorage to filter only addresses that are actively watched
	watchedWallets, err := s.walletStorage.FilterWatchedWallets(ctx, network, walletsSet.ToSlice())
	if err != nil {
		return nil, err
	}

	// Step 3: Build final result mapping each watched wallet to its full list of transactions
	matchingTxsByWallet := types.NewDefaultMap[string](func() []Transaction { return make([]Transaction, 0) })

	for _, address := range watchedWallets {
		// For each watched wallet, iterate over all transaction hashes it was involved in
		for txHash := range transactionsByWallet.Get(address).ToIter() {
			// Append the full transaction object to the result
			txs := matchingTxsByWallet.Get(address)
			matchingTxsByWallet.Set(address, append(txs, transactionsMap[txHash]))
		}
	}

	// Step 4: Return the result as a plain map[string][]Transaction
	return matchingTxsByWallet.ToMap(), nil
}

// notifyWatchedWalletTransactions analyzes a set of transactions to identify
// those involving watched wallets and notifies external systems accordingly.
//
// It uses WalletStorage to identify relevant addresses and TransactionNotifier
// to send the notifications.
//
// Parameters:
//   - ctx: context for cancellation and timeout.
//   - network: blockchain network name.
//   - txs: list of transactions from the block.
//
// Returns:
//   - An error if processing or notification fails.
func (s *service) notifyWatchedWalletTransactions(ctx context.Context, network string, txs []Transaction) error {
	matchingTxsByWallet, err := s.getTransactionsByWallet(ctx, network, txs)
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

// NotifyWatchedTransactions is the primary entry point for processing a newly observed blockchain block
// to identify and notify wallet-related transactions involving opted-in addresses.
//
// This function performs three core responsibilities:
//
//  1. **Idempotency Enforcement**:
//     It first attempts to claim the block using an IdempotencyGuard, which ensures that no two processes
//     handle the same block concurrently. This is crucial for distributed systems or retry-prone workflows.
//     - If the block is already being processed, it returns ErrStillInProgress.
//     - If the block has already been processed successfully, it returns ErrAlreadyFinished.
//     These sentinel errors allow upstream logic to decide whether to skip, wait, or retry.
//
//  2. **Wallet-Based Transaction Extraction and Notification**:
//     Once the block is successfully claimed, it invokes internal logic to:
//     a. Determine which transactions in the block involve wallets that are being watched.
//     b. Notify an external sink or system (via TransactionNotifier) for each wallet that matches,
//     providing all relevant transactions.
//
//  3. **Completion Marking**:
//     After a successful notification pass, the block is marked as completed via IdempotencyGuard.
//     This prevents the block from being processed again, even after system restarts or retries.
//     - If marking fails, the error is logged, but processing is still considered successful.
//
// Parameters:
//   - ctx: context used to propagate cancellation signals or enforce deadlines.
//   - block: a fully hydrated blockchain block, including network metadata and all transactions.
//
// Returns:
//   - nil if the block was processed and marked as completed without any critical errors.
//   - ErrStillInProgress or ErrAlreadyFinished if the block cannot be claimed for processing.
//   - Any other error encountered during extraction, notification, or coordination phases.
//
// Note:
//   - If the notification step fails, the function returns the error immediately (processing halts).
//   - If the final marking step fails, it logs the issue but does not propagate the error further.
//     This design favors at-most-once guarantees over retrying completed work.
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
