package blockwatch

import "context"

type Block struct {
	Number       int64
	Transactions []Transaction
}

type BlockStorage interface {
	SetLatestBlockNumber(ctx context.Context, blockNumber int64) error
	GetLatestBlockNumber(ctx context.Context) (int64, error)
}
