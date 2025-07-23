package rabbitmq

import (
	"context"
	"errors"
	"io"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/walletwatch"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Client defines the RabbitMQ interface used by higher-level services.
//
// It exposes domain-specific adapter methods to enable messaging capabilities,
// such as:
//   - walletwatch.TransactionNotifier
//   - chainstream.DispatchFailureNotifier
//
// Implementations must also support graceful shutdown via io.Closer.
type Client interface {
	io.Closer

	// AsChainstreamDispatchFailureNotifier returns an adapter that publishes
	// dispatch failures to the specified RabbitMQ exchange and routing key.
	AsChainstreamDispatchFailureNotifier(exchange, routingKey string) chainstream.DispatchFailureNotifier

	// AsWalletwatchTransactionNotifier returns an adapter that publishes
	// transaction notifications to the specified RabbitMQ exchange and routing key.
	AsWalletwatchTransactionNotifier(exchange, routingKey string) walletwatch.TransactionNotifier
}

// client wraps a RabbitMQ connection and channel, providing domain-specific adapter methods.
//
// It is the default implementation of the Client interface.
type client struct {
	conn    *amqp.Connection // Underlying AMQP connection.
	channel *amqp.Channel    // Channel used for publishing messages.
}

// Close gracefully shuts down both the RabbitMQ channel and connection.
//
// Should be called during service shutdown to release messaging resources.
func (c *client) Close() error {
	return errors.Join(
		c.channel.Close(),
		c.conn.Close(),
	)
}

// New creates and verifies a new RabbitMQ client instance.
//
// It establishes an AMQP connection and opens a channel for message publishing.
//
// Parameters:
//   - ctx: request-scoped context for cancellation (currently unused).
//   - uri: AMQP connection URI (e.g., "amqp://user:pass@host:5672/vhost").
//
// Returns:
//   - A fully initialized RabbitMQ client.
//   - An error if connection or channel creation fails.
func New(ctx context.Context, uri string) (*client, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &client{
		conn:    conn,
		channel: channel,
	}, nil
}

// Compile-time assertion to ensure client implements the Client interface.
var _ Client = (*client)(nil)
