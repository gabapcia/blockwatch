package watcher

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/pkg/types"
)

// BlockchainEvent represents an event emitted by the blockchain watcher.
// It always includes the block height that was processed, and may include
// either the full block data or an error if retrieval failed.
type BlockchainEvent struct {
	Height types.Hex // block height (always set)
	Block  Block     // block contents (zero value if Err is set)
	Err    error     // any error encountered (nil on success)
}

// Blockchain defines a source of blockchain data.
// It supports fetching individual blocks by height and streaming new blocks.
type Blockchain interface {
	// FetchBlockByHeight retrieves the block at the specified height.
	// It returns the full Block data or an error if fetching or decoding fails.
	FetchBlockByHeight(ctx context.Context, height types.Hex) (Block, error)

	// Subscribe begins streaming blocks from fromHeight (inclusive).
	// If fromHeight is the zero value, the implementation should fetch
	// the latest known block and begin streaming from there.
	//
	// It returns a receive-only channel of BlockchainEvent. The channel
	// is closed when ctx is canceled.
	Subscribe(ctx context.Context, fromHeight types.Hex) (<-chan BlockchainEvent, error)
}
