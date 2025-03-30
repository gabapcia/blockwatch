package memory

import (
	"sync"

	"github.com/gabapcia/blockwatch"
)

type storage struct {
	mu sync.RWMutex

	currentBlockNumber    int64
	addressesTransactions map[string][]blockwatch.Transaction
}

func New() *storage {
	return &storage{
		currentBlockNumber:    0,
		addressesTransactions: make(map[string][]blockwatch.Transaction),
	}
}
