package storage

import (
	"context"
	"errors"
	"io"
	"testing"

	postgresqlmocks "github.com/gabapcia/blockwatch/internal/infra/storage/postgresql/mocks"
	redismocks "github.com/gabapcia/blockwatch/internal/infra/storage/redis/mocks"
	storageconfig "github.com/gabapcia/blockwatch/internal/pkg/config/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve(t *testing.T) {
	t.Run("resolves Redis client using default engine", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)

		defaults = map[string]any{
			"REDIS": mockRedisClient,
		}

		picker := storageconfig.Picker{
			Engine: "redis",
		}

		// Execute
		client, err := Resolve[*redismocks.Client](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockRedisClient, client)
	})

	t.Run("resolves PostgreSQL client using default engine", func(t *testing.T) {
		// Setup
		mockPostgreSQLClient := postgresqlmocks.NewClient(t)

		defaults = map[string]any{
			"POSTGRESQL": mockPostgreSQLClient,
		}

		picker := storageconfig.Picker{
			Engine: "postgresql",
		}

		// Execute
		client, err := Resolve[*postgresqlmocks.Client](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockPostgreSQLClient, client)
	})

	t.Run("resolves using inline Redis configuration", func(t *testing.T) {
		// Setup - clear defaults to force inline config usage
		defaults = map[string]any{}

		mockRedisClient := redismocks.NewClient(t)

		// Mock the factory to return our mock client
		originalFactory := storageFactories["REDIS"]
		storageFactories["REDIS"] = func(ctx context.Context, config any) (any, error) {
			redisCfg := config.(storageconfig.Redis)
			assert.Equal(t, "localhost:6379", redisCfg.Address)
			assert.Equal(t, "user", redisCfg.Username)
			assert.Equal(t, "pass", redisCfg.Password)
			assert.Equal(t, 1, redisCfg.DB)
			return mockRedisClient, nil
		}
		defer func() {
			storageFactories["REDIS"] = originalFactory
		}()

		picker := storageconfig.Picker{
			InlineConfig: storageconfig.InlineConfig{
				Redis: &storageconfig.Redis{
					Address:  "localhost:6379",
					Username: "user",
					Password: "pass",
					DB:       1,
				},
			},
		}

		// Execute
		client, err := Resolve[*redismocks.Client](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockRedisClient, client)
	})

	t.Run("resolves using inline PostgreSQL configuration", func(t *testing.T) {
		// Setup - clear defaults to force inline config usage
		defaults = map[string]any{}

		mockPostgreSQLClient := postgresqlmocks.NewClient(t)

		// Mock the factory to return our mock client
		originalFactory := storageFactories["POSTGRESQL"]
		storageFactories["POSTGRESQL"] = func(ctx context.Context, config any) (any, error) {
			pgCfg := config.(storageconfig.PostgreSQL)
			assert.Equal(t, "postgres://localhost/test", pgCfg.DSN)
			return mockPostgreSQLClient, nil
		}
		defer func() {
			storageFactories["POSTGRESQL"] = originalFactory
		}()

		picker := storageconfig.Picker{
			InlineConfig: storageconfig.InlineConfig{
				PostgreSQL: &storageconfig.PostgreSQL{
					DSN: "postgres://localhost/test",
				},
			},
		}

		// Execute
		client, err := Resolve[*postgresqlmocks.Client](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockPostgreSQLClient, client)
	})

	t.Run("tracks closeable connections from inline config", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}
		openedConnections = []io.Closer{} // Reset

		mockRedisClient := redismocks.NewClient(t)

		originalFactory := storageFactories["REDIS"]
		storageFactories["REDIS"] = func(ctx context.Context, config any) (any, error) {
			return mockRedisClient, nil
		}
		defer func() {
			storageFactories["REDIS"] = originalFactory
		}()

		picker := storageconfig.Picker{
			InlineConfig: storageconfig.InlineConfig{
				Redis: &storageconfig.Redis{
					Address: "localhost:6379",
				},
			},
		}

		// Execute
		_, err := Resolve[*redismocks.Client](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Len(t, openedConnections, 1)
		assert.Equal(t, mockRedisClient, openedConnections[0])
	})

	t.Run("returns error when default engine not found", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		picker := storageconfig.Picker{
			Engine: "redis",
		}

		// Execute
		client, err := Resolve[*redismocks.Client](t.Context(), picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no default instance found for selected engine \"REDIS\"")
		assert.Nil(t, client)
	})

	t.Run("returns error when default instance has unexpected type", func(t *testing.T) {
		// Setup
		defaults = map[string]any{
			"REDIS": "not a redis client", // Wrong type
		}

		picker := storageconfig.Picker{
			Engine: "redis",
		}

		// Execute
		client, err := Resolve[*redismocks.Client](t.Context(), picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "default instance for engine \"REDIS\" has unexpected type")
		assert.Nil(t, client)
	})

	t.Run("returns error when no factory registered for inline engine", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		picker := storageconfig.Picker{
			InlineConfig: storageconfig.InlineConfig{
				Redis: &storageconfig.Redis{
					Address: "localhost:6379",
				},
			},
		}

		// Temporarily remove the factory
		originalFactory := storageFactories["REDIS"]
		delete(storageFactories, "REDIS")
		defer func() {
			storageFactories["REDIS"] = originalFactory
		}()

		// Execute
		client, err := Resolve[*redismocks.Client](t.Context(), picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no factory registered for inline-configured engine \"REDIS\"")
		assert.Nil(t, client)
	})

	t.Run("returns error when inline connection creation fails", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		connectionError := errors.New("connection failed")
		originalFactory := storageFactories["REDIS"]
		storageFactories["REDIS"] = func(ctx context.Context, config any) (any, error) {
			return nil, connectionError
		}
		defer func() {
			storageFactories["REDIS"] = originalFactory
		}()

		picker := storageconfig.Picker{
			InlineConfig: storageconfig.InlineConfig{
				Redis: &storageconfig.Redis{
					Address: "localhost:6379",
				},
			},
		}

		// Execute
		client, err := Resolve[*redismocks.Client](t.Context(), picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create inline instance for engine \"REDIS\"")
		assert.Contains(t, err.Error(), "connection failed")
		assert.Nil(t, client)
	})

	t.Run("returns error when inline instance has unexpected type", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		originalFactory := storageFactories["REDIS"]
		storageFactories["REDIS"] = func(ctx context.Context, config any) (any, error) {
			return "not a redis client", nil // Wrong type
		}
		defer func() {
			storageFactories["REDIS"] = originalFactory
		}()

		picker := storageconfig.Picker{
			InlineConfig: storageconfig.InlineConfig{
				Redis: &storageconfig.Redis{
					Address: "localhost:6379",
				},
			},
		}

		// Execute
		client, err := Resolve[*redismocks.Client](t.Context(), picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inline instance for engine \"REDIS\" has unexpected type")
		assert.Nil(t, client)
	})

	t.Run("returns error when no valid storage engine configuration provided", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		picker := storageconfig.Picker{}

		// Execute
		client, err := Resolve[*redismocks.Client](t.Context(), picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no valid storage engine configuration provided")
		assert.Nil(t, client)
	})

	t.Run("handles case-insensitive engine names", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		defaults = map[string]any{
			"REDIS": mockRedisClient,
		}

		picker := storageconfig.Picker{
			Engine: "Redis", // Mixed case
		}

		// Execute
		client, err := Resolve[*redismocks.Client](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockRedisClient, client)
	})

	t.Run("handles empty engine name", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		mockRedisClient := redismocks.NewClient(t)

		picker := storageconfig.Picker{
			Engine: "", // Empty engine name should trigger inline config path
			InlineConfig: storageconfig.InlineConfig{
				Redis: &storageconfig.Redis{
					Address: "localhost:6379",
				},
			},
		}

		// Mock the factory
		originalFactory := storageFactories["REDIS"]
		storageFactories["REDIS"] = func(ctx context.Context, config any) (any, error) {
			return mockRedisClient, nil
		}
		defer func() {
			storageFactories["REDIS"] = originalFactory
		}()

		// Execute
		client, err := Resolve[*redismocks.Client](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockRedisClient, client)
	})

	t.Run("resolves with io.Closer interface", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		defaults = map[string]any{
			"REDIS": mockRedisClient,
		}

		picker := storageconfig.Picker{
			Engine: "redis",
		}

		// Execute - test with interface type
		client, err := Resolve[io.Closer](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockRedisClient, client)
	})

	t.Run("context cancellation during inline config creation", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		// Create a factory that checks context cancellation
		originalFactory := storageFactories["REDIS"]
		storageFactories["REDIS"] = func(ctx context.Context, config any) (any, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return redismocks.NewClient(t), nil
			}
		}
		defer func() {
			storageFactories["REDIS"] = originalFactory
		}()

		// Create a cancelled context
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		picker := storageconfig.Picker{
			InlineConfig: storageconfig.InlineConfig{
				Redis: &storageconfig.Redis{
					Address: "localhost:6379",
				},
			},
		}

		// Execute
		client, err := Resolve[*redismocks.Client](ctx, picker)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create inline instance for engine \"REDIS\"")
		assert.Nil(t, client)
	})

	t.Run("multiple inline configs provided - uses first non-nil", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		mockRedisClient := redismocks.NewClient(t)

		originalRedisFactory := storageFactories["REDIS"]
		storageFactories["REDIS"] = func(ctx context.Context, config any) (any, error) {
			return mockRedisClient, nil
		}
		defer func() {
			storageFactories["REDIS"] = originalRedisFactory
		}()

		picker := storageconfig.Picker{
			InlineConfig: storageconfig.InlineConfig{
				Redis: &storageconfig.Redis{
					Address: "localhost:6379",
				},
				PostgreSQL: &storageconfig.PostgreSQL{
					DSN: "postgres://localhost/test",
				},
			},
		}

		// Execute - should use Redis since it comes first in the struct
		client, err := Resolve[*redismocks.Client](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockRedisClient, client)
	})

	t.Run("resolves PostgreSQL when Redis is nil in inline config", func(t *testing.T) {
		// Setup
		defaults = map[string]any{}

		mockPostgreSQLClient := postgresqlmocks.NewClient(t)

		originalPostgreSQLFactory := storageFactories["POSTGRESQL"]
		storageFactories["POSTGRESQL"] = func(ctx context.Context, config any) (any, error) {
			return mockPostgreSQLClient, nil
		}
		defer func() {
			storageFactories["POSTGRESQL"] = originalPostgreSQLFactory
		}()

		picker := storageconfig.Picker{
			InlineConfig: storageconfig.InlineConfig{
				Redis: nil, // Nil Redis config
				PostgreSQL: &storageconfig.PostgreSQL{
					DSN: "postgres://localhost/test",
				},
			},
		}

		// Execute - should use PostgreSQL since Redis is nil
		client, err := Resolve[*postgresqlmocks.Client](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockPostgreSQLClient, client)
	})

	t.Run("uppercase engine names work correctly", func(t *testing.T) {
		// Setup
		mockRedisClient := redismocks.NewClient(t)
		defaults = map[string]any{
			"REDIS": mockRedisClient,
		}

		picker := storageconfig.Picker{
			Engine: "REDIS", // Uppercase
		}

		// Execute
		client, err := Resolve[*redismocks.Client](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockRedisClient, client)
	})

	t.Run("lowercase engine names work correctly", func(t *testing.T) {
		// Setup
		mockPostgreSQLClient := postgresqlmocks.NewClient(t)
		defaults = map[string]any{
			"POSTGRESQL": mockPostgreSQLClient,
		}

		picker := storageconfig.Picker{
			Engine: "postgresql", // Lowercase
		}

		// Execute
		client, err := Resolve[*postgresqlmocks.Client](t.Context(), picker)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, mockPostgreSQLClient, client)
	})
}
