package blockwatch

import "context"

type Parser interface {
	// GetCurrentBlock last parsed block
	GetCurrentBlock() int64

	// Subscribe add address to observer
	Subscribe(address string) bool

	// GetTransactions list of inbound or outbound transactions for an address
	GetTransactions(address string) []Transaction
}

// Monitor implements the Parser interface and is responsible for tracking blockchain transactions.
// It interacts with a Blockchain instance to fetch block data and maintains an internal record
// of the current block number as well as a mapping of addresses to their associated transactions.
type Monitor struct {
	// blockchain is the underlying Blockchain interface used to fetch blockchain data.
	blockchain Blockchain

	// currentBlockNumber holds the last parsed block number.
	currentBlockNumber int64

	// transactiosByAddress maps a blockchain address to its corresponding latest transactions.
	transactiosByAddress map[string][]Transaction
}

// New creates and initializes a new Monitor instance using the provided Blockchain instance.
// It retrieves the latest block number from the blockchain and initializes the transaction map.
// If fetching the latest block number fails, it returns an error.
func New(ctx context.Context, blockchain Blockchain) (*Monitor, error) {
	currentBlockNumber, err := blockchain.GetLatestBlockNumber(ctx)
	if err != nil {
		return nil, err
	}

	return &Monitor{
		blockchain:           blockchain,
		currentBlockNumber:   currentBlockNumber,
		transactiosByAddress: make(map[string][]Transaction),
	}, nil
}

// Ensure that Monitor implements the Parser interface.
var _ Parser = (*Monitor)(nil)
