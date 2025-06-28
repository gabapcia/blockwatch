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
