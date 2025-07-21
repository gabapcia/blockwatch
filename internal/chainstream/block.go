package chainstream

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/pkg/flow/chflow"
	"github.com/gabapcia/blockwatch/internal/pkg/logger"
	"github.com/gabapcia/blockwatch/internal/pkg/types"
)

// Transaction represents a basic blockchain transaction,
// including its hash, sender address, and recipient address.
type Transaction struct {
	Hash string // Unique transaction hash identifier.
	From string // Sender address.
	To   string // Recipient address.
}

// Block represents a blockchain block with its height, hash,
// and a list of transactions included in the block.
type Block struct {
	Height       types.Hex     // Block height represented as a hex string.
	Hash         string        // Unique block hash.
	Transactions []Transaction // List of transactions contained in the block.
}

// ObservedBlock represents a blockchain block that has been detected by the chainstream system,
// annotated with the network from which it originated. It includes the full block data and
// the name of the blockchain network (e.g., "ethereum", "polygon").
//
// This struct is typically used as the primary output of the chainstream package, enabling
// consumers to process new blocks along with their network context.
type ObservedBlock struct {
	Network string // Name of the blockchain network (e.g., "ethereum", "polygon").
	Block          // Embedded Block struct containing block height, hash, and transactions.
}

// DispatchFailureNotifier defines the interface used to handle unrecoverable dispatch errors.
//
// Implementations of this interface can perform actions such as logging, alerting, or
// persisting the failure for later analysis. It is invoked when a block dispatch permanently fails.
type DispatchFailureNotifier interface {
	// NotifyDispatchFailure is called when a dispatch failure occurs that cannot be recovered.
	// It receives contextual information and the failure details.
	//
	// If an error is returned, it will be logged by the service.
	NotifyDispatchFailure(ctx context.Context, failure BlockDispatchFailure) error
}

// handleDispatchFailures consumes unrecoverable block dispatch errors from dispatchErrCh
// and sends them to the configured DispatchFailureNotifier for handling.
//
// This method blocks until dispatchErrCh is closed or ctx is canceled.
// If the notifier returns an error, it is logged using the default logger.
func (s *service) handleDispatchFailures(ctx context.Context, dispatchErrCh <-chan BlockDispatchFailure) {
	for {
		dispatchFailure, ok := chflow.Receive(ctx, dispatchErrCh)
		if !ok {
			return
		}

		if err := s.dispatchFailureNotifier.NotifyDispatchFailure(ctx, dispatchFailure); err != nil {
			logger.Error(ctx, "NotifyDispatchFailure error",
				"block.network", dispatchFailure.Network,
				"block.height", dispatchFailure.Height,
				"block.errors", dispatchFailure.Errors,
				"error", err,
			)
		}
	}
}

// startHandleDispatchFailures launches handleDispatchFailures in a background goroutine.
//
// It starts the error-handling loop that listens for dispatch failures until
// dispatchErrCh is closed or ctx is canceled. This function is invoked during service startup.
func (s *service) startHandleDispatchFailures(ctx context.Context, dispatchErrCh <-chan BlockDispatchFailure) {
	go s.handleDispatchFailures(ctx, dispatchErrCh)
}

// logDispatchFailureNotifier is the default implementation of DispatchFailureNotifier.
// It logs unrecoverable dispatch failures using the application's logger.
type logDispatchFailureNotifier struct{}

// NotifyDispatchFailure logs the details of the unrecoverable dispatch failure.
// It always returns nil, ensuring that no additional error handling is triggered.
func (logDispatchFailureNotifier) NotifyDispatchFailure(ctx context.Context, failure BlockDispatchFailure) error {
	logger.Error(ctx, "block dispatch failure",
		"block.network", failure.Network,
		"block.height", failure.Height,
		"block.errors", failure.Errors,
	)

	return nil
}

// Compile-time assertion to ensure logDispatchFailureNotifier implements DispatchFailureNotifier.
var _ DispatchFailureNotifier = new(logDispatchFailureNotifier)
