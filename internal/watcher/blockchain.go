package watcher

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/pkg/types"
)

// BlockchainEvent represents either a newly fetched block or an error that occurred while polling.
type BlockchainEvent struct {
	Block Block // the block that was retrieved
	Err   error // any error encountered during retrieval
}

// Blockchain streams new blocks starting at a given block number.
type Blockchain interface {
	// Subscribe begins streaming blocks from fromBlock (inclusive).
	// If fromBlock is the zero value, the implementation should fetch
	// the latest known block and begin streaming from there.
	// It returns a receive-only channel of BlockchainEvent; the channel is closed when ctx is canceled.
	Subscribe(ctx context.Context, fromBlock types.Hex) (<-chan BlockchainEvent, error)
}
