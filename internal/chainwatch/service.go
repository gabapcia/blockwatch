package chainwatch

import (
	"context"
	"errors"
	"sync"

	"github.com/gabapcia/blockwatch/internal/pkg/logger"
	"github.com/gabapcia/blockwatch/internal/pkg/resilience/retry"
	"github.com/gabapcia/blockwatch/internal/pkg/x/chflow"
)

// ErrServiceAlreadyStarted is returned when Start is called on a Service
// that has already been started. A Service instance must not be started more than once.
var ErrServiceAlreadyStarted = errors.New("service already started")

// Constants defining the buffer size for internal channels used by the service.
const (
	dispatchFailureChannelBufferSize = 5  // Buffer size for dispatch failure events
	retryFailureChannelBufferSize    = 5  // Buffer size for failures retried by retry logic
	observedBlockChannelBufferSize   = 10 // Buffer size for final successfully observed blocks
)

// Service represents a chainwatch streaming component responsible for subscribing
// to one or more blockchain networks, handling block retrieval, retry logic,
// and emitting observed blocks or transformed data for downstream consumers.
type Service[T any] interface {
	// Start begins the block observation process and returns a channel of observed blocks or transformed data.
	//
	// It must be called only once; calling Start again returns ErrServiceAlreadyStarted.
	// The returned channel is closed only when Close is called or the context is canceled.
	Start(ctx context.Context) (<-chan T, error)

	// Close terminates all background processes, closes internal channels,
	// and makes the Service eligible for reinitialization if desired.
	Close()
}

// closeFunc defines a cleanup routine executed when the service is closed.
// It is responsible for canceling internal contexts and closing channels
// to gracefully shut down all background operations.
type closeFunc func()

// dispatchFailureHandler is a user-provided function called whenever a block dispatch
// (i.e., sending a fetched block for processing) fails and cannot be recovered.
type dispatchFailureHandler func(ctx context.Context, dispatchFailure BlockDispatchFailure)

// transformFunc is an optional user-defined function that transforms an ObservedBlock into type T.
// If nil, the ObservedBlock is sent as-is (T must be ObservedBlock in this case).
type transformFunc[T any] func(ObservedBlock) T

// service is the internal implementation of the Service interface.
// It orchestrates subscriptions, retries, and block delivery for multiple blockchain networks.
type service[T any] struct {
	mu        sync.Mutex // protects lifecycle state
	isStarted bool       // indicates whether Start was called
	closeFunc closeFunc  // cancels background routines and cleans up channels

	networks          map[string]Blockchain // registered blockchain clients by network name
	checkpointStorage CheckpointStorage     // mechanism for saving/restoring last processed height

	retry                  retry.Retry            // optional retry logic for failed block fetches
	dispatchFailureHandler dispatchFailureHandler // user-defined callback for unrecoverable dispatch errors
	transformFunc          transformFunc[T]       // optional transformation function from ObservedBlock to T
}

// Compile-time check to ensure *service implements the Service interface.
var _ Service[ObservedBlock] = (*service[ObservedBlock])(nil)

// checkpointAndForward reads ObservedBlock values from blockIn,
// saves a checkpoint for each block, applies transformation, and forwards them to transformedOut.
//
// This function ensures that every successfully observed block has its height
// saved as a checkpoint before being transformed and sent to downstream consumers.
//
// blockIn and transformedOut are managed by the caller and must be closed appropriately.
// Panics if transformFunc is nil (indicates an internal package bug).
func (s *service[T]) checkpointAndForward(ctx context.Context, blockIn <-chan ObservedBlock, transformedOut chan<- T) {
	for {
		observedBlock, ok := chflow.Receive(ctx, blockIn)
		if !ok {
			return
		}

		// Save checkpoint before forwarding the block
		if err := s.checkpointStorage.SaveCheckpoint(ctx, observedBlock.Network, observedBlock.Height); err != nil {
			// Log the error but continue processing - checkpoint failure shouldn't stop block processing
			logger.Error(ctx, "failed to save checkpoint",
				"block.network", observedBlock.Network,
				"block.height", observedBlock.Height,
				"error", err,
			)
			continue
		}

		// Transform the block using the provided transform function
		output := s.transformFunc(observedBlock)

		// Forward the transformed output to the final output channel
		if ok := chflow.Send(ctx, transformedOut, output); !ok {
			return
		}
	}
}

// startCheckpointAndForward launches checkpointAndForward
// in a background goroutine. It returns immediately, leaving the processor running
// until blockIn is closed or ctx is canceled.
func (s *service[T]) startCheckpointAndForward(ctx context.Context, blockIn <-chan ObservedBlock, transformedOut chan<- T) {
	go s.checkpointAndForward(ctx, blockIn, transformedOut)
}

// Start initializes all subscriptions for registered networks,
// starts retry and dispatch failure handlers, and returns a channel of transformed data.
//
// The returned channel emits transformed values as new blocks are fetched and verified.
// If the service was already started, Start returns ErrServiceAlreadyStarted.
//
// The caller is responsible for eventually calling Close to clean up resources.
func (s *service[T]) Start(ctx context.Context) (<-chan T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isStarted {
		return nil, ErrServiceAlreadyStarted
	}

	ctx, cancel := context.WithCancel(ctx)

	var (
		retryFailureCh    chan BlockDispatchFailure
		dispatchFailureCh = make(chan BlockDispatchFailure, dispatchFailureChannelBufferSize)
		preCheckpointCh   = make(chan ObservedBlock, observedBlockChannelBufferSize)
		finalOut          = make(chan T, observedBlockChannelBufferSize)
	)

	s.closeFunc = func() {
		cancel()
		close(preCheckpointCh)
		close(finalOut)
		if retryFailureCh != nil {
			close(retryFailureCh)
		}
		close(dispatchFailureCh)
	}

	s.startHandleDispatchFailures(ctx, dispatchFailureCh)

	if s.retry != nil {
		retryFailureCh = make(chan BlockDispatchFailure, retryFailureChannelBufferSize)
		s.startRetryFailedBlockFetches(ctx, retryFailureCh, preCheckpointCh, dispatchFailureCh)
	}

	// Start the checkpoint processor that sits between internal processing and final output
	s.startCheckpointAndForward(ctx, preCheckpointCh, finalOut)

	errorSubmissionCh := chflow.FirstNonNil(retryFailureCh, dispatchFailureCh)
	if err := s.launchAllNetworkSubscriptions(ctx, preCheckpointCh, errorSubmissionCh); err != nil {
		s.closeFunc()
		return nil, err
	}

	s.isStarted = true
	return finalOut, nil
}

// Close shuts down the service, cancels all active routines, and closes internal channels.
//
// It is safe to call Close even if the service was never started.
// After calling Close, the Service can be safely discarded.
func (s *service[T]) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closeFunc != nil {
		s.closeFunc()
	}
	s.isStarted = false
	s.closeFunc = nil
}

// config holds the configuration parameters used to initialize a service instance.
// These are populated using functional options passed to New.
type config struct {
	retry                  retry.Retry            // optional retry mechanism for transient fetch failures
	checkpointStorage      CheckpointStorage      // storage backend for tracking the last processed block
	dispatchFailureHandler dispatchFailureHandler // user-defined handler for unrecoverable dispatch errors
}

// defaultTransformFunc is the default transformation function that returns the ObservedBlock as-is.
func defaultTransformFunc(ob ObservedBlock) ObservedBlock {
	return ob
}

// Option defines a functional option for configuring a Service instance.
// It is applied inside the New constructor.
type Option func(*config)

// New creates a new instance of the chainwatch service that returns ObservedBlock values.
//
// It requires a map of network identifiers to Blockchain clients.
// Optional behavior like retry logic, checkpoint persistence, and error handling
// can be customized via the provided Option functions.
//
// Defaults:
//   - No retry logic (retry = nil)
//   - No persistent checkpointing (uses a no-op CheckpointStorage)
//   - Dispatch failures are logged using the default logger
//   - Transform function returns ObservedBlock as-is
func New(networks map[string]Blockchain, opts ...Option) *service[ObservedBlock] {
	cfg := config{
		retry:                  nil,
		checkpointStorage:      nopCheckpoint{},
		dispatchFailureHandler: defaultOnDispatchFailure,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &service[ObservedBlock]{
		networks:               networks,
		checkpointStorage:      cfg.checkpointStorage,
		retry:                  cfg.retry,
		dispatchFailureHandler: cfg.dispatchFailureHandler,
		transformFunc:          defaultTransformFunc,
	}
}

// NewWithTransform creates a new instance of the chainwatch service with a custom transform function.
//
// It requires a map of network identifiers to Blockchain clients and a transform function.
// Optional behavior like retry logic, checkpoint persistence, and error handling
// can be customized via the provided Option functions.
//
// Defaults:
//   - No retry logic (retry = nil)
//   - No persistent checkpointing (uses a no-op CheckpointStorage)
//   - Dispatch failures are logged using the default logger
func NewWithTransform[T any](networks map[string]Blockchain, transformFunc transformFunc[T], opts ...Option) *service[T] {
	cfg := config{
		retry:                  nil,
		checkpointStorage:      nopCheckpoint{},
		dispatchFailureHandler: defaultOnDispatchFailure,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &service[T]{
		networks:               networks,
		checkpointStorage:      cfg.checkpointStorage,
		retry:                  cfg.retry,
		dispatchFailureHandler: cfg.dispatchFailureHandler,
		transformFunc:          transformFunc,
	}
}

// defaultOnDispatchFailure is the default handler used when the user does not provide one.
// It logs the failure using the application's logger with context and error details.
func defaultOnDispatchFailure(ctx context.Context, dispatchFailure BlockDispatchFailure) {
	logger.Error(ctx, "block dispatch failure",
		"block.network", dispatchFailure.Network,
		"block.height", dispatchFailure.Height,
		"block.errors", dispatchFailure.Errors,
	)
}

// WithDispatchFailureHandler sets a custom function to handle unrecoverable
// block dispatch failures (e.g., due to permanent fetch errors).
//
// By default, failures are logged using the standard logger.
func WithDispatchFailureHandler(f dispatchFailureHandler) Option {
	return func(c *config) {
		c.dispatchFailureHandler = f
	}
}

// WithRetry configures the service with a retry strategy for transient block fetch failures.
//
// If not set, no retries will be attempted by default (retry = nil).
func WithRetry(r retry.Retry) Option {
	return func(c *config) {
		c.retry = r
	}
}

// WithCheckpointStorage sets the component responsible for persisting
// the latest successfully processed block height per network.
//
// By default, a no-op implementation is used (nopCheckpoint), which
// disables checkpointing and always starts from scratch.
func WithCheckpointStorage(cs CheckpointStorage) Option {
	return func(c *config) {
		c.checkpointStorage = cs
	}
}
