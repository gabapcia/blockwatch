package chainwatch

import (
	"context"
	"errors"

	"github.com/gabapcia/blockwatch/internal/pkg/types"
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
