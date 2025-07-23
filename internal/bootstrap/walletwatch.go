package bootstrap

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/bootstrap/messaging"
	"github.com/gabapcia/blockwatch/internal/bootstrap/storage"
	"github.com/gabapcia/blockwatch/internal/pkg/config"
	"github.com/gabapcia/blockwatch/internal/walletwatch"
)

// setupWalletWatch initializes the WalletWatch service with all required dependencies.
//
// It resolves the storage and messaging components based on the provided configuration
// and applies any optional parameters such as max processing time and idempotency guard.
//
// Parameters:
//   - ctx: request-scoped context for cancellation and timeouts.
//   - config: WalletWatch-specific configuration containing picker settings and options.
//
// Returns:
//   - A walletwatch.Service instance ready to use.
//   - An error if any dependency fails to resolve or instantiate.
func setupWalletWatch(ctx context.Context, config config.WalletWatch) (walletwatch.Service, error) {
	walletStorage, err := storage.Resolve[walletwatch.WalletStorage](ctx, config.WalletStorage)
	if err != nil {
		return nil, err
	}

	transactionNotifier, err := messaging.Resolve[walletwatch.TransactionNotifier](ctx, config.TransactionNotifier)
	if err != nil {
		return nil, err
	}

	opts := make([]walletwatch.Option, 0)

	if config.MaxProcessingTime > 0 {
		opts = append(opts, walletwatch.WithMaxProcessingTime(config.MaxProcessingTime))
	}

	if config.IdempotencyGuard != nil {
		idempotencyGuard, err := storage.Resolve[walletwatch.IdempotencyGuard](ctx, *config.IdempotencyGuard)
		if err != nil {
			return nil, err
		}

		opts = append(opts, walletwatch.WithIdempotencyGuard(idempotencyGuard))
	}

	return walletwatch.New(walletStorage, transactionNotifier, opts...), nil
}
