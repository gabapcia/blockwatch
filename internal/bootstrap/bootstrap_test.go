package bootstrap

import (
	"testing"
	"time"

	"github.com/gabapcia/blockwatch/internal/bootstrap/messaging"
	"github.com/gabapcia/blockwatch/internal/bootstrap/storage"
	"github.com/gabapcia/blockwatch/internal/pkg/config"
	blockchainconfig "github.com/gabapcia/blockwatch/internal/pkg/config/blockchain"
	messagingconfig "github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	pkgconfig "github.com/gabapcia/blockwatch/internal/pkg/config/pkg"
	storageconfig "github.com/gabapcia/blockwatch/internal/pkg/config/storage"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

func setupRedisContainer(t *testing.T) storageconfig.Redis {
	t.Helper()

	ctx := t.Context()

	// Start Redis container
	redisContainer, err := rediscontainer.Run(ctx,
		"redis:8-alpine",
		rediscontainer.WithSnapshotting(10, 1),
		rediscontainer.WithLogLevel(rediscontainer.LogLevelVerbose),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		redisContainer.Terminate(ctx)
	})

	// Get connection details
	connectionString, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err)

	// Parse connection string to get host and port
	opts, err := redis.ParseURL(connectionString)
	require.NoError(t, err)

	return storageconfig.Redis{
		Address:  opts.Addr,
		Username: opts.Username,
		Password: opts.Password,
		DB:       opts.DB,
	}
}

func TestSetupChainStream(t *testing.T) {
	t.Run("with all options", func(t *testing.T) {
		redisContainerCfg := setupRedisContainer(t)

		cfg := config.ChainStream{
			Networks: blockchainconfig.Networks{
				Ethereum: &pkgconfig.JsonRPC{
					ProviderEndpoint: "http://localhost:8545",
				},
				Solana: &pkgconfig.JsonRPC{
					ProviderEndpoint: "http://localhost:8899",
				},
			},
			Retry: &pkgconfig.Retry{
				Attempts: 5,
				Delay:    1 * time.Second,
				MaxDelay: 10 * time.Second,
			},
			CheckpointStorage: &storageconfig.Picker{
				Engine: storageconfig.EngineRedis,
			},
			DispatchFailureNotifier: &messagingconfig.Picker{
				Engine: messagingconfig.EngineRedis,
				MessagePublisher: messagingconfig.MessagePublisher{
					Redis: &messagingconfig.RedisPublisher{
						Stream: "test-stream",
					},
				},
			},
		}

		err := storage.Init(t.Context(), storageconfig.Engines{
			Redis: &redisContainerCfg,
		})
		require.NoError(t, err)
		defer storage.Close()

		err = messaging.Init(t.Context(), messagingconfig.Engines{
			Redis: &messagingconfig.RedisConnection{
				Address: redisContainerCfg.Address,
			},
		})
		require.NoError(t, err)
		defer messaging.Close()

		_, err = setupChainStream(t.Context(), cfg)
		assert.NoError(t, err)
	})

	t.Run("with storage resolver error", func(t *testing.T) {
		cfg := config.ChainStream{
			CheckpointStorage: &storageconfig.Picker{
				Engine: "invalid",
			},
		}

		_, err := setupChainStream(t.Context(), cfg)
		assert.Error(t, err)
	})

	t.Run("with messaging resolver error", func(t *testing.T) {
		cfg := config.ChainStream{
			DispatchFailureNotifier: &messagingconfig.Picker{
				Engine: "invalid",
			},
		}

		_, err := setupChainStream(t.Context(), cfg)
		assert.Error(t, err)
	})
}

func TestSetupWalletWatch(t *testing.T) {
	t.Run("with all options", func(t *testing.T) {
		redisContainerCfg := setupRedisContainer(t)

		cfg := config.WalletWatch{
			WalletStorage: storageconfig.Picker{
				Engine: storageconfig.EngineRedis,
			},
			TransactionNotifier: messagingconfig.Picker{
				Engine: messagingconfig.EngineRedis,
				MessagePublisher: messagingconfig.MessagePublisher{
					Redis: &messagingconfig.RedisPublisher{
						Stream: "test-stream",
					},
				},
			},
			MaxProcessingTime: 5 * time.Second,
			IdempotencyGuard: &storageconfig.Picker{
				Engine: storageconfig.EngineRedis,
			},
		}

		err := storage.Init(t.Context(), storageconfig.Engines{
			Redis: &redisContainerCfg,
		})
		require.NoError(t, err)
		defer storage.Close()

		err = messaging.Init(t.Context(), messagingconfig.Engines{
			Redis: &messagingconfig.RedisConnection{
				Address: redisContainerCfg.Address,
			},
		})
		require.NoError(t, err)
		defer messaging.Close()

		_, err = setupWalletWatch(t.Context(), cfg)
		assert.NoError(t, err)
	})

	t.Run("with wallet storage resolver error", func(t *testing.T) {
		cfg := config.WalletWatch{
			WalletStorage: storageconfig.Picker{
				Engine: "invalid",
			},
		}

		_, err := setupWalletWatch(t.Context(), cfg)
		assert.Error(t, err)
	})

	t.Run("with transaction notifier resolver error", func(t *testing.T) {
		cfg := config.WalletWatch{
			WalletStorage: storageconfig.Picker{
				Engine: storageconfig.EngineRedis,
			},
			TransactionNotifier: messagingconfig.Picker{
				Engine: "invalid",
			},
		}

		redisContainerCfg := setupRedisContainer(t)
		err := storage.Init(t.Context(), storageconfig.Engines{
			Redis: &redisContainerCfg,
		})
		require.NoError(t, err)
		defer storage.Close()

		_, err = setupWalletWatch(t.Context(), cfg)
		assert.Error(t, err)
	})

	t.Run("with idempotency guard resolver error", func(t *testing.T) {
		cfg := config.WalletWatch{
			WalletStorage: storageconfig.Picker{
				Engine: storageconfig.EngineRedis,
			},
			TransactionNotifier: messagingconfig.Picker{
				Engine: messagingconfig.EngineRedis,
				MessagePublisher: messagingconfig.MessagePublisher{
					Redis: &messagingconfig.RedisPublisher{
						Stream: "test-stream",
					},
				},
			},
			IdempotencyGuard: &storageconfig.Picker{
				Engine: "invalid",
			},
		}

		redisContainerCfg := setupRedisContainer(t)
		err := storage.Init(t.Context(), storageconfig.Engines{
			Redis: &redisContainerCfg,
		})
		require.NoError(t, err)
		defer storage.Close()

		err = messaging.Init(t.Context(), messagingconfig.Engines{
			Redis: &messagingconfig.RedisConnection{
				Address: redisContainerCfg.Address,
			},
		})
		require.NoError(t, err)
		defer messaging.Close()

		_, err = setupWalletWatch(t.Context(), cfg)
		assert.Error(t, err)
	})
}

func TestSetupWalletRegistry(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		redisContainerCfg := setupRedisContainer(t)

		cfg := config.WalletRegistry{
			WalletStorage: storageconfig.Picker{
				Engine: storageconfig.EngineRedis,
			},
		}

		err := storage.Init(t.Context(), storageconfig.Engines{
			Redis: &redisContainerCfg,
		})
		require.NoError(t, err)
		defer storage.Close()

		_, err = setupWalletRegistry(t.Context(), cfg)
		assert.NoError(t, err)
	})

	t.Run("with resolver error", func(t *testing.T) {
		cfg := config.WalletRegistry{
			WalletStorage: storageconfig.Picker{
				Engine: "invalid",
			},
		}

		_, err := setupWalletRegistry(t.Context(), cfg)
		assert.Error(t, err)
	})
}

func TestNew(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		redisContainerCfg := setupRedisContainer(t)
		cfg := config.Config{
			Engines: config.Engines{
				Storage: storageconfig.Engines{
					Redis: &redisContainerCfg,
				},
				Messaging: messagingconfig.Engines{
					Redis: &messagingconfig.RedisConnection{
						Address: redisContainerCfg.Address,
					},
				},
			},
			Chainstream: config.ChainStream{
				CheckpointStorage: &storageconfig.Picker{
					Engine: storageconfig.EngineRedis,
				},
			},
			Walletwatch: config.WalletWatch{
				WalletStorage: storageconfig.Picker{
					Engine: storageconfig.EngineRedis,
				},
				TransactionNotifier: messagingconfig.Picker{
					Engine: messagingconfig.EngineRedis,
					MessagePublisher: messagingconfig.MessagePublisher{
						Redis: &messagingconfig.RedisPublisher{
							Stream: "test-stream",
						},
					},
				},
			},
			Walletregistry: config.WalletRegistry{
				WalletStorage: storageconfig.Picker{
					Engine: storageconfig.EngineRedis,
				},
			},
		}

		_, err := New(t.Context(), cfg)
		assert.NoError(t, err)
	})

	t.Run("with storage init error", func(t *testing.T) {
		cfg := config.Config{
			Engines: config.Engines{
				Storage: storageconfig.Engines{
					Redis: &storageconfig.Redis{},
				},
			},
		}

		_, err := New(t.Context(), cfg)
		assert.Error(t, err)
	})

	t.Run("with messaging init error", func(t *testing.T) {
		redisContainerCfg := setupRedisContainer(t)
		cfg := config.Config{
			Engines: config.Engines{
				Storage: storageconfig.Engines{
					Redis: &redisContainerCfg,
				},
				Messaging: messagingconfig.Engines{
					Redis: &messagingconfig.RedisConnection{},
				},
			},
		}

		_, err := New(t.Context(), cfg)
		assert.Error(t, err)
	})

	t.Run("with setupChainStream error", func(t *testing.T) {
		redisContainerCfg := setupRedisContainer(t)
		cfg := config.Config{
			Engines: config.Engines{
				Storage: storageconfig.Engines{
					Redis: &redisContainerCfg,
				},
				Messaging: messagingconfig.Engines{
					Redis: &messagingconfig.RedisConnection{
						Address: redisContainerCfg.Address,
					},
				},
			},
			Chainstream: config.ChainStream{
				CheckpointStorage: &storageconfig.Picker{
					Engine: "invalid",
				},
			},
		}

		_, err := New(t.Context(), cfg)
		assert.Error(t, err)
	})

	t.Run("with setupWalletWatch error", func(t *testing.T) {
		redisContainerCfg := setupRedisContainer(t)
		cfg := config.Config{
			Engines: config.Engines{
				Storage: storageconfig.Engines{
					Redis: &redisContainerCfg,
				},
				Messaging: messagingconfig.Engines{
					Redis: &messagingconfig.RedisConnection{
						Address: redisContainerCfg.Address,
					},
				},
			},
			Chainstream: config.ChainStream{
				CheckpointStorage: &storageconfig.Picker{
					Engine: storageconfig.EngineRedis,
				},
			},
			Walletwatch: config.WalletWatch{
				WalletStorage: storageconfig.Picker{
					Engine: "invalid",
				},
			},
		}

		_, err := New(t.Context(), cfg)
		assert.Error(t, err)
	})

	t.Run("with setupWalletRegistry error", func(t *testing.T) {
		redisContainerCfg := setupRedisContainer(t)
		cfg := config.Config{
			Engines: config.Engines{
				Storage: storageconfig.Engines{
					Redis: &redisContainerCfg,
				},
				Messaging: messagingconfig.Engines{
					Redis: &messagingconfig.RedisConnection{
						Address: redisContainerCfg.Address,
					},
				},
			},
			Chainstream: config.ChainStream{
				CheckpointStorage: &storageconfig.Picker{
					Engine: storageconfig.EngineRedis,
				},
			},
			Walletwatch: config.WalletWatch{
				WalletStorage: storageconfig.Picker{
					Engine: storageconfig.EngineRedis,
				},
				TransactionNotifier: messagingconfig.Picker{
					Engine: messagingconfig.EngineRedis,
					MessagePublisher: messagingconfig.MessagePublisher{
						Redis: &messagingconfig.RedisPublisher{
							Stream: "test-stream",
						},
					},
				},
			},
			Walletregistry: config.WalletRegistry{
				WalletStorage: storageconfig.Picker{
					Engine: "invalid",
				},
			},
		}

		_, err := New(t.Context(), cfg)
		assert.Error(t, err)
	})
}

func TestBootstrap_Close(t *testing.T) {
	redisContainerCfg := setupRedisContainer(t)
	cfg := config.Config{
		Engines: config.Engines{
			Storage: storageconfig.Engines{
				Redis: &redisContainerCfg,
			},
			Messaging: messagingconfig.Engines{
				Redis: &messagingconfig.RedisConnection{
					Address: redisContainerCfg.Address,
				},
			},
		},
		Chainstream: config.ChainStream{
			CheckpointStorage: &storageconfig.Picker{
				Engine: storageconfig.EngineRedis,
			},
		},
		Walletwatch: config.WalletWatch{
			WalletStorage: storageconfig.Picker{
				Engine: storageconfig.EngineRedis,
			},
			TransactionNotifier: messagingconfig.Picker{
				Engine: messagingconfig.EngineRedis,
				MessagePublisher: messagingconfig.MessagePublisher{
					Redis: &messagingconfig.RedisPublisher{
						Stream: "test-stream",
					},
				},
			},
		},
		Walletregistry: config.WalletRegistry{
			WalletStorage: storageconfig.Picker{
				Engine: storageconfig.EngineRedis,
			},
		},
	}

	b, err := New(t.Context(), cfg)
	require.NoError(t, err)

	err = b.Close()
	assert.NoError(t, err)
}
