package memory

import (
	"context"

	"github.com/gabapcia/blockwatch"
)

// SetLatestBlockNumber sets the current block number in the in-memory storage.
// It acquires a write lock to ensure safe concurrent access while updating the block number.
// The provided context is not used in this implementation but is part of the interface contract.
func (s *storage) SetLatestBlockNumber(ctx context.Context, blockNumber int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentBlockNumber = blockNumber
	return nil
}

// GetLatestBlockNumber retrieves the latest block number from the in-memory storage.
// It acquires a read lock to ensure safe concurrent access while reading the block number.
// The context parameter is included to satisfy the interface requirements.
func (s *storage) GetLatestBlockNumber(ctx context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.currentBlockNumber, nil
}

// Ensure that *storage implements the blockwatch.BlockStorage interface.
var _ blockwatch.BlockStorage = (*storage)(nil)
