package chainstream

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

// ObservedBlock represents a blockchain block that has been detected by the chainstream system,
// annotated with the network from which it originated. It includes the full block data and
// the name of the blockchain network (e.g., "ethereum", "polygon").
//
// This struct is typically used as the primary output of the chainstream package, enabling
// consumers to process new blocks along with their network context.
type ObservedBlock struct {
	Network string // Name of the blockchain network (e.g., "ethereum", "polygon")
	Block          // Embedded Block struct containing block height, hash, and transactions
}
