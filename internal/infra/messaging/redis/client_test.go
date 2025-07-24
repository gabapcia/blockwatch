package redis

import (
	"testing"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

func setupRedisContainer(t *testing.T) (*client, func()) {
	t.Helper()

	ctx := t.Context()

	// Start Redis container
	redisContainer, err := rediscontainer.Run(ctx,
		"redis:8-alpine",
		rediscontainer.WithSnapshotting(10, 1),
		rediscontainer.WithLogLevel(rediscontainer.LogLevelVerbose),
	)
	require.NoError(t, err)

	// Get connection details
	connectionString, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err)

	// Parse connection string to get host and port
	opts, err := redis.ParseURL(connectionString)
	require.NoError(t, err)

	// Create client using the New function
	redisClient, err := New(ctx, opts.Addr, opts.Username, opts.Password, opts.DB)
	require.NoError(t, err)

	// Return client and cleanup function
	cleanup := func() {
		redisClient.Close()
		redisContainer.Terminate(ctx)
	}

	return redisClient, cleanup
}

func TestNew(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		// Assert
		require.NotNil(t, client)
		err := client.conn.Ping(t.Context()).Err()
		require.NoError(t, err)
	})

	t.Run("ping error", func(t *testing.T) {
		// Execute
		client, err := New(t.Context(), "invalid:9999", "", "", 0)

		// Assert
		require.Error(t, err)
		require.Nil(t, client)
	})
}

func TestClient_Close(t *testing.T) {
	t.Run("closes the connection", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := t.Context()

		// Pre-condition: connection should be alive
		err := client.conn.Ping(ctx).Err()
		require.NoError(t, err)

		// Execute
		err = client.Close()

		// Assert
		require.NoError(t, err)

		// Post-condition: connection should be closed
		err = client.conn.Ping(ctx).Err()
		assert.Error(t, err)
	})
}
