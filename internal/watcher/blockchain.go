package watcher

import (
	"context"
	"errors"

	"github.com/gabapcia/blockwatch/internal/pkg/types"
)

// ErrNetworkNotRegistered is returned when attempting to operate on an unregistered network.
var ErrNetworkNotRegistered = errors.New("network not registered")

// BlockchainEvent represents an event emitted by the blockchain watcher.
// It always includes the block height that was processed, and may include
// either the full block data or an error if retrieval failed.
type BlockchainEvent struct {
	Height types.Hex // block height (always set)
	Block  Block     // block contents (zero value if Err is set)
	Err    error     // any error encountered (nil on success)
}

// Blockchain defines a source of blockchain data.
// It supports fetching individual blocks by height and streaming new blocks.
type Blockchain interface {
	// FetchBlockByHeight retrieves the block at the specified height.
	// It returns the full Block data or an error if fetching or decoding fails.
	FetchBlockByHeight(ctx context.Context, height types.Hex) (Block, error)

	// Subscribe begins streaming blocks from fromHeight (inclusive).
	// If fromHeight is the zero value, the implementation should fetch
	// the latest known block and begin streaming from there.
	//
	// It returns a receive-only channel of BlockchainEvent. The channel
	// is closed when ctx is canceled.
	Subscribe(ctx context.Context, fromHeight types.Hex) (<-chan BlockchainEvent, error)
}

// BlockDispatchFailure represents a failure that occurred when attempting to dispatch
// a block for processing. This typically happens when a BlockchainEvent contains an error.
//
// The error may be a single failure or a chain of multiple errors joined using errors.Join,
// especially if retries were attempted. Callers may use errors.Unwrap or errors.As to inspect
// the underlying errors.
type BlockDispatchFailure struct {
	Network string    // name of the blockchain network (e.g., "ethereum")
	Height  types.Hex // block height that failed to be dispatched
	Err     error     // one or more errors encountered; may be joined via errors.Join
}

// retryFailedBlockFetches contains the core retry logic for failed block fetches.
// It reads BlockDispatchFailure events from retryCh, attempts to re-fetch each block
// via s.retry.Execute, and:
//   - On success: sends the recovered blockProcessingState to recoveredCh (original error dropped).
//   - On persistent failure: merges the retry error with the original and forwards
//     the combined BlockDispatchFailure to finalErrorCh.
//
// retryCh, recoveredCh, and finalErrorCh are shared global channels; this function
// does not close any of them.
func (s *service) retryFailedBlockFetches(ctx context.Context, retryCh <-chan BlockDispatchFailure, recoveredCh chan<- blockProcessingState, finalErrorCh chan<- BlockDispatchFailure) {
	for netErr := range retryCh {
		retryErr := s.retry.Execute(ctx, func() error {
			client, ok := s.networks[netErr.Network]
			if !ok {
				return ErrNetworkNotRegistered
			}

			block, err := client.FetchBlockByHeight(ctx, netErr.Height)
			if err == nil {
				recoveredCh <- newBlockProcessingState(netErr.Network, block)
			}

			return err
		})
		if retryErr == nil {
			continue // success: drop the event
		}

		// persistent failure: attach retryErr and forward
		netErr.Err = errors.Join(netErr.Err, retryErr)

		select {
		case <-ctx.Done():
			return
		case finalErrorCh <- netErr:
		}
	}
}

// StartRetryFailedBlockFetches launches a background goroutine that invokes
// retryFailedBlockFetches. It returns immediately, leaving the retry loop
// running until retryCh is closed or ctx is canceled.
// retryCh, recoveredCh, and finalErrorCh must be closed by the caller.
func (s *service) startRetryFailedBlockFetches(ctx context.Context, retryCh <-chan BlockDispatchFailure, recoveredCh chan<- blockProcessingState, finalErrorCh chan<- BlockDispatchFailure) {
	go s.retryFailedBlockFetches(ctx, retryCh, recoveredCh, finalErrorCh)
}

// dispatchSubscriptionEvents reads BlockchainEvent values from eventsCh and routes them:
//   - On event.Err != nil, wraps the error and sends a BlockDispatchFailure to errorsCh.
//   - On success, sends a blockProcessingState to blocksCh.
//
// blocksCh and errorsCh are global shared channels and must be closed by the caller.
func (s *service) dispatchSubscriptionEvents(ctx context.Context, network string, eventsCh <-chan BlockchainEvent, blocksCh chan<- blockProcessingState, errorsCh chan<- BlockDispatchFailure) {
	for event := range eventsCh {
		if event.Err != nil {
			errorsCh <- BlockDispatchFailure{
				Network: network,
				Height:  event.Height,
				Err:     event.Err,
			}
			continue
		}

		select {
		case <-ctx.Done():
			return
		case blocksCh <- newBlockProcessingState(network, event.Block):
		}
	}
}

// launchAllNetworkSubscriptions initializes and starts a subscription for each registered network.
// For each network it:
//  1. Loads the last checkpointed block height (if any).
//  2. If a checkpoint exists, increments the start height by 1.
//  3. Calls Subscribe on the Blockchain client to obtain eventsCh.
//  4. Launches dispatchSubscriptionEvents in its own goroutine to forward blocks and errors.
//
// blocksCh and errorsCh are global shared channels and must be managed and closed by the caller.
// Returns an error if any initial subscription or checkpoint load (aside from no-checkpoint) fails.
func (s *service) launchAllNetworkSubscriptions(ctx context.Context, blocksCh chan<- blockProcessingState, errorsCh chan<- BlockDispatchFailure) error {
	for network, client := range s.networks {
		startHeight, err := s.checkpointStorage.LoadLatestCheckpoint(ctx, network)
		if err != nil && !errors.Is(err, ErrNoCheckpointFound) {
			return err
		}

		if !startHeight.IsEmpty() {
			startHeight = startHeight.Add(1)
		}

		eventsCh, err := client.Subscribe(ctx, startHeight)
		if err != nil {
			return err
		}

		go s.dispatchSubscriptionEvents(ctx, network, eventsCh, blocksCh, errorsCh)
	}

	return nil
}
