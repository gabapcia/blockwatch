package redis

import (
	"context"

	redis "github.com/redis/go-redis/v9"
)

// client provides a Redis-backed implementation for infrastructure-level concerns,
// such as checkpointing, idempotency tracking, or message publishing.
type client struct {
	// conn is the underlying Redis client connection.
	conn *redis.Client
}

// Close shuts down the Redis connection and releases any open resources.
//
// Returns:
//   - An error if the connection could not be closed.
func (c *client) Close() error {
	return c.conn.Close()
}

// New creates a new Redis client with the specified connection parameters,
// and validates the connection by issuing a PING command.
//
// Parameters:
//   - ctx: context for timeout and cancellation.
//   - addr: address of the Redis server (e.g., "localhost:6379").
//   - username: optional Redis username (for ACL-authenticated Redis instances).
//   - password: password or token for authentication.
//   - db: Redis database index to select (typically 0).
//
// Returns:
//   - A pointer to a Redis client.
//   - An error if the connection could not be established or the PING fails.
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
