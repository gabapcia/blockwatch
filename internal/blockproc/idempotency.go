package blockproc

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/pkg/types"
)

// IdempotencyGuard defines a mechanism to ensure that blockchain blocks
// are processed exactly once, even in the presence of retries, concurrent
// processing attempts, or application restarts.
//
// This interface is responsible for acquiring exclusive rights to process a block
// and marking it as completed once successfully handled.
//
// A ProcessingID is typically a deterministic identifier derived from the block's
// network and hash (e.g., using SHA-256 over "<network>:<blockHash>").
//
// The implementation is expected to support TTL semantics: if a lock is acquired
// but processing is not finalized within a reasonable time (e.g., due to crash or timeout),
// the lock may be considered expired and re-acquirable.
type IdempotencyGuard interface {
	// TryAcquireProcessingLock attempts to acquire a processing lock for the given block.
	//
	// If the block has never been processed or the lock has expired, this should return true.
	// If the block is currently being processed by another process (or recently was), this returns false.
	//
	// Parameters:
	//   - ctx: provides cancellation and timeout context.
	//   - processingID: the unique ID of the block to process (e.g., SHA-256 of network + hash).
	//   - network: the blockchain network name (e.g., "ethereum").
	//   - height: the height of the block being evaluated.
	//
	// Returns:
	//   - acquired: true if processing can proceed, false if it should be skipped or retried later.
	//   - err: any underlying error accessing the guard mechanism (e.g., storage or network).
	TryAcquireProcessingLock(ctx context.Context, processingID, network string, height types.Hex) (bool, error)

	// MarkProcessed finalizes the processing of a block, indicating that it has
	// been fully and successfully handled.
	//
	// After this call, the processing lock should be cleared or converted into a
	// durable marker that prevents future reprocessing of this block.
	//
	// This method must be called only after all processing steps are completed.
	//
	// Parameters:
	//   - ctx: provides cancellation and timeout context.
	//   - processingID: the unique ID of the block to mark as processed.
	//   - network: the blockchain network name.
	//   - height: the height of the processed block.
	//
	// Returns:
	//   - err: any error that occurred while persisting the "processed" state.
	MarkProcessed(ctx context.Context, processingID, network string, height types.Hex) error
}
