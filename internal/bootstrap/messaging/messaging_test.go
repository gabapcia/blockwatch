package messaging

import (
	"context"
	"errors"
	"io"
	"testing"

	messagingconfig "github.com/gabapcia/blockwatch/internal/pkg/config/messaging"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rabbitmqcontainer "github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

func setupRedisContainer(t *testing.T) messagingconfig.RedisConnection {
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

	return messagingconfig.RedisConnection{
		Address:  opts.Addr,
		Username: opts.Username,
		Password: opts.Password,
		DB:       opts.DB,
	}
}

func setupRabbitMQContainer(t *testing.T) messagingconfig.RabbitMQConnection {
	t.Helper()

	ctx := t.Context()

	// Start RabbitMQ container
	rabbitmqContainer, err := rabbitmqcontainer.Run(ctx, "rabbitmq:4-management")
	require.NoError(t, err)

	t.Cleanup(func() {
		rabbitmqContainer.Terminate(ctx)
	})

	// Get connection details
	amqpURL, err := rabbitmqContainer.AmqpURL(ctx)
	require.NoError(t, err)

	return messagingconfig.RabbitMQConnection{
		URI: amqpURL,
	}
}

func TestInit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		redisCfg := setupRedisContainer(t)
		err := Init(t.Context(), messagingconfig.Engines{
			Redis: &redisCfg,
		})
		assert.NoError(t, err)
		assert.NotNil(t, defaults["REDIS"])
		assert.Len(t, openedConnections, 1)

		err = Close()
		assert.NoError(t, err)
	})

	t.Run("unsupported engine", func(t *testing.T) {
		originalFactories := messagingFactories
		messagingFactories = make(map[string]messagingFactory)
		t.Cleanup(func() {
			messagingFactories = originalFactories
		})

		err := Init(t.Context(), messagingconfig.Engines{
			RabbitMQ: &messagingconfig.RabbitMQConnection{},
		})
		assert.Error(t, err)
	})

	t.Run("factory error", func(t *testing.T) {
		err := Init(t.Context(), messagingconfig.Engines{
			Redis: &messagingconfig.RedisConnection{},
		})
		assert.Error(t, err)
	})

	t.Run("factory error with specific message", func(t *testing.T) {
		originalRedisFactory := messagingFactories["REDIS"]
		messagingFactories["REDIS"] = messagingFactory{
			BuildConnection: func(ctx context.Context, config any) (any, error) {
				return nil, errors.New("factory failed")
			},
		}
		t.Cleanup(func() {
			messagingFactories["REDIS"] = originalRedisFactory
		})

		err := Init(t.Context(), messagingconfig.Engines{
			Redis: &messagingconfig.RedisConnection{},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize messaging engine \"REDIS\"")
	})
}

type mockCloser struct {
	shouldFail bool
}

func (m mockCloser) Close() error {
	if m.shouldFail {
		return errors.New("failed to close")
	}
	return nil
}

func TestClose(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		openedConnections = []io.Closer{
			mockCloser{shouldFail: false},
			mockCloser{shouldFail: false},
		}

		err := Close()
		assert.NoError(t, err)
	})

	t.Run("with error", func(t *testing.T) {
		openedConnections = []io.Closer{
			mockCloser{shouldFail: false},
			mockCloser{shouldFail: true},
		}

		err := Close()
		assert.Error(t, err)
	})
}
