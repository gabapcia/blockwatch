package messaging

import (
	"context"
	"errors"
	"io"
	"testing"

	rabbitmqmocks "github.com/gabapcia/blockwatch/internal/infra/messaging/rabbitmq/mocks"
	redismocks "github.com/gabapcia/blockwatch/internal/infra/messaging/redis/mocks"
	messagingconfig "github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	t.Run("initializes Redis engine successfully", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)

		originalFactory := messagingFactories["REDIS"]
		messagingFactories["REDIS"] = messagingFactory{
			BuildConnection: func(ctx context.Context, config any) (any, error) {
				redisCfg := config.(messagingconfig.RedisConnection)
				assert.Equal(t, "localhost:6379", redisCfg.Address)
				assert.Equal(t, "testuser", redisCfg.Username)
				assert.Equal(t, "testpass", redisCfg.Password)
				assert.Equal(t, 2, redisCfg.DB)
				return mockRedisClient, nil
			},
			InterfaceAdapters: originalFactory.InterfaceAdapters,
		}
		defer func() {
			messagingFactories["REDIS"] = originalFactory
		}()

		engines := messagingconfig.Engines{
			Redis: &messagingconfig.RedisConnection{
				Address:  "localhost:6379",
				Username: "testuser",
				Password: "testpass",
				DB:       2,
			},
		}

		// Execute
		err := Init(t.Context(), engines)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, defaults, "REDIS")
		assert.Equal(t, mockRedisClient, defaults["REDIS"])
		assert.Contains(t, openedConnections, mockRedisClient)
	})

	t.Run("initializes RabbitMQ engine successfully", func(t *testing.T) {
		// Setup
		mockRabbitMQClient := rabbitmqmocks.NewClient(t)

		originalFactory := messagingFactories["RABBITMQ"]
		messagingFactories["RABBITMQ"] = messagingFactory{
			BuildConnection: func(ctx context.Context, config any) (any, error) {
				rabbitCfg := config.(messagingconfig.RabbitMQConnection)
				assert.Equal(t, "amqp://test:test@localhost:5672", rabbitCfg.URI)
				return mockRabbitMQClient, nil
			},
			InterfaceAdapters: originalFactory.InterfaceAdapters,
		}
		defer func() {
			messagingFactories["RABBITMQ"] = originalFactory
		}()

		engines := messagingconfig.Engines{
			RabbitMQ: &messagingconfig.RabbitMQConnection{
				URI: "amqp://test:test@localhost:5672",
			},
		}

		// Execute
		err := Init(t.Context(), engines)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, defaults, "RABBITMQ")
		assert.Equal(t, mockRabbitMQClient, defaults["RABBITMQ"])
		assert.Contains(t, openedConnections, mockRabbitMQClient)
	})

	t.Run("initializes multiple engines successfully", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		mockRabbitMQClient := rabbitmqmocks.NewClient(t)

		originalRedisFactory := messagingFactories["REDIS"]
		originalRabbitFactory := messagingFactories["RABBITMQ"]

		messagingFactories["REDIS"] = messagingFactory{
			BuildConnection: func(ctx context.Context, config any) (any, error) {
				return mockRedisClient, nil
			},
			InterfaceAdapters: originalRedisFactory.InterfaceAdapters,
		}

		messagingFactories["RABBITMQ"] = messagingFactory{
			BuildConnection: func(ctx context.Context, config any) (any, error) {
				return mockRabbitMQClient, nil
			},
			InterfaceAdapters: originalRabbitFactory.InterfaceAdapters,
		}

		defer func() {
			messagingFactories["REDIS"] = originalRedisFactory
			messagingFactories["RABBITMQ"] = originalRabbitFactory
		}()

		engines := messagingconfig.Engines{
			Redis: &messagingconfig.RedisConnection{
				Address: "localhost:6379",
			},
			RabbitMQ: &messagingconfig.RabbitMQConnection{
				URI: "amqp://localhost:5672",
			},
		}

		// Execute
		err := Init(t.Context(), engines)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, defaults, "REDIS")
		assert.Contains(t, defaults, "RABBITMQ")
		assert.Equal(t, mockRedisClient, defaults["REDIS"])
		assert.Equal(t, mockRabbitMQClient, defaults["RABBITMQ"])
		assert.Len(t, openedConnections, 2)
		assert.Contains(t, openedConnections, mockRedisClient)
		assert.Contains(t, openedConnections, mockRabbitMQClient)
	})

	t.Run("skips nil engine configurations", func(t *testing.T) {
		// Setup
		engines := messagingconfig.Engines{
			Redis:    nil, // This should be skipped
			RabbitMQ: nil, // This should be skipped
		}

		// Execute
		err := Init(t.Context(), engines)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, defaults)
		assert.Empty(t, openedConnections)
	})

	t.Run("returns error when no factory registered for engine", func(t *testing.T) {
		// Setup
		engines := messagingconfig.Engines{
			Redis: &messagingconfig.RedisConnection{
				Address: "localhost:6379",
			},
		}

		// Temporarily remove the factory
		originalFactory := messagingFactories["REDIS"]
		delete(messagingFactories, "REDIS")
		defer func() {
			messagingFactories["REDIS"] = originalFactory
		}()

		// Execute
		err := Init(t.Context(), engines)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no messaging factory registered for engine \"REDIS\"")
	})

	t.Run("returns error when connection creation fails", func(t *testing.T) {
		// Setup
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

		engines := messagingconfig.Engines{
			Redis: &messagingconfig.RedisConnection{
				Address: "localhost:6379",
			},
		}

		// Execute
		err := Init(t.Context(), engines)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize messaging engine \"REDIS\"")
		assert.Contains(t, err.Error(), "connection failed")
	})

	t.Run("handles engines that don't implement io.Closer", func(t *testing.T) {
		// Setup - create a mock that doesn't implement io.Closer
		type NonCloserClient struct{}
		mockNonCloserClient := &NonCloserClient{}

		originalFactory := messagingFactories["REDIS"]
		messagingFactories["REDIS"] = messagingFactory{
			BuildConnection: func(ctx context.Context, config any) (any, error) {
				return mockNonCloserClient, nil
			},
			InterfaceAdapters: originalFactory.InterfaceAdapters,
		}
		defer func() {
			messagingFactories["REDIS"] = originalFactory
		}()

		engines := messagingconfig.Engines{
			Redis: &messagingconfig.RedisConnection{
				Address: "localhost:6379",
			},
		}

		// Execute
		err := Init(t.Context(), engines)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, defaults, "REDIS")
		assert.Equal(t, mockNonCloserClient, defaults["REDIS"])
		// Should not be added to openedConnections since it doesn't implement io.Closer
		assert.Empty(t, openedConnections)
	})

	t.Run("resets defaults and openedConnections on each call", func(t *testing.T) {
		// Setup - pre-populate with some data
		defaults = map[string]any{"OLD": "data"}
		openedConnections = []io.Closer{redismocks.NewClient(t)}

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

		engines := messagingconfig.Engines{
			Redis: &messagingconfig.RedisConnection{
				Address: "localhost:6379",
			},
		}

		// Execute
		err := Init(t.Context(), engines)

		// Assert
		require.NoError(t, err)
		// Should only contain the new Redis connection, not the old data
		assert.Len(t, defaults, 1)
		assert.Contains(t, defaults, "REDIS")
		assert.NotContains(t, defaults, "OLD")
		assert.Len(t, openedConnections, 1)
		assert.Contains(t, openedConnections, mockRedisClient)
	})
}

func TestClose(t *testing.T) {
	t.Run("closes all connections successfully", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		mockRabbitMQClient := rabbitmqmocks.NewClient(t)

		mockRedisClient.EXPECT().Close().Return(nil).Once()
		mockRabbitMQClient.EXPECT().Close().Return(nil).Once()

		openedConnections = []io.Closer{mockRedisClient, mockRabbitMQClient}

		// Execute
		err := Close()

		// Assert
		require.NoError(t, err)
	})

	t.Run("returns error when some connections fail to close", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		mockRabbitMQClient := rabbitmqmocks.NewClient(t)

		redisError := errors.New("redis close failed")
		rabbitError := errors.New("rabbit close failed")

		mockRedisClient.EXPECT().Close().Return(redisError).Once()
		mockRabbitMQClient.EXPECT().Close().Return(rabbitError).Once()

		openedConnections = []io.Closer{mockRedisClient, mockRabbitMQClient}

		// Execute
		err := Close()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis close failed")
		assert.Contains(t, err.Error(), "rabbit close failed")
	})

	t.Run("handles mixed success and failure", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		mockRabbitMQClient := rabbitmqmocks.NewClient(t)

		rabbitError := errors.New("rabbit close failed")

		mockRedisClient.EXPECT().Close().Return(nil).Once()
		mockRabbitMQClient.EXPECT().Close().Return(rabbitError).Once()

		openedConnections = []io.Closer{mockRedisClient, mockRabbitMQClient}

		// Execute
		err := Close()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rabbit close failed")
		assert.NotContains(t, err.Error(), "redis")
	})

	t.Run("handles empty connections list", func(t *testing.T) {
		// Setup
		openedConnections = []io.Closer{}

		// Execute
		err := Close()

		// Assert
		require.NoError(t, err)
	})

	t.Run("handles single connection", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		mockRedisClient.EXPECT().Close().Return(nil).Once()

		openedConnections = []io.Closer{mockRedisClient}

		// Execute
		err := Close()

		// Assert
		require.NoError(t, err)
	})

	t.Run("handles single connection with error", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		closeError := errors.New("close failed")
		mockRedisClient.EXPECT().Close().Return(closeError).Once()

		openedConnections = []io.Closer{mockRedisClient}

		// Execute
		err := Close()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "close failed")
	})
}
