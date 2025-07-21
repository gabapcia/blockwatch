package redis

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// client wraps a Redis connection and provides high-level operations
// such as stream publishing or closing the connection.
type client struct {
	conn *redis.Client // Underlying Redis client
}

// Close gracefully closes the Redis connection.
//
// This should be called when the client is no longer needed.
func (c *client) Close() error {
	return c.conn.Close()
}

// New initializes a new Redis client with the provided parameters,
// performs a health check via PING, and returns the wrapped client.
//
// Parameters:
//   - ctx: context used to perform the PING request.
//   - addr: Redis server address (e.g., "localhost:6379").
//   - username: optional username for Redis ACL authentication.
//   - password: password or token used for authentication.
//   - db: Redis logical database index (typically 0).
//
// Returns:
//   - A pointer to the initialized client.
//   - An error if the connection or PING test fails.
func New(ctx context.Context, addr, username, password string, db int) (*client, error) {
	conn := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: username,
		Password: password,
		DB:       db,
	})

	if err := conn.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &client{
		conn: conn,
	}, nil
}
