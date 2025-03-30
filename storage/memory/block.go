package memory

import (
	"context"

	"github.com/gabapcia/blockwatch"
)

func (s *storage) SetLatestBlockNumber(ctx context.Context, blockNumber int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentBlockNumber = blockNumber
	return nil
}

func (s *storage) GetLatestBlockNumber(ctx context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.currentBlockNumber, nil
}

var _ blockwatch.BlockStorage = (*storage)(nil)
