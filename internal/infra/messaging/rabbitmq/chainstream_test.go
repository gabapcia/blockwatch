package rabbitmq

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/pkg/x/types"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainstreamDispatchFailureNotifier_NotifyDispatchFailure(t *testing.T) {
	client, cleanup := setupRabbitMQContainer(t)
	defer cleanup()

	t.Run("successful notification", func(t *testing.T) {
		// Setup
		ctx := t.Context()
		exchange := "test-exchange-failure"
		routingKey := "test-key-failure"
		queueName := "test-queue-failure"

		setupQueue(t, client, exchange, routingKey, queueName)

		notifier := client.AsChainstreamDispatchFailureNotifier(exchange, routingKey)

		failure := chainstream.BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x1"),
			Errors:  []error{errors.New("error 1"), errors.New("error 2")},
		}

		// Execute
		err := notifier.NotifyDispatchFailure(ctx, failure)

		// Assert
		require.NoError(t, err)

		// Verify the message was published
		msg, ok, err := client.channel.Get(queueName, true)
		require.NoError(t, err)
		require.True(t, ok)

		var publishedMsg chainstreamBlockDispatchFailureMessage
		err = json.Unmarshal(msg.Body, &publishedMsg)
		require.NoError(t, err)

		expectedMsg := makeBlockDispatchFailureMessage(failure)
		assert.Equal(t, expectedMsg, publishedMsg)
	})

	t.Run("handles closed channel", func(t *testing.T) {
		// Setup
		ctx := t.Context()
		exchange := "test-exchange-failure-closed"
		routingKey := "test-key-failure-closed"

		// Create a new client and close its channel
		closedClient, cleanup := setupRabbitMQContainer(t)
		defer cleanup()
		notifier := closedClient.AsChainstreamDispatchFailureNotifier(exchange, routingKey)
		closedClient.channel.Close()

		failure := chainstream.BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x1"),
			Errors:  []error{errors.New("error 1")},
		}

		// Execute
		err := notifier.NotifyDispatchFailure(ctx, failure)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, amqp.ErrClosed)
	})
}
