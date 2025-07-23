package rabbitmq

import (
	"context"
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Client wraps a RabbitMQ connection and channel, providing a simplified interface
// for publishing and managing messaging resources.
type Client struct {
	conn    *amqp.Connection // Underlying AMQP connection.
	channel *amqp.Channel    // Channel used for publishing and other operations.
}

// Close gracefully closes both the channel and connection.
//
// It joins and returns any errors that occur during the closing process.
// This method should be called to release resources when the Client is no longer needed.
func (c *Client) Close() error {
	return errors.Join(
		c.channel.Close(),
		c.conn.Close(),
	)
}

// New creates a new RabbitMQ Client by establishing a connection and opening a channel.
//
// Parameters:
//   - ctx: Context for cancellation (currently unused but reserved for future improvements).
//   - uri: AMQP URI (e.g., "amqp://user:password@host:5672/vhost") used to connect to RabbitMQ.
//
// Returns:
//   - A pointer to the initialized Client.
//   - An error if the connection or channel creation fails.
func New(ctx context.Context, uri string) (*Client, error) {
	conn, err := amqp.Dial(uri)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:    conn,
		channel: channel,
	}, nil
}
