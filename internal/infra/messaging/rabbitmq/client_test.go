package rabbitmq

import (
	"testing"

	"github.com/stretchr/testify/require"
	rabbitmqcontainer "github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

func setupRabbitMQContainer(t *testing.T) (*client, func()) {
	t.Helper()

	ctx := t.Context()
	container, err := rabbitmqcontainer.Run(ctx, "rabbitmq:4-management")
	require.NoError(t, err)

	addr, err := container.AmqpURL(ctx)
	require.NoError(t, err)

	cli, err := New(ctx, addr)
	require.NoError(t, err)

	cleanup := func() {
		cli.Close()
		require.NoError(t, container.Terminate(ctx))
	}

	return cli, cleanup
}

func setupQueue(t *testing.T, c *client, exchange, routingKey, queueName string) {
	t.Helper()

	err := c.channel.ExchangeDeclare(exchange, "direct", true, false, false, false, nil)
	require.NoError(t, err)

	_, err = c.channel.QueueDeclare(queueName, true, false, false, false, nil)
	require.NoError(t, err)

	err = c.channel.QueueBind(queueName, routingKey, exchange, false, nil)
	require.NoError(t, err)
}

func TestNew(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		// Setup
		_, cleanup := setupRabbitMQContainer(t)
		defer cleanup()
	})

	t.Run("invalid uri", func(t *testing.T) {
		// Setup
		ctx := t.Context()
		invalidURI := "amqp://guest:guest@localhost:5672/%2f"

		// Execute
		_, err := New(ctx, invalidURI)

		// Assert
		require.Error(t, err)
	})
}

func TestClient_Close(t *testing.T) {
	t.Run("closes the connection and channel", func(t *testing.T) {
		// Setup
		client, cleanup := setupRabbitMQContainer(t)
		defer cleanup()

		// Execute
		err := client.Close()

		// Assert
		require.NoError(t, err)
		require.True(t, client.conn.IsClosed())
		require.True(t, client.channel.IsClosed())
	})
}
