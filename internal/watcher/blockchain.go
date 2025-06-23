package watcher

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/pkg/types"
)

// BlockchainEvent represents an event emitted by the blockchain watcher.
// It contains either a new block or an error encountered during processing.
type BlockchainEvent struct {
	NewBlock Block // The new block that was detected
	Error    error // Any error encountered while retrieving or decoding a block
}

// Blockchain defines the interface for a blockchain data source.
// Implementations should provide a way to stream new blocks starting from a given block number.
type Blockchain interface {
	// Listen begins listening for new blocks starting from the given block number.
	// It returns a receive-only channel of BlockchainEvent and an error, if initialization fails.
	//
	// The context can be used to cancel the listening process.
	//
	// If startFromBlockNumber is empty (zero value), the implementation should fetch
	// the latest known block and begin streaming from there.
	Listen(ctx context.Context, startFromBlockNumber types.Hex) (<-chan BlockchainEvent, error)
}
