package txwatcher

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrStillInProgress indicates that the block is currently being processed by another instance.
	ErrStillInProgress = errors.New("processing still in progress")

	// ErrAlreadyFinished indicates that the block has already been processed successfully.
	ErrAlreadyFinished = errors.New("processing already finished")
)

// IdempotencyGuard defines a coordination mechanism that ensures each block
// is scanned for wallet-related transactions exactly once within a time window.
//
// This interface is typically implemented using durable or distributed storage
// (e.g., Redis, PostgreSQL, DynamoDB) to prevent redundant processing in
// multi-instance or retry-prone environments.
type IdempotencyGuard interface {
	// ClaimBlockForTxWatch attempts to claim exclusive rights to process a given block
	// for wallet transaction watching. It prevents duplicate work across parallel workers.
	//
	// If the block is already in progress or marked as completed, it returns a sentinel error:
	//   - ErrStillInProgress: if another process is already scanning the block.
	//   - ErrAlreadyFinished: if the block was fully processed before.
	//
	// The claim is time-bound (via ttl) to avoid deadlocks in case the process crashes.
	//
	// Parameters:
	//   - ctx: standard context for cancellation and timeouts.
	//   - network: blockchain network name (e.g., "ethereum", "solana").
	//   - blockHash: unique block identifier used as the idempotency key.
	//   - ttl: duration after which an in-progress claim can be retried by another process.
	//
	// Returns:
	//   - nil if the claim was successfully acquired.
	//   - ErrStillInProgress or ErrAlreadyFinished as expected control flow signals.
	//   - Any other error indicates a failure in the underlying guard mechanism.
	ClaimBlockForTxWatch(ctx context.Context, network, blockHash string, ttl time.Duration) error

	// MarkBlockTxWatchComplete signals that the given block was fully scanned
	// and processed for wallet activity, making future claims unnecessary.
	//
	// Implementations must persist this status to ensure completed blocks
	// are never reprocessed, even after system restarts.
	//
	// Parameters:
	//   - ctx: context for cancellation and timeout control.
	//   - network: blockchain network name.
	//   - blockHash: hash of the processed block.
	//
	// Returns:
	//   - nil on success, or an error if the state could not be persisted.
	MarkBlockTxWatchComplete(ctx context.Context, network, blockHash string) error
}

// nopIdempotencyGuard is a no-op IdempotencyGuard that treats every block as unprocessed.
//
// It always allows processing and never stores state. This is intended only for
// local development, testing, or scenarios where duplicate processing is acceptable.
type nopIdempotencyGuard struct{}

// Ensure compile-time compliance with the IdempotencyGuard interface.
var _ IdempotencyGuard = (*nopIdempotencyGuard)(nil)

// ClaimBlockForTxWatch in nopIdempotencyGuard always returns nil, allowing all claims.
func (nopIdempotencyGuard) ClaimBlockForTxWatch(ctx context.Context, network, blockHash string, ttl time.Duration) error {
	return nil
}

// MarkBlockTxWatchComplete in nopIdempotencyGuard does nothing.
func (nopIdempotencyGuard) MarkBlockTxWatchComplete(ctx context.Context, network, blockHash string) error {
	return nil
}
