package chainstream

import (
	"context"
	"errors"

	"github.com/gabapcia/blockwatch/internal/pkg/logger"
	"github.com/gabapcia/blockwatch/internal/pkg/types"
	"github.com/gabapcia/blockwatch/internal/pkg/x/chflow"
)

// ErrNoCheckpointFound is returned by LoadLatestCheckpoint when no checkpoint
// has been saved yet for the requested network.
var ErrNoCheckpointFound = errors.New("no checkpoint found for network")

// CheckpointStorage persists and retrieves the latest processed block height
// for each blockchain network.
type CheckpointStorage interface {
	// SaveCheckpoint records the given block height as the latest checkpoint
	// for the specified network (e.g., "ethereum", "polygon").
	//
	// Calling SaveCheckpoint multiple times for the same network should
	// overwrite any previous checkpoint.
	//
	// ctx controls cancellation and deadlines for any underlying I/O.
	SaveCheckpoint(ctx context.Context, network string, height types.Hex) error

	// LoadLatestCheckpoint returns the most recent block height saved for
	// the specified network.
	//
	// If no checkpoint exists for the network, LoadLatestCheckpoint should
	// return ErrNoCheckpointFound.
	//
	// ctx controls cancellation and deadlines for any underlying I/O.
	LoadLatestCheckpoint(ctx context.Context, network string) (types.Hex, error)
}

// checkpointAndForward ensures checkpoint persistence for each received ObservedBlock
// before forwarding it directly to the output channel.
//
// It continuously reads from blockIn until the channel is closed or the context is canceled.
// For each block received:
//  1. A checkpoint is attempted using the configured CheckpointStorage.
//  2. If the checkpoint save fails, the error is logged, but the block is still processed.
//  3. The block is forwarded directly to the blockOut channel.
//
// Failures to persist the checkpoint do not interrupt processing — the system logs the failure
// and continues to forward the block.
//
// This function exits cleanly when:
//   - The input channel is closed,
//   - The context is canceled, or
//   - Sending to blockOut fails due to context cancellation.
//
// The output channel is not closed by this function — the caller is responsible for managing its lifecycle.
func (s *service) checkpointAndForward(ctx context.Context, blockIn <-chan ObservedBlock, blockOut chan<- ObservedBlock) {
	for {
		observedBlock, ok := chflow.Receive(ctx, blockIn)
		if !ok {
			return
		}

		// Save checkpoint before forwarding the block
		if err := s.checkpointStorage.SaveCheckpoint(ctx, observedBlock.Network, observedBlock.Height); err != nil {
			// Log the error but continue processing - checkpoint failure shouldn't stop block processing
			logger.Error(ctx, "failed to save checkpoint",
				"block.network", observedBlock.Network,
				"block.height", observedBlock.Height,
				"error", err,
			)
		}

		// Forward the transformed output to the final output channel
		if ok := chflow.Send(ctx, blockOut, observedBlock); !ok {
			return
		}
	}
}

// startCheckpointAndForward starts the checkpointAndForward function in a new goroutine,
// allowing asynchronous processing of ObservedBlock values.
//
// This stage sits between internal block ingestion and final delivery to clients.
// It ensures that checkpoints are persisted for each block before forwarding it.
//
// Parameters:
//   - ctx: controls cancellation of the processing goroutine.
//   - blockIn: channel from which raw ObservedBlocks are consumed.
//   - blockOut: channel to which ObservedBlock values are sent.
//
// This function returns immediately and does not block the caller.
// The processing loop will continue running in the background until:
//   - The context is canceled, or
//   - The input channel is closed.
//
// It is the caller's responsibility to close `blockIn` and `blockOut` at the appropriate time.
func (s *service) startCheckpointAndForward(ctx context.Context, blockIn <-chan ObservedBlock, blockOut chan<- ObservedBlock) {
	go s.checkpointAndForward(ctx, blockIn, blockOut)
}

// nopCheckpoint is a no-op implementation of CheckpointStorage.
// It performs no persistence and always returns ErrNoCheckpointFound
// when loading checkpoints.
type nopCheckpoint struct{}

// SaveCheckpoint is a no-op. It accepts the checkpoint input but does not store anything.
func (nopCheckpoint) SaveCheckpoint(_ context.Context, _ string, _ types.Hex) error {
	return nil
}

// LoadLatestCheckpoint always returns ErrNoCheckpointFound, as no state is persisted.
func (nopCheckpoint) LoadLatestCheckpoint(_ context.Context, _ string) (types.Hex, error) {
	return "", ErrNoCheckpointFound
}
