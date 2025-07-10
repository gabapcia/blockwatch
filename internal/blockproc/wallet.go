package blockproc

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/pkg/flow/chflow"
	"github.com/gabapcia/blockwatch/internal/pkg/logger"
	"github.com/gabapcia/blockwatch/internal/walletwatch"
)

// mapObservedToWalletBlock converts a chainstream.ObservedBlock into a walletwatch.Block,
// allowing cross-module compatibility while preserving clear separation of concerns.
//
// This mapping enables the block to be processed by wallet-related services without
// introducing direct dependencies or domain coupling.
func mapObservedToWalletBlock(b chainstream.ObservedBlock) walletwatch.Block {
	transactions := make([]walletwatch.Transaction, len(b.Transactions))
	for i, tx := range b.Transactions {
		transactions[i] = walletwatch.Transaction(tx)
	}

	return walletwatch.Block{
		Network:      b.Network,
		Height:       b.Height,
		Hash:         b.Hash,
		Transactions: transactions,
	}
}

// handleWalletActivity processes incoming blockchain blocks and applies all wallet-related logic.
//
// This function listens to the given channel of ObservedBlock values and, for each received block,
// it triggers NotifyWatchedTransactions to notify any opted-in wallet observers.
//
// Note: If additional wallet-specific processing steps are introduced in the future
// (e.g., alerts, analytics, metrics, or workflow triggers), they should be added
// directly within this function to maintain centralized handling of wallet activity logic.
func (s *service) handleWalletActivity(ctx context.Context, blocksCh <-chan chainstream.ObservedBlock) {
	for {
		block, ok := chflow.Receive(ctx, blocksCh)
		if !ok {
			return
		}

		if err := s.walletwatch.NotifyWatchedTransactions(ctx, mapObservedToWalletBlock(block)); err != nil {
			logger.Error(ctx, "error notifying wallet transactions", "error", err)
		}
	}
}

// startHandleWalletActivity launches the wallet activity processing loop in a separate goroutine.
//
// It continuously consumes blocks from the given channel and delegates their handling
// to the handleWalletActivity function.
//
// This function should be called once during system startup to activate all wallet-related workflows.
func (s *service) startHandleWalletActivity(ctx context.Context, blocksCh <-chan chainstream.ObservedBlock) {
	go s.handleWalletActivity(ctx, blocksCh)
}
