package memory

import (
	"context"

	"github.com/gabapcia/blockwatch"
)

func (s *storage) SaveAddress(ctx context.Context, address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.addressesTransactions[address]; ok {
		return blockwatch.ErrAddressAlreadySaved
	}

	s.addressesTransactions[address] = make([]blockwatch.Transaction, 0)
	return nil
}

func (s *storage) ListAddresses(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	addresses := make([]string, 0)
	for address, _ := range s.addressesTransactions {
		addresses = append(addresses, address)
	}

	return addresses, nil
}

var _ blockwatch.AddressStorage = (*storage)(nil)
