package memory

import (
	"context"

	"github.com/gabapcia/blockwatch"
)

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

func (s *storage) ListAddressTransactions(ctx context.Context, address string) ([]blockwatch.Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.addressesTransactions[address], nil
}

var _ blockwatch.TransactionStorage = (*storage)(nil)
