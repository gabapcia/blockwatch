package blockwatch

import "context"

// Blockchain defines the interface for interacting with a blockchain.
// It provides methods for retrieving the latest block number and for listing blocks
// that have been mined since a specified block number.
type Blockchain interface {
	// GetLatestBlockNumber retrieves the most recent block number from the blockchain.
	// The method returns the block number as an int64 along with any error encountered.
	GetLatestBlockNumber(ctx context.Context) (int64, error)

	// ListBlocksSince returns a slice of Block instances representing all blocks
	// that have been mined after the provided blockNumber.
	// The blockNumber parameter is the starting point (exclusive) for the block retrieval.
	// It returns a slice of Block and any error encountered.
	ListBlocksSince(ctx context.Context, blockNumber int64) ([]Block, error)
}

// Sync synchronizes the Monitor's internal state with the latest blockchain data.
// It performs the following steps:
//  1. Calls ListBlocksSince on the underlying Blockchain interface to retrieve all blocks
//     mined since the current block number stored in the Monitor.
//  2. If no new blocks are found, Sync returns nil without modifying the state.
//  3. Otherwise, it creates a new mapping for transactions for each address currently subscribed.
//  4. For each fetched block, it iterates through its transactions. If the transaction's
//     sender (tx.From) or receiver (tx.To) is in the subscribed addresses, the transaction is appended
//     to that address's list in the new mapping.
//  5. Updates the Monitor's current block number to the number of the last block fetched.
//  6. Replaces the internal mapping of addresses to transactions with the new mapping.
//  7. Returns nil upon successful synchronization, or an error if the blockchain call fails.
func (m *Monitor) Sync(ctx context.Context) error {
	blocks, err := m.blockchain.ListBlocksSince(ctx, m.currentBlockNumber)
	if err != nil {
		return err
	}

	if len(blocks) == 0 {
		return nil
	}

	currentTransactiosByAddress := make(map[string][]Transaction)
	for address := range m.transactiosByAddress {
		currentTransactiosByAddress[address] = make([]Transaction, 0)
	}

	for _, block := range blocks {
		for _, tx := range block.Transactions {
			if _, ok := currentTransactiosByAddress[tx.From]; ok {
				currentTransactiosByAddress[tx.From] = append(currentTransactiosByAddress[tx.From], tx)
			}

			if _, ok := currentTransactiosByAddress[tx.To]; ok {
				currentTransactiosByAddress[tx.To] = append(currentTransactiosByAddress[tx.To], tx)
			}
		}
	}

	m.currentBlockNumber = blocks[len(blocks)-1].Number
	m.transactiosByAddress = currentTransactiosByAddress
	return nil
}
