package memory

import (
	"sync"

	"github.com/gabapcia/blockwatch"
)

// storage is an in-memory implementation for storing the current block number
// and a mapping from addresses to their associated blockchain transactions.
// It uses a read-write mutex to ensure safe concurrent access.
type storage struct {
	// mu protects access to currentBlockNumber and addressesTransactions.
	mu sync.RWMutex

	// currentBlockNumber holds the latest processed block number.
	currentBlockNumber int64

	// addressesTransactions maps a blockchain address to a slice of transactions.
	addressesTransactions map[string][]blockwatch.Transaction
}

// New creates and returns a new instance of storage.
// It initializes currentBlockNumber to 0 and prepares an empty map for addressesTransactions.
func New() *storage {
	return &storage{
		currentBlockNumber:    0,
		addressesTransactions: make(map[string][]blockwatch.Transaction),
	}
}
