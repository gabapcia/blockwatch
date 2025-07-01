package blockproc

import (
	"time"

	"github.com/google/uuid"
)

// blockProcessingState encapsulates the lifecycle and metadata for processing
// a single blockchain block. It tracks when the block was received, how many
// times it has been processed, and any encountered errors.
//
// It is used internally to manage retries, ensure idempotency, and provide
// visibility into processing behavior (including timing and failure reasons).
type blockProcessingState struct {
	receivedAt       time.Time       // When the block was initially received for processing
	processingID     string          // Unique identifier for this processing attempt (UUIDv7)
	network          string          // Blockchain network name (e.g., "ethereum", "polygon")
	block            Block           // The blockchain block associated with this processing state
	lastAttemptAt    *time.Time      // When the last processing attempt occurred (nil if never attempted)
	lastAttemptError error           // The most recent error encountered (if any)
	attempts         uint8           // Total number of processing attempts
	attemptErrorLog  map[int64]error // Map of Unix timestamps to the error from that attempt
	finalized        bool            // Indicates whether processing has been finalized (success or terminal failure)
	finalizedAt      *time.Time      // When the block processing was finalized (nil if not yet finalized)
}

// newBlockProcessingState creates a new blockProcessingState for the given
// block and network, initializing timestamps, counters, and generating a
// unique processing ID.
func newBlockProcessingState(network string, block Block) blockProcessingState {
	return blockProcessingState{
		processingID:    uuid.Must(uuid.NewV7()).String(),
		receivedAt:      time.Now().UTC(),
		attempts:        0,
		attemptErrorLog: make(map[int64]error),
		network:         network,
		block:           block,
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
		Network:      s.network,
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
		Network:       s.network,
		Block:         s.block,
		FailedAt:      *s.finalizedAt,
		Attempts:      s.attempts,
		LastError:     s.lastAttemptError,
		AttemptErrors: s.attemptErrorLog,
	}
}
