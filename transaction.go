package blockwatch

// Transaction represents a blockchain transaction.
// It contains the block number where the transaction was recorded,
// the unique hash identifier of the transaction, the sender's address,
// the receiver's address, and the value transferred.
type Transaction struct {
	// BlockNumber is the number of the block in which the transaction was included.
	BlockNumber int64
	// Hash is the unique identifier of the transaction.
	Hash string
	// From is the address that initiated the transaction.
	From string
	// To is the address that received the transaction.
	To string
	// Value is the amount transferred in the transaction.
	Value string
}

// GetTransactions returns all transactions associated with the specified address.
// It retrieves the transactions from the internal mapping of addresses to transactions in the Monitor.
func (m *Monitor) GetTransactions(address string) []Transaction {
	return m.transactiosByAddress[address]
}
