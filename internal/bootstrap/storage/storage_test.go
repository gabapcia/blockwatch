package storage

import (
	"context"
	"errors"
	"io"
	"testing"

	storageconfig "github.com/gabapcia/blockwatch/internal/pkg/config/storage"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
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

func setupPostgresContainer(t *testing.T) storageconfig.PostgreSQL {
	t.Helper()

	ctx := t.Context()

	// Start PostgreSQL container
	postgresContainer, err := postgrescontainer.Run(ctx,
		"postgres:17-alpine",
		postgrescontainer.WithDatabase("test-db"),
		postgrescontainer.WithUsername("user"),
		postgrescontainer.WithPassword("password"),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		postgresContainer.Terminate(ctx)
	})

	// Get connection details
	dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	return storageconfig.PostgreSQL{
		DSN: dsn,
	}
}

func TestInit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		redisCfg := setupRedisContainer(t)
		err := Init(t.Context(), storageconfig.Engines{
			Redis: &redisCfg,
		})
		assert.NoError(t, err)
		assert.NotNil(t, defaults["REDIS"])
		assert.Len(t, openedConnections, 1)

		err = Close()
		assert.NoError(t, err)
	})

	t.Run("unsupported engine", func(t *testing.T) {
		originalFactories := storageFactories
		storageFactories = make(map[string]storageFactory)
		t.Cleanup(func() {
			storageFactories = originalFactories
		})

		err := Init(t.Context(), storageconfig.Engines{
			PostgreSQL: &storageconfig.PostgreSQL{},
		})
		assert.Error(t, err)
	})

	t.Run("factory error", func(t *testing.T) {
		err := Init(t.Context(), storageconfig.Engines{
			Redis: &storageconfig.Redis{},
		})
		assert.Error(t, err)
	})

	t.Run("factory error with specific message", func(t *testing.T) {
		originalRedisFactory := storageFactories["REDIS"]
		storageFactories["REDIS"] = func(ctx context.Context, config any) (any, error) {
			return nil, errors.New("factory failed")
		}
		t.Cleanup(func() {
			storageFactories["REDIS"] = originalRedisFactory
		})

		err := Init(t.Context(), storageconfig.Engines{
			Redis: &storageconfig.Redis{},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize storage engine \"REDIS\"")
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
