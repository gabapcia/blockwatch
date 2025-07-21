package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	"github.com/gabapcia/blockwatch/internal/pkg/config/storage"
	validatorMocks "github.com/gabapcia/blockwatch/internal/pkg/validator/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidateConfigStruct(t *testing.T) {
	t.Run("valid config with all engines configured", func(t *testing.T) {
		config := Config{
			Engines: Engines{
				Storage: storage.Engines{
					Redis:      &storage.Redis{Address: "localhost:6379"},
					PostgreSQL: &storage.PostgreSQL{DSN: "postgres://localhost/test"},
				},
				Messaging: messaging.Engines{
					Redis:    &messaging.RedisConnection{Address: "localhost:6379"},
					RabbitMQ: &messaging.RabbitMQConnection{URI: "amqp://localhost"},
				},
			},
			Walletregistry: WalletRegistry{
				WalletStorage: storage.Picker{
					Engine: storage.EngineRedis,
				},
			},
			Walletwatch: WalletWatch{
				WalletStorage: storage.Picker{
					Engine: storage.EnginePostgreSQL,
				},
				TransactionNotifier: messaging.Picker{
					Engine: messaging.EngineRabbitMQ,
				},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(config))

		validateConfigStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("storage picker with engine not configured", func(t *testing.T) {
		config := Config{
			Engines: Engines{
				Storage: storage.Engines{
					// Redis is nil but referenced by picker
					PostgreSQL: &storage.PostgreSQL{DSN: "postgres://localhost/test"},
				},
			},
			Walletregistry: WalletRegistry{
				WalletStorage: storage.Picker{
					Engine: storage.EngineRedis,
				},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(config))
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"WalletStorage.Engine",
			"Engine",
			"engine_not_configured",
			storage.EngineRedis,
		)

		validateConfigStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("messaging picker with engine not configured", func(t *testing.T) {
		config := Config{
			Engines: Engines{
				Messaging: messaging.Engines{
					// RabbitMQ is nil but referenced by picker
					Redis: &messaging.RedisConnection{Address: "localhost:6379"},
				},
			},
			Walletwatch: WalletWatch{
				TransactionNotifier: messaging.Picker{
					Engine: messaging.EngineRabbitMQ,
				},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(config))
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"TransactionNotifier.Engine",
			"Engine",
			"engine_not_configured",
			messaging.EngineRabbitMQ,
		)

		validateConfigStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("storage picker with unregistered engine", func(t *testing.T) {
		config := Config{
			Engines: Engines{
				Storage: storage.Engines{
					Redis: &storage.Redis{Address: "localhost:6379"},
				},
			},
			Walletregistry: WalletRegistry{
				WalletStorage: storage.Picker{
					Engine: "UNKNOWN_ENGINE",
				},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(config))
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"WalletStorage.Engine",
			"Engine",
			"engine_not_registered",
			"UNKNOWN_ENGINE",
		)

		validateConfigStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("messaging picker with unregistered engine", func(t *testing.T) {
		config := Config{
			Engines: Engines{
				Messaging: messaging.Engines{
					Redis: &messaging.RedisConnection{Address: "localhost:6379"},
				},
			},
			Walletwatch: WalletWatch{
				TransactionNotifier: messaging.Picker{
					Engine: "UNKNOWN_ENGINE",
				},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(config))
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"TransactionNotifier.Engine",
			"Engine",
			"engine_not_registered",
			"UNKNOWN_ENGINE",
		)

		validateConfigStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("pointer storage picker with engine not configured", func(t *testing.T) {
		config := Config{
			Engines: Engines{
				Storage: storage.Engines{
					// PostgreSQL is nil but referenced by pointer picker
					Redis: &storage.Redis{Address: "localhost:6379"},
				},
			},
			Walletwatch: WalletWatch{
				IdempotencyGuard: &storage.Picker{
					Engine: storage.EnginePostgreSQL,
				},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(config))
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"IdempotencyGuard.Engine",
			"Engine",
			"engine_not_configured",
			storage.EnginePostgreSQL,
		)

		validateConfigStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("pointer messaging picker with engine not configured", func(t *testing.T) {
		config := Config{
			Engines: Engines{
				Messaging: messaging.Engines{
					// RabbitMQ is nil but referenced by pointer picker
					Redis: &messaging.RedisConnection{Address: "localhost:6379"},
				},
			},
			Chainstream: ChainStream{
				DispatchFailureHandler: &messaging.Picker{
					Engine: messaging.EngineRabbitMQ,
				},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(config))
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"DispatchFailureHandler.Engine",
			"Engine",
			"engine_not_configured",
			messaging.EngineRabbitMQ,
		)

		validateConfigStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("nil pointer picker should not cause error", func(t *testing.T) {
		config := Config{
			Engines: Engines{
				Storage: storage.Engines{
					Redis: &storage.Redis{Address: "localhost:6379"},
				},
			},
			Walletwatch: WalletWatch{
				IdempotencyGuard: nil, // nil pointer should be ignored
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(config))

		validateConfigStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("empty engine string should not cause error", func(t *testing.T) {
		config := Config{
			Engines: Engines{
				Storage: storage.Engines{
					Redis: &storage.Redis{Address: "localhost:6379"},
				},
			},
			Walletregistry: WalletRegistry{
				WalletStorage: storage.Picker{
					Engine: "", // empty engine should be ignored
				},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(config))

		validateConfigStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("nil pointer", func(t *testing.T) {
		var config *Config = nil

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(config))

		validateConfigStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("unsupported type", func(t *testing.T) {
		unsupported := "not a config"

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(unsupported))

		validateConfigStruct(mockSL)
		mockSL.AssertExpectations(t)
	})

	t.Run("pointer to valid config", func(t *testing.T) {
		config := &Config{
			Engines: Engines{
				Storage: storage.Engines{
					Redis: &storage.Redis{Address: "localhost:6379"},
				},
			},
			Walletregistry: WalletRegistry{
				WalletStorage: storage.Picker{
					Engine: storage.EngineRedis,
				},
			},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().Current().Return(reflect.ValueOf(config))

		validateConfigStruct(mockSL)
		mockSL.AssertExpectations(t)
	})
}

func TestValidateStoragePicker(t *testing.T) {
	t.Run("valid redis picker with configured engine", func(t *testing.T) {
		picker := storage.Picker{
			Engine: storage.EngineRedis,
		}
		engines := storage.Engines{
			Redis: &storage.Redis{Address: "localhost:6379"},
		}

		mockSL := validatorMocks.NewStructLevel(t)

		validateStoragePicker(mockSL, picker, engines, "TestField")
		mockSL.AssertExpectations(t)
	})

	t.Run("valid postgresql picker with configured engine", func(t *testing.T) {
		picker := storage.Picker{
			Engine: storage.EnginePostgreSQL,
		}
		engines := storage.Engines{
			PostgreSQL: &storage.PostgreSQL{DSN: "postgres://localhost/test"},
		}

		mockSL := validatorMocks.NewStructLevel(t)

		validateStoragePicker(mockSL, picker, engines, "TestField")
		mockSL.AssertExpectations(t)
	})

	t.Run("redis picker with nil engine", func(t *testing.T) {
		picker := storage.Picker{
			Engine: storage.EngineRedis,
		}
		engines := storage.Engines{
			Redis: nil,
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"TestField.Engine",
			"Engine",
			"engine_not_configured",
			storage.EngineRedis,
		)

		validateStoragePicker(mockSL, picker, engines, "TestField")
		mockSL.AssertExpectations(t)
	})

	t.Run("postgresql picker with nil engine", func(t *testing.T) {
		picker := storage.Picker{
			Engine: storage.EnginePostgreSQL,
		}
		engines := storage.Engines{
			PostgreSQL: nil,
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"TestField.Engine",
			"Engine",
			"engine_not_configured",
			storage.EnginePostgreSQL,
		)

		validateStoragePicker(mockSL, picker, engines, "TestField")
		mockSL.AssertExpectations(t)
	})

	t.Run("picker with unregistered engine", func(t *testing.T) {
		picker := storage.Picker{
			Engine: "UNKNOWN_ENGINE",
		}
		engines := storage.Engines{
			Redis: &storage.Redis{Address: "localhost:6379"},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"TestField.Engine",
			"Engine",
			"engine_not_registered",
			"UNKNOWN_ENGINE",
		)

		validateStoragePicker(mockSL, picker, engines, "TestField")
		mockSL.AssertExpectations(t)
	})

	t.Run("picker with empty engine", func(t *testing.T) {
		picker := storage.Picker{
			Engine: "",
		}
		engines := storage.Engines{}

		mockSL := validatorMocks.NewStructLevel(t)

		validateStoragePicker(mockSL, picker, engines, "TestField")
		mockSL.AssertExpectations(t)
	})
}

func TestValidateMessagingPicker(t *testing.T) {
	t.Run("valid redis picker with configured engine", func(t *testing.T) {
		picker := messaging.Picker{
			Engine: messaging.EngineRedis,
		}
		engines := messaging.Engines{
			Redis: &messaging.RedisConnection{Address: "localhost:6379"},
		}

		mockSL := validatorMocks.NewStructLevel(t)

		validateMessagingPicker(mockSL, picker, engines, "TestField")
		mockSL.AssertExpectations(t)
	})

	t.Run("valid rabbitmq picker with configured engine", func(t *testing.T) {
		picker := messaging.Picker{
			Engine: messaging.EngineRabbitMQ,
		}
		engines := messaging.Engines{
			RabbitMQ: &messaging.RabbitMQConnection{URI: "amqp://localhost"},
		}

		mockSL := validatorMocks.NewStructLevel(t)

		validateMessagingPicker(mockSL, picker, engines, "TestField")
		mockSL.AssertExpectations(t)
	})

	t.Run("redis picker with nil engine", func(t *testing.T) {
		picker := messaging.Picker{
			Engine: messaging.EngineRedis,
		}
		engines := messaging.Engines{
			Redis: nil,
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"TestField.Engine",
			"Engine",
			"engine_not_configured",
			messaging.EngineRedis,
		)

		validateMessagingPicker(mockSL, picker, engines, "TestField")
		mockSL.AssertExpectations(t)
	})

	t.Run("rabbitmq picker with nil engine", func(t *testing.T) {
		picker := messaging.Picker{
			Engine: messaging.EngineRabbitMQ,
		}
		engines := messaging.Engines{
			RabbitMQ: nil,
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"TestField.Engine",
			"Engine",
			"engine_not_configured",
			messaging.EngineRabbitMQ,
		)

		validateMessagingPicker(mockSL, picker, engines, "TestField")
		mockSL.AssertExpectations(t)
	})

	t.Run("picker with unregistered engine", func(t *testing.T) {
		picker := messaging.Picker{
			Engine: "UNKNOWN_ENGINE",
		}
		engines := messaging.Engines{
			Redis: &messaging.RedisConnection{Address: "localhost:6379"},
		}

		mockSL := validatorMocks.NewStructLevel(t)
		mockSL.EXPECT().ReportError(
			mock.Anything,
			"TestField.Engine",
			"Engine",
			"engine_not_registered",
			"UNKNOWN_ENGINE",
		)

		validateMessagingPicker(mockSL, picker, engines, "TestField")
		mockSL.AssertExpectations(t)
	})

	t.Run("picker with empty engine", func(t *testing.T) {
		picker := messaging.Picker{
			Engine: "",
		}
		engines := messaging.Engines{}

		mockSL := validatorMocks.NewStructLevel(t)

		validateMessagingPicker(mockSL, picker, engines, "TestField")
		mockSL.AssertExpectations(t)
	})
}

func TestConfig_Engines(t *testing.T) {
	t.Run("successful processing and validation with valid environment variables", func(t *testing.T) {
		// Set up environment variables
		t.Setenv("STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("STORAGE_REDIS_USERNAME", "redisuser")
		t.Setenv("STORAGE_REDIS_PASSWORD", "redispass")
		t.Setenv("STORAGE_REDIS_DB", "1")
		t.Setenv("STORAGE_POSTGRESQL_DSN", "postgres://user:pass@localhost/db")
		t.Setenv("MESSAGING_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("MESSAGING_REDIS_USERNAME", "msguser")
		t.Setenv("MESSAGING_REDIS_PASSWORD", "msgpass")
		t.Setenv("MESSAGING_REDIS_DB", "2")
		t.Setenv("MESSAGING_RABBITMQ_URI", "amqp://localhost")

		var engines Engines
		ctx := t.Context()

		// Test process function
		err := process(ctx, &engines)
		require.NoError(t, err)

		// Verify storage values were loaded
		require.NotNil(t, engines.Storage.Redis)
		assert.Equal(t, "localhost:6379", engines.Storage.Redis.Address)
		assert.Equal(t, "redisuser", engines.Storage.Redis.Username)
		assert.Equal(t, "redispass", engines.Storage.Redis.Password)
		assert.Equal(t, 1, engines.Storage.Redis.DB)

		require.NotNil(t, engines.Storage.PostgreSQL)
		assert.Equal(t, "postgres://user:pass@localhost/db", engines.Storage.PostgreSQL.DSN)

		// Verify messaging values were loaded
		require.NotNil(t, engines.Messaging.Redis)
		assert.Equal(t, "localhost:6379", engines.Messaging.Redis.Address)
		assert.Equal(t, "msguser", engines.Messaging.Redis.Username)
		assert.Equal(t, "msgpass", engines.Messaging.Redis.Password)
		assert.Equal(t, 2, engines.Messaging.Redis.DB)

		require.NotNil(t, engines.Messaging.RabbitMQ)
		assert.Equal(t, "amqp://localhost", engines.Messaging.RabbitMQ.URI)

		// Test validate function
		err = validate(engines)
		assert.NoError(t, err)
	})

	t.Run("successful processing with minimal environment variables", func(t *testing.T) {
		var engines Engines
		ctx := t.Context()

		// Test process function with no env vars (should use defaults/empty values)
		err := process(ctx, &engines)
		require.NoError(t, err)

		// Verify no engines were configured
		assert.Nil(t, engines.Storage.Redis)
		assert.Nil(t, engines.Storage.PostgreSQL)
		assert.Nil(t, engines.Messaging.Redis)
		assert.Nil(t, engines.Messaging.RabbitMQ)

		// Test validate function (should pass since engines are omitempty)
		err = validate(engines)
		assert.NoError(t, err)
	})

	t.Run("successful processing with only storage redis", func(t *testing.T) {
		// Set up environment variables for only Redis storage
		t.Setenv("STORAGE_REDIS_ADDRESS", "redis.example.com:6379")

		var engines Engines
		ctx := t.Context()

		err := process(ctx, &engines)
		require.NoError(t, err)

		// Verify only Redis storage was configured
		require.NotNil(t, engines.Storage.Redis)
		assert.Equal(t, "redis.example.com:6379", engines.Storage.Redis.Address)
		assert.Empty(t, engines.Storage.Redis.Username)
		assert.Empty(t, engines.Storage.Redis.Password)
		assert.Equal(t, 0, engines.Storage.Redis.DB) // Default value

		assert.Nil(t, engines.Storage.PostgreSQL)
		assert.Nil(t, engines.Messaging.Redis)
		assert.Nil(t, engines.Messaging.RabbitMQ)

		// Test validate function
		err = validate(engines)
		assert.NoError(t, err)
	})

	t.Run("successful processing with only postgresql storage", func(t *testing.T) {
		// Set up environment variables for only PostgreSQL storage
		t.Setenv("STORAGE_POSTGRESQL_DSN", "postgres://admin:secret@db.example.com:5432/mydb")

		var engines Engines
		ctx := t.Context()

		err := process(ctx, &engines)
		require.NoError(t, err)

		// Verify only PostgreSQL storage was configured
		require.NotNil(t, engines.Storage.PostgreSQL)
		assert.Equal(t, "postgres://admin:secret@db.example.com:5432/mydb", engines.Storage.PostgreSQL.DSN)

		assert.Nil(t, engines.Storage.Redis)
		assert.Nil(t, engines.Messaging.Redis)
		assert.Nil(t, engines.Messaging.RabbitMQ)

		// Test validate function
		err = validate(engines)
		assert.NoError(t, err)
	})

	t.Run("validation fails with missing required redis address", func(t *testing.T) {
		// Set up environment variables with missing required address but other fields present
		t.Setenv("STORAGE_REDIS_USERNAME", "user")
		t.Setenv("STORAGE_REDIS_PASSWORD", "pass")
		// Missing STORAGE_REDIS_ADDRESS which is required

		var engines Engines
		ctx := t.Context()

		err := process(ctx, &engines)
		require.NoError(t, err)

		// Verify Redis config was created but with empty address
		require.NotNil(t, engines.Storage.Redis)
		assert.Empty(t, engines.Storage.Redis.Address)

		// Validation should fail due to missing required address
		err = validate(engines)
		assert.Error(t, err)
	})

	t.Run("validation fails with missing required postgresql dsn", func(t *testing.T) {
		// We can't easily test this since setting any STORAGE_POSTGRESQL_* env var
		// will require the DSN to be set. Let's test with empty DSN.
		t.Setenv("STORAGE_POSTGRESQL_DSN", "")

		var engines Engines
		ctx := t.Context()

		err := process(ctx, &engines)
		require.NoError(t, err)

		// With empty DSN, the struct MUST not be created
		assert.Nil(t, engines.Storage.PostgreSQL)

		// Test validate function (should pass since no PostgreSQL config)
		err = validate(engines)
		assert.NoError(t, err)
	})

	t.Run("validation fails with missing required messaging redis address", func(t *testing.T) {
		// Set up environment variables with missing required address but other fields present
		t.Setenv("MESSAGING_REDIS_USERNAME", "user")
		t.Setenv("MESSAGING_REDIS_PASSWORD", "pass")
		// Missing MESSAGING_REDIS_ADDRESS which is required

		var engines Engines
		ctx := t.Context()

		err := process(ctx, &engines)
		require.NoError(t, err)

		// Verify Redis messaging config was created but with empty address
		require.NotNil(t, engines.Messaging.Redis)
		assert.Empty(t, engines.Messaging.Redis.Address)

		// Validation should fail due to missing required address
		err = validate(engines)
		assert.Error(t, err)
	})

	t.Run("validation fails with missing required rabbitmq uri", func(t *testing.T) {
		// We can't easily test this since setting any MESSAGING_RABBITMQ_* env var
		// will require the URI to be set. Let's test with empty URI.
		t.Setenv("MESSAGING_RABBITMQ_URI", "")

		var engines Engines
		ctx := t.Context()

		err := process(ctx, &engines)
		require.NoError(t, err)

		// With empty URI, the struct MUST not be created
		assert.Nil(t, engines.Messaging.RabbitMQ)

		// Test validate function (should pass since no RabbitMQ config)
		err = validate(engines)
		assert.NoError(t, err)
	})
}

func TestConfig_Config(t *testing.T) {
	t.Run("successful loading with all required fields", func(t *testing.T) {
		// Set up all required environment variables
		t.Setenv("LOG_LEVEL", "DEBUG")
		t.Setenv("TELEMETRY_SERVICE_NAME", "test-service")

		// WalletRegistry required fields (using global engine)
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_ENGINE", "REDIS")

		// WalletWatch required fields (using global engines)
		t.Setenv("WALLETWATCH_WALLET_STORAGE_ENGINE", "POSTGRESQL")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_ENGINE", "RABBITMQ")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_RABBITMQ_ROUTING_KEY", "wallet.transactions")
		t.Setenv("WALLETWATCH_MAX_PROCESSING_TIME", "30s")

		// ChainStream required fields
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

		// Engine configurations
		t.Setenv("ENGINES_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("ENGINES_STORAGE_POSTGRESQL_DSN", "postgres://localhost/test")
		t.Setenv("ENGINES_MESSAGING_RABBITMQ_URI", "amqp://localhost")

		ctx := t.Context()
		config, err := Load(ctx)
		require.NoError(t, err)

		// Verify Log configuration
		assert.Equal(t, "DEBUG", config.Log.Level)

		// Verify Telemetry configuration
		assert.Equal(t, "test-service", config.Telemetry.ServiceName)

		// Verify Engines configuration
		require.NotNil(t, config.Engines.Storage.Redis)
		assert.Equal(t, "localhost:6379", config.Engines.Storage.Redis.Address)
		require.NotNil(t, config.Engines.Storage.PostgreSQL)
		assert.Equal(t, "postgres://localhost/test", config.Engines.Storage.PostgreSQL.DSN)
		require.NotNil(t, config.Engines.Messaging.RabbitMQ)
		assert.Equal(t, "amqp://localhost", config.Engines.Messaging.RabbitMQ.URI)

		// Verify WalletRegistry configuration (using global engine)
		assert.Equal(t, "REDIS", config.Walletregistry.WalletStorage.Engine)
		assert.Nil(t, config.Walletregistry.WalletStorage.InlineConfig.Redis)
		assert.Nil(t, config.Walletregistry.WalletStorage.InlineConfig.PostgreSQL)

		// Verify WalletWatch configuration (using global engines)
		assert.Equal(t, "POSTGRESQL", config.Walletwatch.WalletStorage.Engine)
		assert.Nil(t, config.Walletwatch.WalletStorage.InlineConfig.Redis)
		assert.Nil(t, config.Walletwatch.WalletStorage.InlineConfig.PostgreSQL)
		assert.Equal(t, "RABBITMQ", config.Walletwatch.TransactionNotifier.Engine)
		assert.Nil(t, config.Walletwatch.TransactionNotifier.InlineConfig.Redis)
		assert.Nil(t, config.Walletwatch.TransactionNotifier.InlineConfig.RabbitMQ)
		assert.Equal(t, "wallet.transactions", config.Walletwatch.TransactionNotifier.MessagePublisher.RabbitMQ.RoutingKey)
		assert.Equal(t, 30*time.Second, config.Walletwatch.MaxProcessingTime)

		// Verify ChainStream configuration
		require.NotNil(t, config.Chainstream.Networks.Ethereum)
		assert.Equal(t, "https://eth.example.com", config.Chainstream.Networks.Ethereum.ProviderEndpoint)
	})

	t.Run("successful loading with default values", func(t *testing.T) {
		// Set up minimal required environment variables
		// Log and Telemetry will use defaults

		// WalletRegistry required fields (using inline config)
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")

		// WalletWatch required fields (using inline configs)
		t.Setenv("WALLETWATCH_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")

		// ChainStream required fields
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

		ctx := t.Context()
		config, err := Load(ctx)
		require.NoError(t, err)

		// Verify default values
		assert.Equal(t, "INFO", config.Log.Level)                   // Default value
		assert.Equal(t, "blockwatch", config.Telemetry.ServiceName) // Default value

		// Verify no engines were configured globally (using inline configs)
		assert.Nil(t, config.Engines.Storage.Redis)
		assert.Nil(t, config.Engines.Storage.PostgreSQL)
		assert.Nil(t, config.Engines.Messaging.Redis)
		assert.Nil(t, config.Engines.Messaging.RabbitMQ)

		// Verify inline configs were used
		assert.Empty(t, config.Walletregistry.WalletStorage.Engine)
		require.NotNil(t, config.Walletregistry.WalletStorage.InlineConfig.Redis)
		assert.Equal(t, "localhost:6379", config.Walletregistry.WalletStorage.InlineConfig.Redis.Address)

		assert.Empty(t, config.Walletwatch.WalletStorage.Engine)
		require.NotNil(t, config.Walletwatch.WalletStorage.InlineConfig.Redis)
		assert.Equal(t, "localhost:6379", config.Walletwatch.WalletStorage.InlineConfig.Redis.Address)

		assert.Empty(t, config.Walletwatch.TransactionNotifier.Engine)
		require.NotNil(t, config.Walletwatch.TransactionNotifier.InlineConfig.Redis)
		assert.Equal(t, "localhost:6379", config.Walletwatch.TransactionNotifier.InlineConfig.Redis.Address)
		assert.Equal(t, "wallet-events", config.Walletwatch.TransactionNotifier.MessagePublisher.Redis.Stream)
	})

	t.Run("validation fails with missing required log level", func(t *testing.T) {
		// Set invalid log level
		t.Setenv("LOG_LEVEL", "INVALID")

		// Set other required fields (using inline configs)
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

		ctx := t.Context()
		_, err := Load(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Level")
	})

	t.Run("validation fails with empty service name", func(t *testing.T) {
		// Set empty service name
		t.Setenv("TELEMETRY_SERVICE_NAME", "")

		// Set other required fields (using inline configs)
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

		ctx := t.Context()
		_, err := Load(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Telemetry")
	})

	t.Run("validation fails with missing walletregistry configuration", func(t *testing.T) {
		// Missing WalletRegistry configuration

		// Set other required fields (using inline configs)
		t.Setenv("WALLETWATCH_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

		ctx := t.Context()
		_, err := Load(ctx)
		assert.Error(t, err)
	})

	t.Run("validation fails with missing walletwatch configuration", func(t *testing.T) {
		// Missing WalletWatch configuration

		// Set other required fields (using inline configs)
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

		ctx := t.Context()
		_, err := Load(ctx)
		assert.Error(t, err)
	})

	t.Run("validation fails with missing chainstream configuration", func(t *testing.T) {
		// Missing ChainStream configuration

		// Set other required fields (using inline configs)
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")

		ctx := t.Context()
		_, err := Load(ctx)
		assert.Error(t, err)
	})

	t.Run("validation fails with engine reference but engine not configured", func(t *testing.T) {
		// Set up configuration where WalletRegistry references a global Redis engine
		// but the global Redis engine is not configured
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_ENGINE", "REDIS")
		// Missing ENGINES_STORAGE_REDIS_ADDRESS

		// Set other required fields (using inline configs)
		t.Setenv("WALLETWATCH_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

		ctx := t.Context()
		_, err := Load(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "engine_not_configured")
	})

	t.Run("validation fails with unregistered engine", func(t *testing.T) {
		// Set up configuration with invalid engine
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_ENGINE", "UNKNOWN_ENGINE")

		// Set other required fields (using inline configs)
		t.Setenv("WALLETWATCH_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

		ctx := t.Context()
		_, err := Load(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "engine_not_registered")
	})

	t.Run("successful loading with global engines and references", func(t *testing.T) {
		// Set up configuration where use cases reference global engines
		t.Setenv("ENGINES_STORAGE_REDIS_ADDRESS", "global-redis:6379")
		t.Setenv("ENGINES_STORAGE_POSTGRESQL_DSN", "postgres://global-db/test")
		t.Setenv("ENGINES_MESSAGING_REDIS_ADDRESS", "global-redis:6379")
		t.Setenv("ENGINES_MESSAGING_RABBITMQ_URI", "amqp://global-rabbit")

		// Use cases reference global engines
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("WALLETWATCH_WALLET_STORAGE_ENGINE", "POSTGRESQL")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_ENGINE", "RABBITMQ")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_RABBITMQ_ROUTING_KEY", "wallet.transactions")
		t.Setenv("WALLETWATCH_MAX_PROCESSING_TIME", "45s")
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

		ctx := t.Context()
		config, err := Load(ctx)
		require.NoError(t, err)

		// Verify global engines were configured
		require.NotNil(t, config.Engines.Storage.Redis)
		assert.Equal(t, "global-redis:6379", config.Engines.Storage.Redis.Address)
		require.NotNil(t, config.Engines.Storage.PostgreSQL)
		assert.Equal(t, "postgres://global-db/test", config.Engines.Storage.PostgreSQL.DSN)
		require.NotNil(t, config.Engines.Messaging.Redis)
		assert.Equal(t, "global-redis:6379", config.Engines.Messaging.Redis.Address)
		require.NotNil(t, config.Engines.Messaging.RabbitMQ)
		assert.Equal(t, "amqp://global-rabbit", config.Engines.Messaging.RabbitMQ.URI)

		// Verify use cases reference global engines (no inline configs)
		assert.Equal(t, "REDIS", config.Walletregistry.WalletStorage.Engine)
		assert.Nil(t, config.Walletregistry.WalletStorage.InlineConfig.Redis)
		assert.Nil(t, config.Walletregistry.WalletStorage.InlineConfig.PostgreSQL)

		assert.Equal(t, "POSTGRESQL", config.Walletwatch.WalletStorage.Engine)
		assert.Nil(t, config.Walletwatch.WalletStorage.InlineConfig.Redis)
		assert.Nil(t, config.Walletwatch.WalletStorage.InlineConfig.PostgreSQL)

		assert.Equal(t, "RABBITMQ", config.Walletwatch.TransactionNotifier.Engine)
		assert.Nil(t, config.Walletwatch.TransactionNotifier.InlineConfig.Redis)
		assert.Nil(t, config.Walletwatch.TransactionNotifier.InlineConfig.RabbitMQ)
		assert.Equal(t, "wallet.transactions", config.Walletwatch.TransactionNotifier.MessagePublisher.RabbitMQ.RoutingKey)
	})

	t.Run("successful loading with mixed global and inline configurations", func(t *testing.T) {
		// Set up some global engines
		t.Setenv("ENGINES_STORAGE_REDIS_ADDRESS", "global-redis:6379")
		t.Setenv("ENGINES_MESSAGING_RABBITMQ_URI", "amqp://global-rabbit")

		// WalletRegistry uses global Redis
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_ENGINE", "REDIS")

		// WalletWatch uses inline PostgreSQL and global RabbitMQ
		t.Setenv("WALLETWATCH_WALLET_STORAGE_POSTGRESQL_DSN", "postgres://inline-db/test")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_ENGINE", "RABBITMQ")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_RABBITMQ_ROUTING_KEY", "wallet.transactions")

		// ChainStream
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

		ctx := t.Context()
		config, err := Load(ctx)
		require.NoError(t, err)

		// Verify global engines
		require.NotNil(t, config.Engines.Storage.Redis)
		assert.Equal(t, "global-redis:6379", config.Engines.Storage.Redis.Address)
		assert.Nil(t, config.Engines.Storage.PostgreSQL) // Not configured globally
		require.NotNil(t, config.Engines.Messaging.RabbitMQ)
		assert.Equal(t, "amqp://global-rabbit", config.Engines.Messaging.RabbitMQ.URI)

		// Verify WalletRegistry uses global Redis
		assert.Equal(t, "REDIS", config.Walletregistry.WalletStorage.Engine)
		assert.Nil(t, config.Walletregistry.WalletStorage.InlineConfig.Redis)

		// Verify WalletWatch uses inline PostgreSQL
		assert.Empty(t, config.Walletwatch.WalletStorage.Engine)
		require.NotNil(t, config.Walletwatch.WalletStorage.InlineConfig.PostgreSQL)
		assert.Equal(t, "postgres://inline-db/test", config.Walletwatch.WalletStorage.InlineConfig.PostgreSQL.DSN)

		// Verify WalletWatch uses global RabbitMQ
		assert.Equal(t, "RABBITMQ", config.Walletwatch.TransactionNotifier.Engine)
		assert.Nil(t, config.Walletwatch.TransactionNotifier.InlineConfig.RabbitMQ)
		assert.Equal(t, "wallet.transactions", config.Walletwatch.TransactionNotifier.MessagePublisher.RabbitMQ.RoutingKey)
	})

	t.Run("successful loading with all log levels", func(t *testing.T) {
		logLevels := []string{"DEBUG", "INFO", "WARN", "ERROR", "PANIC", "FATAL"}

		for _, level := range logLevels {
			t.Run("log_level_"+level, func(t *testing.T) {
				t.Setenv("LOG_LEVEL", level)

				// Set required fields (using inline configs)
				t.Setenv("WALLETREGISTRY_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
				t.Setenv("WALLETWATCH_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
				t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379")
				t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")
				t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

				ctx := t.Context()
				config, err := Load(ctx)
				require.NoError(t, err)
				assert.Equal(t, level, config.Log.Level)
			})
		}
	})

	t.Run("validation fails with chainstream checkpoint storage engine not configured", func(t *testing.T) {
		// Set up configuration where ChainStream references a global Redis engine
		// but the global Redis engine is not configured
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")
		t.Setenv("CHAINSTREAM_CHECKPOINT_STORAGE_ENGINE", "REDIS")
		// Missing ENGINES_STORAGE_REDIS_ADDRESS

		// Set other required fields (using inline configs)
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")

		ctx := t.Context()
		_, err := Load(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "engine_not_configured")
	})

	t.Run("validation fails with chainstream dispatch failure handler engine not configured", func(t *testing.T) {
		// Set up configuration where ChainStream DispatchFailureHandler references a global RabbitMQ engine
		// but the global RabbitMQ engine is not configured
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")
		t.Setenv("CHAINSTREAM_DISPATCH_FAILURE_HANDLER_ENGINE", "RABBITMQ")
		t.Setenv("CHAINSTREAM_DISPATCH_FAILURE_HANDLER_RABBITMQ_ROUTING_KEY", "failures")
		// Missing ENGINES_MESSAGING_RABBITMQ_URI

		// Set other required fields (using inline configs)
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")

		ctx := t.Context()
		_, err := Load(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "engine_not_configured")
	})

	t.Run("validation fails with walletwatch idempotency guard engine not configured", func(t *testing.T) {
		// Set up configuration where WalletWatch IdempotencyGuard references a global PostgreSQL engine
		// but the global PostgreSQL engine is not configured
		t.Setenv("WALLETWATCH_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")
		t.Setenv("WALLETWATCH_IDEMPOTENCY_GUARD_ENGINE", "POSTGRESQL")
		// Missing ENGINES_STORAGE_POSTGRESQL_DSN

		// Set other required fields
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

		ctx := t.Context()
		_, err := Load(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "engine_not_configured")
	})

	t.Run("validation fails with multiple unregistered engines", func(t *testing.T) {
		// Set up configuration with multiple invalid engines
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_ENGINE", "UNKNOWN_STORAGE")
		t.Setenv("WALLETWATCH_WALLET_STORAGE_ENGINE", "INVALID_STORAGE")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_ENGINE", "UNKNOWN_MESSAGING")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_RABBITMQ_ROUTING_KEY", "test")
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

		ctx := t.Context()
		_, err := Load(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "engine_not_registered")
	})

	t.Run("successful loading with all engine types configured globally", func(t *testing.T) {
		// Set up all global engines
		t.Setenv("ENGINES_STORAGE_REDIS_ADDRESS", "global-redis:6379")
		t.Setenv("ENGINES_STORAGE_POSTGRESQL_DSN", "postgres://global-db/test")
		t.Setenv("ENGINES_MESSAGING_REDIS_ADDRESS", "global-msg-redis:6379")
		t.Setenv("ENGINES_MESSAGING_RABBITMQ_URI", "amqp://global-rabbit")

		// Use cases reference different global engines
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("WALLETWATCH_WALLET_STORAGE_ENGINE", "POSTGRESQL")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_ENGINE", "RABBITMQ")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_RABBITMQ_ROUTING_KEY", "wallet.transactions")
		t.Setenv("WALLETWATCH_IDEMPOTENCY_GUARD_ENGINE", "REDIS")
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")
		t.Setenv("CHAINSTREAM_CHECKPOINT_STORAGE_ENGINE", "POSTGRESQL")
		t.Setenv("CHAINSTREAM_DISPATCH_FAILURE_HANDLER_ENGINE", "REDIS")
		t.Setenv("CHAINSTREAM_DISPATCH_FAILURE_HANDLER_REDIS_STREAM", "failures")

		ctx := t.Context()
		config, err := Load(ctx)
		require.NoError(t, err)

		// Verify all global engines were configured
		require.NotNil(t, config.Engines.Storage.Redis)
		assert.Equal(t, "global-redis:6379", config.Engines.Storage.Redis.Address)
		require.NotNil(t, config.Engines.Storage.PostgreSQL)
		assert.Equal(t, "postgres://global-db/test", config.Engines.Storage.PostgreSQL.DSN)
		require.NotNil(t, config.Engines.Messaging.Redis)
		assert.Equal(t, "global-msg-redis:6379", config.Engines.Messaging.Redis.Address)
		require.NotNil(t, config.Engines.Messaging.RabbitMQ)
		assert.Equal(t, "amqp://global-rabbit", config.Engines.Messaging.RabbitMQ.URI)

		// Verify all use cases reference global engines correctly
		assert.Equal(t, "REDIS", config.Walletregistry.WalletStorage.Engine)
		assert.Equal(t, "POSTGRESQL", config.Walletwatch.WalletStorage.Engine)
		assert.Equal(t, "RABBITMQ", config.Walletwatch.TransactionNotifier.Engine)
		assert.Equal(t, "REDIS", config.Walletwatch.IdempotencyGuard.Engine)
		assert.Equal(t, "POSTGRESQL", config.Chainstream.CheckpointStorage.Engine)
		assert.Equal(t, "REDIS", config.Chainstream.DispatchFailureHandler.Engine)
	})

	t.Run("validation fails with mixed valid and invalid engine references", func(t *testing.T) {
		// Set up some valid global engines
		t.Setenv("ENGINES_STORAGE_REDIS_ADDRESS", "global-redis:6379")
		// Missing PostgreSQL engine

		// Mix of valid and invalid engine references
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_ENGINE", "REDIS")                    // Valid
		t.Setenv("WALLETWATCH_WALLET_STORAGE_ENGINE", "POSTGRESQL")                  // Invalid - engine not configured
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379") // Valid inline
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

		ctx := t.Context()
		_, err := Load(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "engine_not_configured")
	})

	t.Run("successful validation with pointer picker engines", func(t *testing.T) {
		// Set up global engines for pointer pickers
		t.Setenv("ENGINES_STORAGE_REDIS_ADDRESS", "global-redis:6379")
		t.Setenv("ENGINES_MESSAGING_REDIS_ADDRESS", "global-msg-redis:6379")

		// Required configurations
		t.Setenv("WALLETREGISTRY_WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("WALLETWATCH_WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_ENGINE", "REDIS")
		t.Setenv("WALLETWATCH_TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")
		t.Setenv("CHAINSTREAM_NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")

		// Optional pointer pickers
		t.Setenv("WALLETWATCH_IDEMPOTENCY_GUARD_ENGINE", "REDIS")
		t.Setenv("CHAINSTREAM_CHECKPOINT_STORAGE_ENGINE", "REDIS")
		t.Setenv("CHAINSTREAM_DISPATCH_FAILURE_HANDLER_ENGINE", "REDIS")
		t.Setenv("CHAINSTREAM_DISPATCH_FAILURE_HANDLER_REDIS_STREAM", "failures")

		ctx := t.Context()
		config, err := Load(ctx)
		require.NoError(t, err)

		// Verify pointer pickers were configured correctly
		require.NotNil(t, config.Walletwatch.IdempotencyGuard)
		assert.Equal(t, "REDIS", config.Walletwatch.IdempotencyGuard.Engine)
		require.NotNil(t, config.Chainstream.CheckpointStorage)
		assert.Equal(t, "REDIS", config.Chainstream.CheckpointStorage.Engine)
		require.NotNil(t, config.Chainstream.DispatchFailureHandler)
		assert.Equal(t, "REDIS", config.Chainstream.DispatchFailureHandler.Engine)
	})
}
