package txwatcher

import (
	"context"
	"time"
)

// defaultMaxProcessingTime defines the default timeout for processing a single block.
const defaultMaxProcessingTime = 5 * time.Minute

// Service defines the interface for watching blocks and notifying on relevant transactions.
//
// Implementations are responsible for analyzing transactions in a block,
// filtering for opted-in wallets, and sending notifications when matches are found.
type Service interface {
	// NotifyWatchedTransactions processes the given block and emits transaction
	// notifications for any watched wallets involved in its transactions.
	//
	// The provided context is used to control timeouts and cancellation.
	// Errors should reflect permanent or transient failures during processing.
	NotifyWatchedTransactions(ctx context.Context, block Block) error
}

// service is the concrete implementation of the Service interface.
// It coordinates wallet lookups, idempotency checks, and transaction notifications.
type service struct {
	maxProcessingTime time.Duration
	idempotencyGuard  IdempotencyGuard

	walletStorage       WalletStorage
	transactionNotifier TransactionNotifier
}

// Ensure service implements the Service interface at compile time.
var _ Service = (*service)(nil)

// config holds optional configuration values for constructing the service.
type config struct {
	maxProcessingTime time.Duration
	idempotencyGuard  IdempotencyGuard
}

// Option defines a functional option for configuring the Service during creation.
type Option func(*config)

// New creates a new transaction watcher service that monitors blocks
// and notifies when watched wallets are involved in transactions.
//
// Parameters:
//   - ws: implementation of WalletStorage to filter relevant transactions.
//   - tn: implementation of TransactionNotifier to emit transaction events.
//   - opts: optional configurations (e.g., max processing time, idempotency guard).
//
// Defaults:
//   - maxProcessingTime: 5 minutes
//   - idempotencyGuard: nopIdempotencyGuard (no deduplication)
func New(ws WalletStorage, tn TransactionNotifier, opts ...Option) *service {
	cfg := config{
		maxProcessingTime: defaultMaxProcessingTime,
		idempotencyGuard:  nopIdempotencyGuard{},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &service{
		maxProcessingTime:   cfg.maxProcessingTime,
		idempotencyGuard:    cfg.idempotencyGuard,
		walletStorage:       ws,
		transactionNotifier: tn,
	}
}

// WithMaxProcessingTime overrides the default processing timeout for blocks.
//
// Use this option to define how long the service should spend processing
// each block before timing out.
func WithMaxProcessingTime(d time.Duration) Option {
	return func(c *config) {
		c.maxProcessingTime = d
	}
}

// WithIdempotencyGuard sets a custom implementation of IdempotencyGuard
// to prevent duplicate block processing.
//
// This is useful for avoiding reprocessing in retry scenarios or across service restarts.
func WithIdempotencyGuard(g IdempotencyGuard) Option {
	return func(c *config) {
		c.idempotencyGuard = g
	}
}
