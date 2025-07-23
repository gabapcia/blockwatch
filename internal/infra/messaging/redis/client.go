package redis

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// Client wraps a Redis connection and provides high-level operations
// such as stream publishing or closing the connection.
type Client struct {
	conn *redis.Client // Underlying Redis Client
}

// Close gracefully closes the Redis connection.
//
// This should be called when the Client is no longer needed.
func (c *Client) Close() error {
	return c.conn.Close()
}

// New initializes a new Redis Client with the provided parameters,
// performs a health check via PING, and returns the wrapped Client.
//
// Parameters:
//   - ctx: context used to perform the PING request.
//   - addr: Redis server address (e.g., "localhost:6379").
//   - username: optional username for Redis ACL authentication.
//   - password: password or token used for authentication.
//   - db: Redis logical database index (typically 0).
//
// Returns:
//   - A pointer to the initialized Client.
//   - An error if the connection or PING test fails.
func New(ctx context.Context, addr, username, password string, db int) (*Client, error) {
	conn := redis.NewClient(&redis.Options{
		Addr:     addr,
		Username: username,
		Password: password,
		DB:       db,
	})

	if err := conn.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Client{
		conn: conn,
	}, nil
}
