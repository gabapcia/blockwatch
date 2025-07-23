package redis

import (
	"context"
	"io"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/walletregistry"
	"github.com/gabapcia/blockwatch/internal/walletwatch"

	redis "github.com/redis/go-redis/v9"
)

// Client defines the Redis-backed infrastructure contract.
//
// It combines various interfaces related to data persistence and coordination,
// and is implemented by the internal Redis client. Specifically, it supports:
//
//   - chainstream.CheckpointStorage: for persisting chainstream checkpoints.
//   - walletregistry.WalletStorage: for managing wallet registry data.
//   - walletwatch.WalletStorage: for tracking active wallets.
//   - walletwatch.IdempotencyGuard: for preventing duplicate event processing.
//   - io.Closer: for graceful shutdown.
type Client interface {
	io.Closer
	chainstream.CheckpointStorage
	walletregistry.WalletStorage
	walletwatch.WalletStorage
	walletwatch.IdempotencyGuard
}

// client provides the default Redis implementation of the Client interface.
//
// It encapsulates the low-level Redis connection and provides concrete
// implementations of the infrastructure interfaces required by the system.
type client struct {
	// conn is the underlying Redis connection.
	conn *redis.Client
}

// Close shuts down the Redis connection and releases associated resources.
//
// Returns:
//   - An error if the Redis client fails to close properly.
func (c *client) Close() error {
	return c.conn.Close()
}

// New initializes a Redis client with the given connection options,
// verifies connectivity using the PING command, and returns the client.
//
// Parameters:
//   - ctx: context for timeout and cancellation control.
//   - addr: Redis server address (e.g., "localhost:6379").
//   - username: optional Redis username (used for ACL).
//   - password: Redis password or access token.
//   - db: Redis logical database index (default is 0).
//
// Returns:
//   - A fully initialized *client instance.
//   - An error if the connection or health check fails.
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

// Compile-time check to ensure client implements the Client interface.
var _ Client = (*client)(nil)
