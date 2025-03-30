package memory

import (
	"context"

	"github.com/gabapcia/blockwatch"
)

// SaveAddress saves a new blockchain address into the storage for tracking transactions.
// If the address is already saved, it returns blockwatch.ErrAddressAlreadySaved.
// It uses a write lock to ensure safe concurrent access while modifying the storage.
func (s *storage) SaveAddress(ctx context.Context, address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.addressesTransactions[address]; ok {
		return blockwatch.ErrAddressAlreadySaved
	}

	s.addressesTransactions[address] = make([]blockwatch.Transaction, 0)
	return nil
}

// ListAddresses returns a slice containing all the saved addresses from the storage.
// It acquires a read lock to ensure safe concurrent access while reading the storage.
func (s *storage) ListAddresses(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	addresses := make([]string, 0)
	for address := range s.addressesTransactions {
		addresses = append(addresses, address)
	}

	return addresses, nil
}

// Ensure that *storage implements the blockwatch.AddressStorage interface.
var _ blockwatch.AddressStorage = (*storage)(nil)
