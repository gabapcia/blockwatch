package bootstrap

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/bootstrap/storage"
	"github.com/gabapcia/blockwatch/internal/pkg/config"
	"github.com/gabapcia/blockwatch/internal/walletregistry"
)

// setupWalletRegistry initializes the WalletRegistry service using the provided configuration.
//
// It resolves the required WalletStorage backend and constructs the service instance.
//
// Parameters:
//   - ctx: request-scoped context for cancellation and timeouts.
//   - config: WalletRegistry-specific configuration containing the storage picker.
//
// Returns:
//   - A walletregistry.Service instance.
//   - An error if the storage resolution fails.
func setupWalletRegistry(ctx context.Context, config config.WalletRegistry) (walletregistry.Service, error) {
	walletStorage, err := storage.Resolve[walletregistry.WalletStorage](ctx, config.WalletStorage)
	if err != nil {
		return nil, err
	}

	return walletregistry.New(walletStorage), nil
}
