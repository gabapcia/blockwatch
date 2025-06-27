package watcher

import (
	"context"
	"time"

	"github.com/gabapcia/blockwatch/internal/pkg/types"
	"github.com/google/uuid"
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

// blockProcessingState encapsulates the lifecycle and metadata for processing
// a single blockchain block. It tracks when the block was received, how many
// times it has been processed, and any encountered errors.
//
// It is used internally to manage retries, ensure idempotency, and provide
// visibility into processing behavior (including timing and failure reasons).
type blockProcessingState struct {
	processingID  string          // Unique identifier for this processing attempt (UUIDv7)
	receivedAt    time.Time       // When the block was initially received for processing
	lastTriedAt   *time.Time      // When the last processing attempt occurred (nil if never attempted)
	network       string          // Blockchain network name (e.g., "ethereum", "polygon")
	attempts      uint8           // Total number of processing attempts
	attemptErrors map[int64]error // Map of Unix timestamps to the error from that attempt
	lastError     error           // The most recent error encountered (if any)
	finalized     bool            // Indicates whether processing has been finalized (success or terminal failure)
	finalizedAt   *time.Time      // When the block processing was finalized (nil if not yet finalized)
	block         Block           // The blockchain block associated with this processing state
}

// newBlockProcessingState creates a new blockProcessingState for the given
// block and network, initializing timestamps, counters, and generating a
// unique processing ID.
func newBlockProcessingState(network string, block Block) blockProcessingState {
	return blockProcessingState{
		processingID:  uuid.Must(uuid.NewV7()).String(),
		receivedAt:    time.Now().UTC(),
		attempts:      0,
		attemptErrors: make(map[int64]error),
		network:       network,
		block:         block,
	}
}

// setSuccess marks the processing as successfully completed.
// This method is a no-op if the state is already finalized.
func (s *blockProcessingState) setSuccess() {
	if s.finalized {
		return
	}

	now := time.Now().UTC()

	s.finalized = true
	s.finalizedAt = &now
	s.lastError = nil
}

// recordAttempt updates the state to reflect a new processing attempt.
// It increments the attempt counter and records the timestamp.
// This method is a no-op if the state is already finalized.
func (s *blockProcessingState) recordAttempt() {
	if s.finalized {
		return
	}

	now := time.Now().UTC()

	s.attempts++
	s.lastTriedAt = &now
}

// recordFailure logs a new failure and stores the error under the current Unix timestamp.
// It also updates the lastError field.
// This method is a no-op if the state is already finalized.
func (s *blockProcessingState) recordFailure(err error) {
	if s.finalized {
		return
	}

	now := time.Now().UTC()

	s.lastError = err
	s.attemptErrors[now.Unix()] = err
}

// setFailure finalizes the processing as permanently failed, recording the current error.
// This method is a no-op if the state is already finalized.
func (s *blockProcessingState) setFailure(err error) {
	if s.finalized {
		return
	}

	now := time.Now().UTC()

	s.finalized = true
	s.finalizedAt = &now
	s.lastError = err
	s.attemptErrors[now.Unix()] = err
}

// toSuccessResult converts the finalized state into a BlockProcessingSuccess value.
// It assumes the block was successfully processed with no errors.
// If the state is not finalized or ended in failure, a zero value is returned.
func (s blockProcessingState) toSuccessResult() BlockProcessingSuccess {
	if !s.finalized || s.lastError != nil {
		return BlockProcessingSuccess{}
	}

	return BlockProcessingSuccess{
		ProcessingID: s.processingID,
		Network:      s.network,
		Block:        s.block,
		ProcessedAt:  *s.finalizedAt,
	}
}

// toFailureResult converts the finalized state into a BlockProcessingFailure value.
// It assumes the block processing ended in a persistent failure.
// If the state is not finalized or has no error, a zero value is returned.
func (s blockProcessingState) toFailureResult() BlockProcessingFailure {
	if !s.finalized || s.lastError == nil {
		return BlockProcessingFailure{}
	}

	return BlockProcessingFailure{
		ProcessingID:  s.processingID,
		Network:       s.network,
		Block:         s.block,
		FailedAt:      *s.finalizedAt,
		Attempts:      s.attempts,
		LastError:     s.lastError,
		AttemptErrors: s.attemptErrors,
	}
}

func (s *service) process(ctx context.Context, events <-chan blockProcessingState) error {
	return nil
}
