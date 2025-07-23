package bootstrap

import (
	"context"
	"testing"
	"time"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/pkg/config"
	blockchainconfig "github.com/gabapcia/blockwatch/internal/pkg/config/blockchain"
	"github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	pkgconfig "github.com/gabapcia/blockwatch/internal/pkg/config/pkg"
	"github.com/gabapcia/blockwatch/internal/pkg/config/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("returns error when storage initialization fails", func(t *testing.T) {
		// Arrange
		cfg := config.Config{
			ServiceName: "test-blockwatch",
			Log: pkgconfig.Logger{
				Level: "info",
			},
			Engines: config.Engines{
				Storage: storage.Engines{
					// No storage engines configured - will cause storage.Init to fail
				},
				Messaging: messaging.Engines{
					Redis: &messaging.RedisConnection{
						Address:  "localhost:6379",
						Username: "",
						Password: "",
						DB:       0,
					},
				},
			},
			Walletregistry: config.WalletRegistry{
				WalletStorage: storage.Picker{
					Engine: storage.EngineRedis,
				},
			},
			Walletwatch: config.WalletWatch{
				MaxProcessingTime: 5 * time.Minute,
				WalletStorage: storage.Picker{
					Engine: storage.EngineRedis,
				},
				TransactionNotifier: messaging.Picker{
					Engine: messaging.EngineRedis,
					MessagePublisher: messaging.MessagePublisher{
						Redis: &messaging.RedisPublisher{
							Stream: "wallet-transactions",
						},
					},
				},
			},
			Chainstream: config.ChainStream{
				Networks: blockchainconfig.Networks{},
			},
		}

		// Act
		bootstrap, err := New(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, bootstrap)
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("returns error when messaging initialization fails", func(t *testing.T) {
		// Arrange
		cfg := config.Config{
			ServiceName: "test-blockwatch",
			Log: pkgconfig.Logger{
				Level: "info",
			},
			Engines: config.Engines{
				Storage: storage.Engines{
					Redis: &storage.Redis{
						Address:  "localhost:6379",
						Username: "",
						Password: "",
						DB:       0,
					},
				},
				Messaging: messaging.Engines{
					// No messaging engines configured - will cause messaging.Init to fail
				},
			},
			Walletregistry: config.WalletRegistry{
				WalletStorage: storage.Picker{
					Engine: storage.EngineRedis,
				},
			},
			Walletwatch: config.WalletWatch{
				MaxProcessingTime: 5 * time.Minute,
				WalletStorage: storage.Picker{
					Engine: storage.EngineRedis,
				},
				TransactionNotifier: messaging.Picker{
					Engine: messaging.EngineRedis,
					MessagePublisher: messaging.MessagePublisher{
						Redis: &messaging.RedisPublisher{
							Stream: "wallet-transactions",
						},
					},
				},
			},
			Chainstream: config.ChainStream{
				Networks: blockchainconfig.Networks{},
			},
		}

		// Act
		bootstrap, err := New(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, bootstrap)
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("returns error when chainstream setup fails", func(t *testing.T) {
		// Arrange
		cfg := config.Config{
			ServiceName: "test-blockwatch",
			Log: pkgconfig.Logger{
				Level: "info",
			},
			Engines: config.Engines{
				Storage: storage.Engines{
					Redis: &storage.Redis{
						Address:  "localhost:6379",
						Username: "",
						Password: "",
						DB:       0,
					},
				},
				Messaging: messaging.Engines{
					Redis: &messaging.RedisConnection{
						Address:  "localhost:6379",
						Username: "",
						Password: "",
						DB:       0,
					},
				},
			},
			Walletregistry: config.WalletRegistry{
				WalletStorage: storage.Picker{
					Engine: storage.EngineRedis,
				},
			},
			Walletwatch: config.WalletWatch{
				MaxProcessingTime: 5 * time.Minute,
				WalletStorage: storage.Picker{
					Engine: storage.EngineRedis,
				},
				TransactionNotifier: messaging.Picker{
					Engine: messaging.EngineRedis,
					MessagePublisher: messaging.MessagePublisher{
						Redis: &messaging.RedisPublisher{
							Stream: "wallet-transactions",
						},
					},
				},
			},
			Chainstream: config.ChainStream{
				Networks: blockchainconfig.Networks{},
				CheckpointStorage: &storage.Picker{
					Engine: storage.EnginePostgreSQL, // PostgreSQL not configured
				},
			},
		}

		// Act
		bootstrap, err := New(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, bootstrap)
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("returns error when walletwatch setup fails", func(t *testing.T) {
		// Arrange
		cfg := config.Config{
			ServiceName: "test-blockwatch",
			Log: pkgconfig.Logger{
				Level: "info",
			},
			Engines: config.Engines{
				Storage: storage.Engines{
					Redis: &storage.Redis{
						Address:  "localhost:6379",
						Username: "",
						Password: "",
						DB:       0,
					},
				},
				Messaging: messaging.Engines{
					Redis: &messaging.RedisConnection{
						Address:  "localhost:6379",
						Username: "",
						Password: "",
						DB:       0,
					},
				},
			},
			Walletregistry: config.WalletRegistry{
				WalletStorage: storage.Picker{
					Engine: storage.EngineRedis,
				},
			},
			Walletwatch: config.WalletWatch{
				MaxProcessingTime: 5 * time.Minute,
				WalletStorage: storage.Picker{
					Engine: "INVALID_ENGINE", // Invalid engine
				},
				TransactionNotifier: messaging.Picker{
					Engine: messaging.EngineRedis,
					MessagePublisher: messaging.MessagePublisher{
						Redis: &messaging.RedisPublisher{
							Stream: "wallet-transactions",
						},
					},
				},
			},
			Chainstream: config.ChainStream{
				Networks: blockchainconfig.Networks{},
			},
		}

		// Act
		bootstrap, err := New(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, bootstrap)
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("returns error when walletregistry setup fails", func(t *testing.T) {
		// Arrange
		cfg := config.Config{
			ServiceName: "test-blockwatch",
			Log: pkgconfig.Logger{
				Level: "info",
			},
			Engines: config.Engines{
				Storage: storage.Engines{
					Redis: &storage.Redis{
						Address:  "localhost:6379",
						Username: "",
						Password: "",
						DB:       0,
					},
				},
				Messaging: messaging.Engines{
					Redis: &messaging.RedisConnection{
						Address:  "localhost:6379",
						Username: "",
						Password: "",
						DB:       0,
					},
				},
			},
			Walletregistry: config.WalletRegistry{
				WalletStorage: storage.Picker{
					Engine: "INVALID_ENGINE", // Invalid engine
				},
			},
			Walletwatch: config.WalletWatch{
				MaxProcessingTime: 5 * time.Minute,
				WalletStorage: storage.Picker{
					Engine: storage.EngineRedis,
				},
				TransactionNotifier: messaging.Picker{
					Engine: messaging.EngineRedis,
					MessagePublisher: messaging.MessagePublisher{
						Redis: &messaging.RedisPublisher{
							Stream: "wallet-transactions",
						},
					},
				},
			},
			Chainstream: config.ChainStream{
				Networks: blockchainconfig.Networks{},
			},
		}

		// Act
		bootstrap, err := New(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, bootstrap)
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("returns error when context is cancelled", func(t *testing.T) {
		// Arrange
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		cfg := config.Config{
			ServiceName: "test-blockwatch",
			Log: pkgconfig.Logger{
				Level: "info",
			},
			Engines: config.Engines{
				Storage: storage.Engines{
					Redis: &storage.Redis{
						Address:  "localhost:6379",
						Username: "",
						Password: "",
						DB:       0,
					},
				},
				Messaging: messaging.Engines{
					Redis: &messaging.RedisConnection{
						Address:  "localhost:6379",
						Username: "",
						Password: "",
						DB:       0,
					},
				},
			},
			Walletregistry: config.WalletRegistry{
				WalletStorage: storage.Picker{
					Engine: storage.EngineRedis,
				},
			},
			Walletwatch: config.WalletWatch{
				MaxProcessingTime: 5 * time.Minute,
				WalletStorage: storage.Picker{
					Engine: storage.EngineRedis,
				},
				TransactionNotifier: messaging.Picker{
					Engine: messaging.EngineRedis,
					MessagePublisher: messaging.MessagePublisher{
						Redis: &messaging.RedisPublisher{
							Stream: "wallet-transactions",
						},
					},
				},
			},
			Chainstream: config.ChainStream{
				Networks: blockchainconfig.Networks{},
			},
		}

		// Act
		bootstrap, err := New(ctx, cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, bootstrap)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

func TestBootstrap_Close(t *testing.T) {
	t.Run("close calls storage and messaging close functions", func(t *testing.T) {
		b := &bootstrap{}

		// Execute
		err := b.Close()
		assert.NoError(t, err)
	})
}

func TestSetupWalletWatch(t *testing.T) {
	t.Run("returns error when redis storage engine is not configured", func(t *testing.T) {
		// Arrange
		cfg := config.WalletWatch{
			WalletStorage: storage.Picker{
				Engine: storage.EngineRedis,
			},
			TransactionNotifier: messaging.Picker{
				Engine: messaging.EngineRedis,
				MessagePublisher: messaging.MessagePublisher{
					Redis: &messaging.RedisPublisher{
						Stream: "wallet-transactions",
					},
				},
			},
		}

		// Act
		service, err := setupWalletWatch(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "no default instance found")
	})

	t.Run("returns error when redis storage engine is not configured with max processing time", func(t *testing.T) {
		// Arrange
		cfg := config.WalletWatch{
			MaxProcessingTime: 10 * time.Minute,
			WalletStorage: storage.Picker{
				Engine: storage.EngineRedis,
			},
			TransactionNotifier: messaging.Picker{
				Engine: messaging.EngineRedis,
				MessagePublisher: messaging.MessagePublisher{
					Redis: &messaging.RedisPublisher{
						Stream: "wallet-transactions",
					},
				},
			},
		}

		// Act
		service, err := setupWalletWatch(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "no default instance found")
	})

	t.Run("returns error when postgresql idempotency guard is not configured", func(t *testing.T) {
		// Arrange
		idempotencyGuardPicker := &storage.Picker{
			Engine: storage.EnginePostgreSQL,
		}
		cfg := config.WalletWatch{
			WalletStorage: storage.Picker{
				Engine: storage.EngineRedis,
			},
			TransactionNotifier: messaging.Picker{
				Engine: messaging.EngineRedis,
				MessagePublisher: messaging.MessagePublisher{
					Redis: &messaging.RedisPublisher{
						Stream: "wallet-transactions",
					},
				},
			},
			IdempotencyGuard: idempotencyGuardPicker,
		}

		// Act
		service, err := setupWalletWatch(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "no default instance found")
	})

	t.Run("returns error when rabbitmq messaging engine is not configured", func(t *testing.T) {
		// Arrange
		idempotencyGuardPicker := &storage.Picker{
			Engine: storage.EnginePostgreSQL,
		}
		cfg := config.WalletWatch{
			MaxProcessingTime: 15 * time.Minute,
			WalletStorage: storage.Picker{
				Engine: storage.EngineRedis,
			},
			TransactionNotifier: messaging.Picker{
				Engine: messaging.EngineRabbitMQ,
				MessagePublisher: messaging.MessagePublisher{
					RabbitMQ: &messaging.RabbitMQPublisher{
						Exchange:   "wallet-exchange",
						RoutingKey: "wallet.transaction",
					},
				},
			},
			IdempotencyGuard: idempotencyGuardPicker,
		}

		// Act
		service, err := setupWalletWatch(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "no default instance found")
	})

	t.Run("returns error when wallet storage resolution fails", func(t *testing.T) {
		// Arrange
		cfg := config.WalletWatch{
			WalletStorage: storage.Picker{
				Engine: "INVALID_ENGINE",
			},
			TransactionNotifier: messaging.Picker{
				Engine: messaging.EngineRedis,
				MessagePublisher: messaging.MessagePublisher{
					Redis: &messaging.RedisPublisher{
						Stream: "wallet-transactions",
					},
				},
			},
		}

		// Act
		service, err := setupWalletWatch(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "INVALID_ENGINE")
	})

	t.Run("returns error when transaction notifier resolution fails", func(t *testing.T) {
		// Arrange
		cfg := config.WalletWatch{
			WalletStorage: storage.Picker{
				Engine: storage.EngineRedis,
			},
			TransactionNotifier: messaging.Picker{
				Engine: "INVALID_ENGINE",
				MessagePublisher: messaging.MessagePublisher{
					Redis: &messaging.RedisPublisher{
						Stream: "wallet-transactions",
					},
				},
			},
		}

		// Act
		service, err := setupWalletWatch(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		// Storage resolution fails first since it's checked before transaction notifier
		assert.Contains(t, err.Error(), "no default instance found")
	})

	t.Run("returns error when idempotency guard resolution fails", func(t *testing.T) {
		// Arrange
		idempotencyGuardPicker := &storage.Picker{
			Engine: "INVALID_ENGINE",
		}
		cfg := config.WalletWatch{
			WalletStorage: storage.Picker{
				Engine: storage.EngineRedis,
			},
			TransactionNotifier: messaging.Picker{
				Engine: messaging.EngineRedis,
				MessagePublisher: messaging.MessagePublisher{
					Redis: &messaging.RedisPublisher{
						Stream: "wallet-transactions",
					},
				},
			},
			IdempotencyGuard: idempotencyGuardPicker,
		}

		// Act
		service, err := setupWalletWatch(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		// Storage resolution fails first since it's checked before idempotency guard
		assert.Contains(t, err.Error(), "no default instance found")
	})

	t.Run("returns error when context is cancelled", func(t *testing.T) {
		// Arrange
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		cfg := config.WalletWatch{
			WalletStorage: storage.Picker{
				Engine: storage.EngineRedis,
			},
			TransactionNotifier: messaging.Picker{
				Engine: messaging.EngineRedis,
				MessagePublisher: messaging.MessagePublisher{
					Redis: &messaging.RedisPublisher{
						Stream: "wallet-transactions",
					},
				},
			},
		}

		// Act
		service, err := setupWalletWatch(ctx, cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "no default instance found")
	})

	t.Run("returns error with zero max processing time when storage not configured", func(t *testing.T) {
		// Arrange
		cfg := config.WalletWatch{
			MaxProcessingTime: 0, // Should not add the option when zero
			WalletStorage: storage.Picker{
				Engine: storage.EngineRedis,
			},
			TransactionNotifier: messaging.Picker{
				Engine: messaging.EngineRedis,
				MessagePublisher: messaging.MessagePublisher{
					Redis: &messaging.RedisPublisher{
						Stream: "wallet-transactions",
					},
				},
			},
		}

		// Act
		service, err := setupWalletWatch(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "no default instance found")
	})
}

func TestSetupWalletRegistry(t *testing.T) {
	t.Run("returns error when redis storage engine is not configured", func(t *testing.T) {
		// Arrange
		cfg := config.WalletRegistry{
			WalletStorage: storage.Picker{
				Engine: storage.EngineRedis,
			},
		}

		// Act
		service, err := setupWalletRegistry(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "no default instance found")
	})

	t.Run("returns error when postgresql storage engine is not configured", func(t *testing.T) {
		// Arrange
		cfg := config.WalletRegistry{
			WalletStorage: storage.Picker{
				Engine: storage.EnginePostgreSQL,
			},
		}

		// Act
		service, err := setupWalletRegistry(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "no default instance found")
	})

	t.Run("returns error when inline redis config cannot connect", func(t *testing.T) {
		// Arrange
		cfg := config.WalletRegistry{
			WalletStorage: storage.Picker{
				InlineConfig: storage.InlineConfig{
					Redis: &storage.Redis{
						Address:  "localhost:6379",
						Username: "",
						Password: "",
						DB:       0,
					},
				},
			},
		}

		// Act
		service, err := setupWalletRegistry(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("returns error when wallet storage resolution fails with invalid engine", func(t *testing.T) {
		// Arrange
		cfg := config.WalletRegistry{
			WalletStorage: storage.Picker{
				Engine: "INVALID_ENGINE",
			},
		}

		// Act
		service, err := setupWalletRegistry(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "INVALID_ENGINE")
	})

	t.Run("returns error when context is cancelled", func(t *testing.T) {
		// Arrange
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		cfg := config.WalletRegistry{
			WalletStorage: storage.Picker{
				Engine: storage.EngineRedis,
			},
		}

		// Act
		service, err := setupWalletRegistry(ctx, cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "no default instance found")
	})
}

func TestBuildJsonrpcClient(t *testing.T) {
	t.Run("creates jsonrpc client with default configuration", func(t *testing.T) {
		// Arrange
		cfg := pkgconfig.JsonRPC{
			HttpClient: pkgconfig.HttpClient{
				Timeout:      5 * time.Second,
				RetryWaitMin: 1 * time.Second,
				RetryWaitMax: 5 * time.Second,
				RetryMax:     2,
			},
			ProviderEndpoint: "https://mainnet.infura.io/v3/test-key",
		}

		// Act
		client := buildJsonrpcClient(cfg)

		// Assert
		assert.NotNil(t, client)
	})

	t.Run("creates jsonrpc client with custom configuration", func(t *testing.T) {
		// Arrange
		cfg := pkgconfig.JsonRPC{
			HttpClient: pkgconfig.HttpClient{
				Timeout:      30 * time.Second,
				RetryWaitMin: 2 * time.Second,
				RetryWaitMax: 10 * time.Second,
				RetryMax:     5,
			},
			ProviderEndpoint: "https://eth-mainnet.alchemyapi.io/v2/custom-key",
		}

		// Act
		client := buildJsonrpcClient(cfg)

		// Assert
		assert.NotNil(t, client)
	})

	t.Run("creates jsonrpc client with minimal retry configuration", func(t *testing.T) {
		// Arrange
		cfg := pkgconfig.JsonRPC{
			HttpClient: pkgconfig.HttpClient{
				Timeout:      1 * time.Second,
				RetryWaitMin: 100 * time.Millisecond,
				RetryWaitMax: 1 * time.Second,
				RetryMax:     0, // No retries
			},
			ProviderEndpoint: "https://localhost:8545",
		}

		// Act
		client := buildJsonrpcClient(cfg)

		// Assert
		assert.NotNil(t, client)
	})
}

func TestSetupChainStream(t *testing.T) {
	t.Run("successfully creates chainstream service with minimal config", func(t *testing.T) {
		// Arrange
		cfg := config.ChainStream{
			Networks: blockchainconfig.Networks{},
		}

		// Act
		service, err := setupChainStream(t.Context(), cfg)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.Implements(t, (*chainstream.Service)(nil), service)
	})

	t.Run("successfully creates chainstream service with ethereum network", func(t *testing.T) {
		// Arrange
		cfg := config.ChainStream{
			Networks: blockchainconfig.Networks{
				Ethereum: &pkgconfig.JsonRPC{
					HttpClient: pkgconfig.HttpClient{
						Timeout:      5 * time.Second,
						RetryWaitMin: 1 * time.Second,
						RetryWaitMax: 5 * time.Second,
						RetryMax:     2,
					},
					ProviderEndpoint: "https://mainnet.infura.io/v3/test",
				},
			},
		}

		// Act
		service, err := setupChainStream(t.Context(), cfg)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.Implements(t, (*chainstream.Service)(nil), service)
	})

	t.Run("successfully creates chainstream service with retry configuration", func(t *testing.T) {
		// Arrange
		cfg := config.ChainStream{
			Networks: blockchainconfig.Networks{},
			Retry: &pkgconfig.Retry{
				Attempts: 3,
				Delay:    1 * time.Second,
				MaxDelay: 5 * time.Second,
			},
		}

		// Act
		service, err := setupChainStream(t.Context(), cfg)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.Implements(t, (*chainstream.Service)(nil), service)
	})

	t.Run("returns error when redis checkpoint storage is not configured", func(t *testing.T) {
		// Arrange
		checkpointStoragePicker := &storage.Picker{
			Engine: storage.EngineRedis,
		}
		cfg := config.ChainStream{
			Networks:          blockchainconfig.Networks{},
			CheckpointStorage: checkpointStoragePicker,
		}

		// Act
		service, err := setupChainStream(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "no default instance found")
	})

	t.Run("returns error when redis dispatch failure notifier is not configured", func(t *testing.T) {
		// Arrange
		dispatchFailureNotifierPicker := &messaging.Picker{
			Engine: messaging.EngineRedis,
			MessagePublisher: messaging.MessagePublisher{
				Redis: &messaging.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}
		cfg := config.ChainStream{
			Networks:                blockchainconfig.Networks{},
			DispatchFailureNotifier: dispatchFailureNotifierPicker,
		}

		// Act
		service, err := setupChainStream(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "no default messaging instance found")
	})

	t.Run("returns error when redis storage and messaging are not configured", func(t *testing.T) {
		// Arrange
		checkpointStoragePicker := &storage.Picker{
			Engine: storage.EngineRedis,
		}
		dispatchFailureNotifierPicker := &messaging.Picker{
			Engine: messaging.EngineRedis,
			MessagePublisher: messaging.MessagePublisher{
				Redis: &messaging.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}
		cfg := config.ChainStream{
			Networks: blockchainconfig.Networks{
				Ethereum: &pkgconfig.JsonRPC{
					HttpClient: pkgconfig.HttpClient{
						Timeout:      10 * time.Second,
						RetryWaitMin: 500 * time.Millisecond,
						RetryWaitMax: 10 * time.Second,
						RetryMax:     5,
					},
					ProviderEndpoint: "https://eth-mainnet.alchemyapi.io/v2/test",
				},
			},
			Retry: &pkgconfig.Retry{
				Attempts: 5,
				Delay:    2 * time.Second,
				MaxDelay: 10 * time.Second,
			},
			CheckpointStorage:       checkpointStoragePicker,
			DispatchFailureNotifier: dispatchFailureNotifierPicker,
		}

		// Act
		service, err := setupChainStream(t.Context(), cfg)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, service)
		assert.Contains(t, err.Error(), "no default instance found")
	})

	t.Run("successfully creates chainstream service with cancelled context", func(t *testing.T) {
		// Arrange
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		cfg := config.ChainStream{
			Networks: blockchainconfig.Networks{},
		}

		// Act
		service, err := setupChainStream(ctx, cfg)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, service)
		assert.Implements(t, (*chainstream.Service)(nil), service)
	})
}
