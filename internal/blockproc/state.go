package blockproc

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// blockProcessingState encapsulates the lifecycle and metadata for processing
// a single blockchain block. It tracks when the block was received, how many
// times it has been processed, and any encountered errors.
//
// It is used internally to manage retries, ensure idempotency, and provide
// visibility into processing behavior (including timing and failure reasons).
type blockProcessingState struct {
	receivedAt       time.Time       // When the block was initially received for processing
	processingID     string          // Unique identifier for this processing attempt (SHA-256)
	block            Block           // The blockchain block associated with this processing state
	lastAttemptAt    *time.Time      // When the last processing attempt occurred (nil if never attempted)
	lastAttemptError error           // The most recent error encountered (if any)
	attempts         uint8           // Total number of processing attempts
	attemptErrorLog  map[int64]error // Map of Unix timestamps to the error from that attempt
	finalized        bool            // Indicates whether processing has been finalized (success or terminal failure)
	finalizedAt      *time.Time      // When the block processing was finalized (nil if not yet finalized)
}

// newBlockProcessingState creates a new blockProcessingState for the given
// block and network. It initializes timestamps, counters, and generates a
// deterministic processing ID for idempotency.
//
// The processing ID is derived by hashing the string "<network>:<blockHash>"
// using SHA-256. This ensures that the same block always results in the same
// ID, allowing the system to detect and prevent duplicate processing attempts.
func newBlockProcessingState(block Block) blockProcessingState {
	var (
		key          = fmt.Sprintf("%s:%s", block.Network, block.Hash)
		sum          = sha256.Sum256([]byte(key))
		processingID = hex.EncodeToString(sum[:])
	)

	return blockProcessingState{
		receivedAt:      time.Now().UTC(),
		processingID:    processingID,
		block:           block,
		attemptErrorLog: make(map[int64]error),
	}
}

// finalizeWithSuccess marks the processing as successfully completed.
// This method is a no-op if the state is already finalized.
func (s *blockProcessingState) finalizeWithSuccess() {
	if s.finalized {
		return
	}

	now := time.Now().UTC()

	s.finalized = true
	s.finalizedAt = &now
	s.lastAttemptError = nil
}

// finalizeWithFailure finalizes the processing as permanently failed, recording the current error.
// This method is a no-op if the state is already finalized.
func (s *blockProcessingState) finalizeWithFailure(err error) {
	if s.finalized {
		return
	}

	now := time.Now().UTC()

	s.finalized = true
	s.finalizedAt = &now
	s.lastAttemptError = err
	s.attemptErrorLog[now.Unix()] = err
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
	s.lastAttemptAt = &now
}

// recordAttemptFailure logs a new failure and stores the error under the current Unix timestamp.
// It also updates the lastAttemptError field.
// This method is a no-op if the state is already finalized.
func (s *blockProcessingState) recordAttemptFailure(err error) {
	if s.finalized {
		return
	}

	now := time.Now().UTC()

	s.lastAttemptError = err
	s.attemptErrorLog[now.Unix()] = err
}

// asSuccessResult converts the finalized state into a BlockProcessingSuccess value.
// It assumes the block was successfully processed with no errors.
// If the state is not finalized or ended in failure, a zero value is returned.
func (s blockProcessingState) asSuccessResult() BlockProcessingSuccess {
	if !s.finalized || s.lastAttemptError != nil {
		return BlockProcessingSuccess{}
	}

	return BlockProcessingSuccess{
		ProcessingID: s.processingID,
		Block:        s.block,
		ProcessedAt:  *s.finalizedAt,
	}
}

// asFailureResult converts the finalized state into a BlockProcessingFailure value.
// It assumes the block processing ended in a persistent failure.
// If the state is not finalized or has no error, a zero value is returned.
func (s blockProcessingState) asFailureResult() BlockProcessingFailure {
	if !s.finalized || s.lastAttemptError == nil {
		return BlockProcessingFailure{}
	}

	return BlockProcessingFailure{
		ProcessingID:  s.processingID,
		Block:         s.block,
		FailedAt:      *s.finalizedAt,
		Attempts:      s.attempts,
		LastError:     s.lastAttemptError,
		AttemptErrors: s.attemptErrorLog,
	}
}
