package bootstrap

import (
	"context"
	"errors"

	"github.com/gabapcia/blockwatch/internal/blockproc"
	"github.com/gabapcia/blockwatch/internal/bootstrap/messaging"
	"github.com/gabapcia/blockwatch/internal/bootstrap/storage"
	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/infra/blockchain/ethereum"
	"github.com/gabapcia/blockwatch/internal/pkg/config"
	blockchainconfig "github.com/gabapcia/blockwatch/internal/pkg/config/blockchain"
	pkgconfig "github.com/gabapcia/blockwatch/internal/pkg/config/pkg"
	"github.com/gabapcia/blockwatch/internal/pkg/resilience/retry"
	"github.com/gabapcia/blockwatch/internal/pkg/transport/http"
	"github.com/gabapcia/blockwatch/internal/pkg/transport/jsonrpc"
	"github.com/gabapcia/blockwatch/internal/walletregistry"
	"github.com/gabapcia/blockwatch/internal/walletwatch"
)

// bootstrap aggregates all services that must be initialized at application startup.
//
// It provides a convenient container to access core modules such as walletwatch,
// walletregistry, chainstream, and block processing services.
type bootstrap struct {
	chainstream    chainstream.Service
	walletwatch    walletwatch.Service
	walletregistry walletregistry.Service
	blockproc      blockproc.Service
}

// Close shuts down the initialized services by closing storage and messaging resources.
//
// Returns:
//   - An aggregated error if any service fails to close properly.
func (b *bootstrap) Close() error {
	return errors.Join(
		storage.Close(),
		messaging.Close(),
	)
}

// New initializes the full application bootstrap with all configured services.
//
// It performs the following steps:
//  1. Initializes shared storage and messaging backends.
//  2. Sets up core services: ChainStream, WalletWatch, and WalletRegistry.
//  3. Constructs the block processor from ChainStream and WalletWatch.
//
// Parameters:
//   - ctx: request-scoped context for cancellation.
//   - config: root configuration struct loaded from environment or file.
//
// Returns:
//   - A fully initialized *bootstrap instance.
//   - An error if any initialization step fails.
func New(ctx context.Context, config config.Config) (*bootstrap, error) {
	if err := storage.Init(ctx, config.Engines.Storage); err != nil {
		return nil, err
	}
	defer storage.Close()

	if err := messaging.Init(ctx, config.Engines.Messaging); err != nil {
		return nil, err
	}
	defer messaging.Close()

	chainstream, err := setupChainStream(ctx, config.Chainstream)
	if err != nil {
		return nil, err
	}

	walletwatch, err := setupWalletWatch(ctx, config.Walletwatch)
	if err != nil {
		return nil, err
	}

	walletregistry, err := setupWalletRegistry(ctx, config.Walletregistry)
	if err != nil {
		return nil, err
	}

	return &bootstrap{
		chainstream:    chainstream,
		walletwatch:    walletwatch,
		walletregistry: walletregistry,
		blockproc:      blockproc.New(chainstream, walletwatch),
	}, nil
}

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

// buildJsonrpcClient creates a new JSON-RPC client with HTTP transport and retry configuration.
//
// It sets up the HTTP client using the specified timeout and retry parameters from the config.
//
// Parameters:
//   - cfg: configuration for the JSON-RPC client, including endpoint, timeout, and retry options.
//
// Returns:
//   - A configured jsonrpc.Client instance.
func buildJsonrpcClient(cfg pkgconfig.JsonRPC) jsonrpc.Client {
	httpClient := http.NewClient(
		http.WithTimeout(cfg.Timeout),
		http.WithRetryMax(cfg.RetryMax),
		http.WithRetryWaitMax(cfg.RetryWaitMax),
		http.WithRetryWaitMin(cfg.RetryWaitMin),
	)

	return jsonrpc.NewClient(httpClient.StandardClient(), cfg.ProviderEndpoint)
}

// setupChainStream initializes the ChainStream service using the provided configuration.
//
// It conditionally configures support for different blockchain networks (e.g., Ethereum),
// retry logic, checkpoint storage, and failure notifications.
//
// Parameters:
//   - ctx: request-scoped context for cancellation and timeouts.
//   - config: ChainStream-specific configuration including blockchain networks and optional backends.
//
// Returns:
//   - A chainstream.Service instance configured according to the provided settings.
//   - An error if any required dependency fails to resolve or initialize.
func setupChainStream(ctx context.Context, config config.ChainStream) (chainstream.Service, error) {
	networks := make(map[string]chainstream.Blockchain)

	if ethereumCfg := config.Networks.Ethereum; ethereumCfg != nil {
		jsonrpcClient := buildJsonrpcClient(*ethereumCfg)
		networks[blockchainconfig.ProviderEthereum] = ethereum.NewClient(jsonrpcClient)
	}

	opts := make([]chainstream.Option, 0)

	if config.Retry != nil {
		retrier := retry.New(
			retry.WithAttempts(config.Retry.Attempts),
			retry.WithDelay(config.Retry.Delay),
			retry.WithMaxDelay(config.Retry.MaxDelay),
		)

		opts = append(opts, chainstream.WithRetry(retrier))
	}

	if config.CheckpointStorage != nil {
		checkpointStorage, err := storage.Resolve[chainstream.CheckpointStorage](ctx, *config.CheckpointStorage)
		if err != nil {
			return nil, err
		}

		opts = append(opts, chainstream.WithCheckpointStorage(checkpointStorage))
	}

	if config.DispatchFailureNotifier != nil {
		dispatchFailureNotifier, err := messaging.Resolve[chainstream.DispatchFailureNotifier](ctx, *config.DispatchFailureNotifier)
		if err != nil {
			return nil, err
		}

		opts = append(opts, chainstream.WithDispatchFailureNotifier(dispatchFailureNotifier))
	}

	return chainstream.New(nil, opts...), nil
}
