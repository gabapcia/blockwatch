package config

import (
	"reflect"
	"testing"

	"github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	"github.com/gabapcia/blockwatch/internal/pkg/config/storage"
	validatorMocks "github.com/gabapcia/blockwatch/internal/pkg/validator/mocks"

	"github.com/stretchr/testify/mock"
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
