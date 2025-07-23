package redis

import (
	"testing"

	redis "github.com/redis/go-redis/v9"
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
	t.Run("successful_creation", func(t *testing.T) {
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		// Verify client is not nil and implements Client interface
		require.NotNil(t, client)
		require.Implements(t, (*Client)(nil), client)

		// Verify connection works
		err := client.conn.Ping(t.Context()).Err()
		require.NoError(t, err)
	})

	t.Run("ping_error", func(t *testing.T) {
		// Test with invalid address that will cause ping to fail
		client, err := New(t.Context(), "invalid:9999", "", "", 0)
		require.Error(t, err)
		require.Nil(t, client)
	})
}
