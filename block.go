package blockwatch

// Block represents a blockchain block. It contains the block number and the list
// of transactions that are included in that block.
type Block struct {
	// Number is the unique identifier for the block, typically the block height.
	Number int64
	// Transactions is a slice of Transaction objects included in the block.
	Transactions []Transaction
}

// GetCurrentBlock returns the most recent block number that has been parsed by the Monitor.
// It retrieves the current block number maintained internally in the Monitor instance.
func (m *Monitor) GetCurrentBlock() int64 {
	return m.currentBlockNumber
}
