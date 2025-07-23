package messaging

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	rabbitmqmocks "github.com/gabapcia/blockwatch/internal/infra/messaging/rabbitmq/mocks"
	redismocks "github.com/gabapcia/blockwatch/internal/infra/messaging/redis/mocks"
	messagingconfig "github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	"github.com/gabapcia/blockwatch/internal/walletwatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	t.Run("resolves Redis TransactionNotifier using default engine", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		mockTransactionNotifier := redismocks.NewWalletwatchTransactionNotifier(t)

		mockRedisClient.EXPECT().AsWalletwatchTransactionNotifier("test-stream").
			Return(mockTransactionNotifier).Once()

		defaults = map[string]any{
			"REDIS": mockRedisClient,
		}

		picker := messagingconfig.Picker{
			Engine: "redis",
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}

		// Execute
		notifier, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockTransactionNotifier, notifier)
	})

	t.Run("resolves RabbitMQ TransactionNotifier using default engine", func(t *testing.T) {
		// Setup
		mockRabbitMQClient := rabbitmqmocks.NewClient(t)
		mockTransactionNotifier := rabbitmqmocks.NewWalletwatchTransactionNotifier(t)

		mockRabbitMQClient.EXPECT().AsWalletwatchTransactionNotifier("test-exchange", "test-routing-key").
			Return(mockTransactionNotifier).Once()

		defaults = map[string]any{
			"RABBITMQ": mockRabbitMQClient,
		}

		picker := messagingconfig.Picker{
			Engine: "rabbitmq",
			MessagePublisher: messagingconfig.MessagePublisher{
				RabbitMQ: &messagingconfig.RabbitMQPublisher{
					Exchange:   "test-exchange",
					RoutingKey: "test-routing-key",
				},
			},
		}

		// Execute
		notifier, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockTransactionNotifier, notifier)
	})

	t.Run("resolves Redis DispatchFailureNotifier using default engine", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		mockDispatchFailureNotifier := redismocks.NewChainstreamDispatchFailureNotifier(t)

		mockRedisClient.EXPECT().AsChainstreamDispatchFailureNotifier("failure-stream").
			Return(mockDispatchFailureNotifier).Once()

		defaults = map[string]any{
			"REDIS": mockRedisClient,
		}

		picker := messagingconfig.Picker{
			Engine: "redis",
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "failure-stream",
				},
			},
		}

		// Execute
		notifier, err := Resolve[chainstream.DispatchFailureNotifier](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockDispatchFailureNotifier, notifier)
	})

	t.Run("resolves RabbitMQ DispatchFailureNotifier using default engine", func(t *testing.T) {
		// Setup
		mockRabbitMQClient := rabbitmqmocks.NewClient(t)
		mockDispatchFailureNotifier := rabbitmqmocks.NewChainstreamDispatchFailureNotifier(t)

		mockRabbitMQClient.EXPECT().AsChainstreamDispatchFailureNotifier("failure-exchange", "failure-routing-key").
			Return(mockDispatchFailureNotifier).Once()

		defaults = map[string]any{
			"RABBITMQ": mockRabbitMQClient,
		}

		picker := messagingconfig.Picker{
			Engine: "rabbitmq",
			MessagePublisher: messagingconfig.MessagePublisher{
				RabbitMQ: &messagingconfig.RabbitMQPublisher{
					Exchange:   "failure-exchange",
					RoutingKey: "failure-routing-key",
				},
			},
		}

		// Execute
		notifier, err := Resolve[chainstream.DispatchFailureNotifier](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockDispatchFailureNotifier, notifier)
	})

	t.Run("resolves using inline Redis configuration", func(t *testing.T) {
		// Setup - clear defaults to force inline config usage
		defaults = map[string]any{}

		// Mock the factory to return our mock client
		mockRedisClient := redismocks.NewClient(t)
		mockTransactionNotifier := redismocks.NewWalletwatchTransactionNotifier(t)

		mockRedisClient.EXPECT().AsWalletwatchTransactionNotifier("inline-stream").
			Return(mockTransactionNotifier).Once()

		originalFactory := messagingFactories["REDIS"]
		messagingFactories["REDIS"] = messagingFactory{
			BuildConnection: func(ctx context.Context, config any) (any, error) {
				redisCfg := config.(messagingconfig.RedisConnection)
				assert.Equal(t, "localhost:6379", redisCfg.Address)
				assert.Equal(t, "user", redisCfg.Username)
				assert.Equal(t, "pass", redisCfg.Password)
				assert.Equal(t, 1, redisCfg.DB)
				return mockRedisClient, nil
			},
			InterfaceAdapters: originalFactory.InterfaceAdapters,
		}
		defer func() {
			messagingFactories["REDIS"] = originalFactory
		}()

		picker := messagingconfig.Picker{
			InlineConfig: messagingconfig.InlineConfig{
				Redis: &messagingconfig.RedisConnection{
					Address:  "localhost:6379",
					Username: "user",
					Password: "pass",
					DB:       1,
				},
			},
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "inline-stream",
				},
			},
		}

		// Execute
		notifier, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockTransactionNotifier, notifier)
	})

	t.Run("resolves using inline RabbitMQ configuration", func(t *testing.T) {
		// Setup - clear defaults to force inline config usage
		defaults = map[string]any{}

		// Mock the factory to return our mock client
		mockRabbitMQClient := rabbitmqmocks.NewClient(t)
		mockTransactionNotifier := rabbitmqmocks.NewWalletwatchTransactionNotifier(t)

		mockRabbitMQClient.EXPECT().AsWalletwatchTransactionNotifier("inline-exchange", "inline-routing-key").
			Return(mockTransactionNotifier).Once()

		originalFactory := messagingFactories["RABBITMQ"]
		messagingFactories["RABBITMQ"] = messagingFactory{
			BuildConnection: func(ctx context.Context, config any) (any, error) {
				rabbitCfg := config.(messagingconfig.RabbitMQConnection)
				assert.Equal(t, "amqp://localhost:5672", rabbitCfg.URI)
				return mockRabbitMQClient, nil
			},
			InterfaceAdapters: originalFactory.InterfaceAdapters,
		}
		defer func() {
			messagingFactories["RABBITMQ"] = originalFactory
		}()

		picker := messagingconfig.Picker{
			InlineConfig: messagingconfig.InlineConfig{
				RabbitMQ: &messagingconfig.RabbitMQConnection{
					URI: "amqp://localhost:5672",
				},
			},
			MessagePublisher: messagingconfig.MessagePublisher{
				RabbitMQ: &messagingconfig.RabbitMQPublisher{
					Exchange:   "inline-exchange",
					RoutingKey: "inline-routing-key",
				},
			},
		}

		// Execute
		notifier, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockTransactionNotifier, notifier)
	})

	t.Run("tracks closeable connections from inline config", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}
		openedConnections = []io.Closer{} // Reset

		mockRedisClient := redismocks.NewClient(t)
		mockTransactionNotifier := redismocks.NewWalletwatchTransactionNotifier(t)

		mockRedisClient.EXPECT().AsWalletwatchTransactionNotifier("test-stream").
			Return(mockTransactionNotifier).Once()

		originalFactory := messagingFactories["REDIS"]
		messagingFactories["REDIS"] = messagingFactory{
			BuildConnection: func(ctx context.Context, config any) (any, error) {
				return mockRedisClient, nil
			},
			InterfaceAdapters: originalFactory.InterfaceAdapters,
		}
		defer func() {
			messagingFactories["REDIS"] = originalFactory
		}()

		picker := messagingconfig.Picker{
			InlineConfig: messagingconfig.InlineConfig{
				Redis: &messagingconfig.RedisConnection{
					Address: "localhost:6379",
				},
			},
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}

		// Execute
		_, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Len(t, openedConnections, 1)
		assert.Equal(t, mockRedisClient, openedConnections[0])
	})

	t.Run("returns error when default engine not found", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		picker := messagingconfig.Picker{
			Engine: "redis",
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}

		// Execute
		notifier, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no default messaging instance found for engine \"REDIS\"")
		assert.Nil(t, notifier)
	})

	t.Run("returns error when publisher config not found for default engine", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		defaults = map[string]any{
			"REDIS": mockRedisClient,
		}

		picker := messagingconfig.Picker{
			Engine: "redis",
			MessagePublisher: messagingconfig.MessagePublisher{
				RabbitMQ: &messagingconfig.RabbitMQPublisher{
					Exchange:   "wrong-exchange",
					RoutingKey: "wrong-key",
				},
			},
		}

		// Execute
		notifier, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no publisher configuration found for engine \"REDIS\"")
		assert.Nil(t, notifier)
	})

	t.Run("returns error when no factory registered for inline engine", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		picker := messagingconfig.Picker{
			InlineConfig: messagingconfig.InlineConfig{
				Redis: &messagingconfig.RedisConnection{
					Address: "localhost:6379",
				},
			},
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}

		// Temporarily remove the factory
		originalFactory := messagingFactories["REDIS"]
		delete(messagingFactories, "REDIS")
		defer func() {
			messagingFactories["REDIS"] = originalFactory
		}()

		// Execute
		notifier, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no messaging factory registered for engine \"REDIS\"")
		assert.Nil(t, notifier)
	})

	t.Run("returns error when inline connection creation fails", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		connectionError := errors.New("connection failed")
		originalFactory := messagingFactories["REDIS"]
		messagingFactories["REDIS"] = messagingFactory{
			BuildConnection: func(ctx context.Context, config any) (any, error) {
				return nil, connectionError
			},
			InterfaceAdapters: originalFactory.InterfaceAdapters,
		}
		defer func() {
			messagingFactories["REDIS"] = originalFactory
		}()

		picker := messagingconfig.Picker{
			InlineConfig: messagingconfig.InlineConfig{
				Redis: &messagingconfig.RedisConnection{
					Address: "localhost:6379",
				},
			},
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}

		// Execute
		notifier, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create inline messaging instance for engine \"REDIS\"")
		assert.Contains(t, err.Error(), "connection failed")
		assert.Nil(t, notifier)
	})

	t.Run("returns error when publisher config not found for inline engine", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		mockRedisClient := redismocks.NewClient(t)
		originalFactory := messagingFactories["REDIS"]
		messagingFactories["REDIS"] = messagingFactory{
			BuildConnection: func(ctx context.Context, config any) (any, error) {
				return mockRedisClient, nil
			},
			InterfaceAdapters: originalFactory.InterfaceAdapters,
		}
		defer func() {
			messagingFactories["REDIS"] = originalFactory
		}()

		picker := messagingconfig.Picker{
			InlineConfig: messagingconfig.InlineConfig{
				Redis: &messagingconfig.RedisConnection{
					Address: "localhost:6379",
				},
			},
			MessagePublisher: messagingconfig.MessagePublisher{
				RabbitMQ: &messagingconfig.RabbitMQPublisher{
					Exchange:   "wrong-exchange",
					RoutingKey: "wrong-key",
				},
			},
		}

		// Execute
		notifier, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no publisher configuration found for engine \"REDIS\"")
		assert.Nil(t, notifier)
	})

	t.Run("returns error when no adapter registered for interface type", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		defaults = map[string]any{
			"REDIS": mockRedisClient,
		}

		// Temporarily remove the adapter for TransactionNotifier
		originalFactory := messagingFactories["REDIS"]
		modifiedFactory := messagingFactory{
			BuildConnection:   originalFactory.BuildConnection,
			InterfaceAdapters: make(map[reflect.Type]func(conn, pubCfg any) any),
		}
		// Only keep the DispatchFailureNotifier adapter
		modifiedFactory.InterfaceAdapters[typeOf[chainstream.DispatchFailureNotifier]()] = originalFactory.InterfaceAdapters[typeOf[chainstream.DispatchFailureNotifier]()]
		messagingFactories["REDIS"] = modifiedFactory
		defer func() {
			messagingFactories["REDIS"] = originalFactory
		}()

		picker := messagingconfig.Picker{
			Engine: "redis",
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}

		// Execute
		notifier, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no adapter registered for type")
		assert.Contains(t, err.Error(), "TransactionNotifier")
		assert.Contains(t, err.Error(), "in engine \"REDIS\"")
		assert.Nil(t, notifier)
	})

	t.Run("returns error when no valid messaging engine configuration provided", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		picker := messagingconfig.Picker{
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}

		// Execute
		notifier, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no valid messaging engine configuration provided")
		assert.Nil(t, notifier)
	})

	t.Run("handles case-insensitive engine names", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		mockTransactionNotifier := redismocks.NewWalletwatchTransactionNotifier(t)

		mockRedisClient.EXPECT().AsWalletwatchTransactionNotifier("test-stream").
			Return(mockTransactionNotifier).Once()

		defaults = map[string]any{
			"REDIS": mockRedisClient,
		}

		picker := messagingconfig.Picker{
			Engine: "Redis", // Mixed case
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}

		// Execute
		notifier, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockTransactionNotifier, notifier)
	})

	t.Run("handles empty engine name", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		mockRedisClient := redismocks.NewClient(t)
		mockTransactionNotifier := redismocks.NewWalletwatchTransactionNotifier(t)

		mockRedisClient.EXPECT().AsWalletwatchTransactionNotifier("test-stream").
			Return(mockTransactionNotifier).Once()

		picker := messagingconfig.Picker{
			Engine: "", // Empty engine name should trigger inline config path
			InlineConfig: messagingconfig.InlineConfig{
				Redis: &messagingconfig.RedisConnection{
					Address: "localhost:6379",
				},
			},
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}

		// Mock the factory
		originalFactory := messagingFactories["REDIS"]
		messagingFactories["REDIS"] = messagingFactory{
			BuildConnection: func(ctx context.Context, config any) (any, error) {
				return mockRedisClient, nil
			},
			InterfaceAdapters: originalFactory.InterfaceAdapters,
		}
		defer func() {
			messagingFactories["REDIS"] = originalFactory
		}()

		// Execute
		notifier, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockTransactionNotifier, notifier)
	})

	t.Run("returns error when adapter returns wrong type", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		defaults = map[string]any{
			"REDIS": mockRedisClient,
		}

		// Create a modified factory that returns the wrong type
		originalFactory := messagingFactories["REDIS"]
		modifiedFactory := messagingFactory{
			BuildConnection: originalFactory.BuildConnection,
			InterfaceAdapters: map[reflect.Type]func(conn, pubCfg any) any{
				typeOf[walletwatch.TransactionNotifier](): func(conn, pubCfg any) any {
					// Return wrong type (string instead of TransactionNotifier)
					return "wrong type"
				},
			},
		}
		messagingFactories["REDIS"] = modifiedFactory
		defer func() {
			messagingFactories["REDIS"] = originalFactory
		}()

		picker := messagingconfig.Picker{
			Engine: "redis",
			MessagePublisher: messagingconfig.MessagePublisher{
				Redis: &messagingconfig.RedisPublisher{
					Stream: "test-stream",
				},
			},
		}

		// Execute
		notifier, err := Resolve[walletwatch.TransactionNotifier](t.Context(), picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "adapter for engine \"REDIS\" returned wrong type")
		assert.Contains(t, err.Error(), "expected")
		assert.Nil(t, notifier)
	})
}

func TestAdaptMessaging(t *testing.T) {
	t.Run("returns error when called with unknown engine", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		pubCfg := messagingconfig.RedisPublisher{
			Stream: "test-stream",
		}

		// Execute - call adaptMessaging directly with unknown engine
		notifier, err := adaptMessaging[walletwatch.TransactionNotifier](mockRedisClient, pubCfg, "UNKNOWN_ENGINE")

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no factory registered for engine \"UNKNOWN_ENGINE\"")
		assert.Nil(t, notifier)
	})
}
