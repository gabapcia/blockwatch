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
	Network      string        // Blockchain network (e.g., "ethereum", "polygon")
	Height       types.Hex     // Block height represented as a hex string
	Hash         string        // Unique block hash
	Transactions []Transaction // List of transactions contained in the block
}

// BlockProcessingSuccess represents a successful processing result
// for a specific blockchain block on a given network.
//
// The ProcessingID is a deterministic SHA-256 hash of the string "<network>:<blockHash>",
// used to guarantee idempotency across retries or service restarts.
type BlockProcessingSuccess struct {
	ProcessingID string    // Deterministic ID for this processing cycle
	ProcessedAt  time.Time // Timestamp of successful completion
	Block        Block     // The block that was successfully processed
}

// BlockProcessingFailure represents a permanent failure during the processing
// of a blockchain block. It includes all failure metadata for diagnostics.
//
// The ProcessingID is a deterministic SHA-256 hash of the string "<network>:<blockHash>",
// used to guarantee idempotency across retries or service restarts.
type BlockProcessingFailure struct {
	ProcessingID  string          // Deterministic ID for this processing cycle
	FailedAt      time.Time       // Timestamp of when failure was finalized
	Attempts      uint8           // Number of attempts made
	LastError     error           // Last error encountered
	AttemptErrors map[int64]error // All attempt errors with their Unix timestamps
	Block         Block           // The block that failed processing
}
