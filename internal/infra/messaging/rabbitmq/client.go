package rabbitmq

import (
	"context"
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"
)

type client struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func (c *client) Close() error {
	return errors.Join(
		c.channel.Close(),
		c.conn.Close(),
	)
}

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
