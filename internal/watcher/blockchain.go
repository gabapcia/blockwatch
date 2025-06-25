package watcher

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/pkg/types"
)

// BlockchainEvent represents an event emitted by the blockchain watcher.
// It always includes the block number that was processed, and may include
// either the full block data or an error if retrieval failed.
type BlockchainEvent struct {
	Height types.Hex // block height (always set)
	Block  Block     // block contents (zero value if Error is set)
	Err    error     // any error encountered (nil on success)
}

// Blockchain streams new blocks starting at a given block number.
type Blockchain interface {
	// Subscribe begins streaming blocks from fromBlock (inclusive).
	// If fromBlock is the zero value, the implementation should fetch
	// the latest known block and begin streaming from there.
	//
	// It returns a receive-only channel of BlockchainEvent. The channel
	// is closed when ctx is canceled.
	Subscribe(ctx context.Context, fromBlock types.Hex) (<-chan BlockchainEvent, error)
}
