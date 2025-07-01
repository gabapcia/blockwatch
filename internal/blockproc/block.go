package blockproc

import (
	"time"

	"github.com/gabapcia/blockwatch/internal/pkg/types"
)

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

// BlockProcessingSuccess represents a successful processing result
// for a specific blockchain block on a given network.
type BlockProcessingSuccess struct {
	ProcessingID string    // ID used to track this processing cycle
	Network      string    // Blockchain network (e.g., "ethereum", "polygon")
	Block        Block     // The block that was successfully processed
	ProcessedAt  time.Time // Timestamp of successful completion
}

// BlockProcessingFailure represents a permanent failure during the processing
// of a blockchain block. It includes all failure metadata for diagnostics.
type BlockProcessingFailure struct {
	ProcessingID  string          // ID used to track this processing cycle
	Network       string          // Blockchain network where the failure occurred
	Block         Block           // The block that failed processing
	FailedAt      time.Time       // Timestamp of when failure was finalized
	Attempts      uint8           // Number of attempts made
	LastError     error           // Last error encountered
	AttemptErrors map[int64]error // All attempt errors with their Unix timestamps
}
