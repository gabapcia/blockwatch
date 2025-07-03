package txwatcher

import (
	"context"
	"time"
)

// IdempotencyGuard coordinates and deduplicates block-level transaction watching,
// ensuring that each block is only scanned once for relevant wallet activity.
//
// This interface is useful in distributed systems to prevent multiple instances
// from performing duplicate transaction scans on the same block.
//
// Typical implementations include in-memory guards for single-node deployments,
// or distributed locks (e.g., Redis-based) for multi-node coordination.
type IdempotencyGuard interface {
	// ClaimBlockForTxWatch attempts to claim the right to scan the given block
	// for watched wallet transactions.
	//
	// The method applies a TTL to the claim, so if the process crashes or fails to complete,
	// other instances may retry after the TTL expires.
	//
	// Parameters:
	//   - ctx: context for timeout and cancellation.
	//   - network: blockchain network name (e.g., "ethereum").
	//   - blockHash: unique identifier of the block to scan.
	//   - ttl: duration for which the claim is considered active.
	//
	// Returns:
	//   - true, nil: if this process has successfully claimed the block for scanning.
	//   - false, nil: if another process is already handling it or it was recently scanned.
	//   - false, error: if an unexpected error occurred (e.g., backend unavailable).
	ClaimBlockForTxWatch(ctx context.Context, network, blockHash string, ttl time.Duration) (bool, error)

	// MarkBlockTxWatchComplete records that the block has been successfully scanned
	// for transactions involving watched wallets.
	//
	// This method is optional depending on the implementation — for example, if TTL-based
	// coordination is sufficient, this may be a no-op.
	//
	// Parameters:
	//   - ctx: context for timeout and cancellation.
	//   - network: blockchain network name (e.g., "ethereum").
	//   - blockHash: hash of the block that has been successfully processed.
	//
	// Returns:
	//   - An error if the recording operation fails, nil otherwise.
	MarkBlockTxWatchComplete(ctx context.Context, network, blockHash string) error
}

// nopIdempotencyGuard is a no-op implementation of IdempotencyGuard.
// It treats all blocks as unprocessed and performs no actual coordination.
//
// Useful for testing, local development, or when idempotency is not a concern.
type nopIdempotencyGuard struct{}

// Ensure compile-time compliance with the IdempotencyGuard interface.
var _ IdempotencyGuard = (*nopIdempotencyGuard)(nil)

// ClaimBlockForTxWatch in the no-op implementation always returns true,
// indicating that every block should be scanned.
func (nopIdempotencyGuard) ClaimBlockForTxWatch(ctx context.Context, network, blockHash string, ttl time.Duration) (bool, error) {
	return true, nil
}

// MarkBlockTxWatchComplete in the no-op implementation performs no action.
func (nopIdempotencyGuard) MarkBlockTxWatchComplete(ctx context.Context, network, blockHash string) error {
	return nil
}
