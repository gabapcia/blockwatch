package blockwatch

import (
	"context"
	"sync"
)

// memoryStorage is an in-memory implementation for storing the current block number
// and a mapping from addresses to their associated blockchain transactions.
// It uses a read-write mutex to ensure safe concurrent access.
type memoryStorage struct {
	// mu protects access to currentBlockNumber and addressesTransactions.
	mu sync.RWMutex

	// currentBlockNumber holds the latest processed block number.
	currentBlockNumber int64

	// addressesTransactions maps a blockchain address to a slice of transactions.
	addressesTransactions map[string][]Transaction
}

// NewMemoryStorage creates and returns a new instance of memoryStorage.
// It initializes currentBlockNumber to 0 and prepares an empty map for addressesTransactions.
func NewMemoryStorage() *memoryStorage {
	return &memoryStorage{
		currentBlockNumber:    0,
		addressesTransactions: make(map[string][]Transaction),
	}
}

// SaveAddress saves a new blockchain address into the storage for tracking transactions.
// If the address is already saved, it returns blockwatch.ErrAddressAlreadySaved.
// It uses a write lock to ensure safe concurrent access while modifying the storage.
func (s *memoryStorage) SaveAddress(ctx context.Context, address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.addressesTransactions[address]; ok {
		return ErrAddressAlreadySaved
	}

	s.addressesTransactions[address] = make([]Transaction, 0)
	return nil
}

// ListAddresses returns a slice containing all the saved addresses from the storage.
// It acquires a read lock to ensure safe concurrent access while reading the storage.
func (s *memoryStorage) ListAddresses(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	addresses := make([]string, 0)
	for address := range s.addressesTransactions {
		addresses = append(addresses, address)
	}

	return addresses, nil
}

// Ensure that *memoryStorage implements the AddressStorage interface.
var _ AddressStorage = (*memoryStorage)(nil)

// SetLatestBlockNumber sets the current block number in the in-memory storage.
// It acquires a write lock to ensure safe concurrent access while updating the block number.
// The provided context is not used in this implementation but is part of the interface contract.
func (s *memoryStorage) SetLatestBlockNumber(ctx context.Context, blockNumber int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentBlockNumber = blockNumber
	return nil
}

// GetLatestBlockNumber retrieves the latest block number from the in-memory storage.
// It acquires a read lock to ensure safe concurrent access while reading the block number.
// The context parameter is included to satisfy the interface requirements.
func (s *memoryStorage) GetLatestBlockNumber(ctx context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.currentBlockNumber, nil
}

// Ensure that *memoryStorage implements the BlockStorage interface.
var _ BlockStorage = (*memoryStorage)(nil)

// SaveAddressTransactions saves a list of transactions for a specific address in the in-memory storage.
// It acquires a write lock to ensure safe concurrent access. If the address does not yet have any stored transactions,
// an empty slice is initialized before appending the new transactions. On success, it returns nil.
func (s *memoryStorage) SaveAddressTransactions(ctx context.Context, address string, transactions []Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	transactionsList, ok := s.addressesTransactions[address]
	if !ok {
		s.addressesTransactions[address] = make([]Transaction, 0)
	}

	s.addressesTransactions[address] = append(transactionsList, transactions...)
	return nil
}

// ListAddressTransactions retrieves all transactions associated with the specified address from the in-memory storage.
// It uses a read lock to ensure safe concurrent access and returns a slice of transactions for the address.
func (s *memoryStorage) ListAddressTransactions(ctx context.Context, address string) ([]Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.addressesTransactions[address], nil
}

// Ensure that *memoryStorage implements the TransactionStorage interface.
var _ TransactionStorage = (*memoryStorage)(nil)
