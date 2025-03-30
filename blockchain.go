package blockwatch

import "context"

type Blockchain interface {
	GetLatestBlockNumber(ctx context.Context) (int64, error)
	ListBlocksSince(ctx context.Context, blockNumber int64) ([]Block, error)
}
