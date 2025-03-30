package memory

import (
	"context"

	"github.com/gabapcia/blockwatch"
)

// SaveAddressTransactions saves a list of transactions for a specific address in the in-memory storage.
// It acquires a write lock to ensure safe concurrent access. If the address does not yet have any stored transactions,
// an empty slice is initialized before appending the new transactions. On success, it returns nil.
func (s *storage) SaveAddressTransactions(ctx context.Context, address string, transactions []blockwatch.Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	transactionsList, ok := s.addressesTransactions[address]
	if !ok {
		s.addressesTransactions[address] = make([]blockwatch.Transaction, 0)
	}

	s.addressesTransactions[address] = append(transactionsList, transactions...)
	return nil
}

// ListAddressTransactions retrieves all transactions associated with the specified address from the in-memory storage.
// It uses a read lock to ensure safe concurrent access and returns a slice of transactions for the address.
func (s *storage) ListAddressTransactions(ctx context.Context, address string) ([]blockwatch.Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.addressesTransactions[address], nil
}

// Ensure that *storage implements the blockwatch.TransactionStorage interface.
var _ blockwatch.TransactionStorage = (*storage)(nil)
