package postgresql

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gabapcia/blockwatch/internal/infra/storage/postgresql/querier/walletwatchidempotency"
	"github.com/gabapcia/blockwatch/internal/walletwatch"

	"github.com/google/uuid"
)

// ClaimBlockForTxWatch attempts to claim a block for processing in a transactional and idempotent way.
// If no active or completed claim exists for the given (network, blockHash), it inserts a new claim.
// If a claim exists and has expired, it extends the TTL and proceeds.
// If the block is still in progress or already marked as finished, a specific error is returned.
//
// Possible return values:
//   - nil: claim was successful (either inserted or extended)
//   - ErrStillInProgress: block is currently being processed by another routine
//   - ErrAlreadyFinished: block has already been processed and marked as finished
//   - error: unexpected failure during claim or query
func (c *client) ClaimBlockForTxWatch(ctx context.Context, network, blockHash string, ttl time.Duration) error {
	// Normalize network name
	network = strings.ToUpper(network)
	now := time.Now().UTC()

	// Try to insert or extend the lock if expired.
	// If this is the first claim or the lock has expired, we claim it successfully.
	affectedRows, err := c.walletwatchIdempotency.ClaimOrExtendIfExpired(ctx, walletwatchidempotency.ClaimOrExtendIfExpiredParams{
		ID:              uuid.Must(uuid.NewV7()),
		Network:         network,
		BlockHash:       blockHash,
		InProgressUntil: now.Add(ttl),
	})
	if err != nil {
		return err
	}
	if affectedRows > 0 {
		// Successfully claimed the block
		return nil
	}

	// The block already exists and is not claimable via upsert — fetch its current state
	guard, err := c.walletwatchIdempotency.FindClaim(ctx, walletwatchidempotency.FindClaimParams{
		Network:   network,
		BlockHash: blockHash,
	})
	if err != nil {
		return err
	}

	// Determine reason for failure using clear errors
	switch {
	case guard.FinishedAt.Valid:
		return walletwatch.ErrAlreadyFinished
	case now.Before(guard.InProgressUntil):
		return walletwatch.ErrStillInProgress
	default:
		// This case should rarely happen — indicates a potential race or unexpected state
		return errors.New("block claim failed unexpectedly: no rows affected but no conflicting state detected")
	}
}

// MarkBlockTxWatchComplete marks a previously claimed block as completed.
// This operation is idempotent and only updates blocks that are still in progress.
//
// Parameters:
//   - network: blockchain network (e.g., ETHEREUM), will be normalized to uppercase
//   - blockHash: identifier of the block to finalize
//
// Returns:
//   - nil if the block was successfully marked as finished or already finished
//   - error if the update fails due to database issues
func (c *client) MarkBlockTxWatchComplete(ctx context.Context, network, blockHash string) error {
	return c.walletwatchIdempotency.MarkClaimAsCompleted(ctx, walletwatchidempotency.MarkClaimAsCompletedParams{
		Network:   strings.ToUpper(network),
		BlockHash: blockHash,
	})
}

// Compile-time assertion that client implements walletwatch.IdempotencyGuard
var _ walletwatch.IdempotencyGuard = (*client)(nil)
