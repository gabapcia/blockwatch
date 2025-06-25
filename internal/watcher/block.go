package watcher

import "github.com/gabapcia/blockwatch/internal/pkg/types"

// Transaction represents a basic blockchain transaction,
// including its hash, sender address, and recipient address.
type Transaction struct {
	Hash string // Unique transaction hash identifier
	From string // Sender address
	To   string // Recipient address
}

// Block represents a blockchain block with its height, hash,
// and a list of transactions included in the block.
type Block struct {
	Height       types.Hex     // Block height represented as a hex string
	Hash         string        // Unique block hash
	Transactions []Transaction // List of transactions contained in the block
}
