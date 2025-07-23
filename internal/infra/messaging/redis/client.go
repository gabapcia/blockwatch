package redis

import (
	"context"
	"io"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/walletwatch"

	redis "github.com/redis/go-redis/v9"
)

// Client defines the Redis interface used by higher-level services.
//
// It wraps the low-level Redis connection and exposes adapter methods
// for domain-specific use cases such as:
//   - walletwatch.TransactionNotifier
//   - chainstream.DispatchFailureNotifier
//
// Implementations must also support graceful shutdown via io.Closer.
type Client interface {
	io.Closer

	// AsChainstreamDispatchFailureNotifier returns an adapter that publishes
	// dispatch failures to the specified Redis stream.
	AsChainstreamDispatchFailureNotifier(stream string) chainstream.DispatchFailureNotifier

	// AsWalletwatchTransactionNotifier returns an adapter that publishes
	// transaction events to the specified Redis stream.
	AsWalletwatchTransactionNotifier(stream string) walletwatch.TransactionNotifier
}

// client wraps a Redis connection and provides domain-specific adapters.
//
// It is the default implementation of the Client interface.
type client struct {
	conn *redis.Client // Underlying Redis connection
}

// Close terminates the Redis connection.
//
// Should be called during shutdown to release resources.
func (c *client) Close() error {
	return c.conn.Close()
}

// New creates and verifies a new Redis client instance.
//
// It initializes the connection using the provided credentials and performs a
// health check via the PING command before returning the client.
//
// Parameters:
//   - ctx: request-scoped context for cancellation.
//   - addr: Redis server address (e.g., "localhost:6379").
//   - username: optional ACL username (if required).
//   - password: password or token for authentication.
//   - db: Redis database index (typically 0).
//
// Returns:
//   - A fully initialized Client.
//   - An error if connection or health check fails.
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

// Compile-time assertion to ensure client implements Client interface.
var _ Client = (*client)(nil)
