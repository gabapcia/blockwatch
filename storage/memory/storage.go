package memory

import (
	"sync"

	"github.com/gabapcia/blockwatch"
)

type storage struct {
	mu sync.RWMutex

	currentBlockID        int64
	addressesTransactions map[string][]blockwatch.Transaction
}

func New() *storage {
	return &storage{
		currentBlockID:        0,
		addressesTransactions: make(map[string][]blockwatch.Transaction),
	}
}
