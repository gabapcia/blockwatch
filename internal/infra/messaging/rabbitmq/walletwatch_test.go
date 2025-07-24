package rabbitmq

import (
	"encoding/json"
	"testing"

	"github.com/gabapcia/blockwatch/internal/walletwatch"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalletwatchTransactionNotifier_NotifyTransactions(t *testing.T) {
	client, cleanup := setupRabbitMQContainer(t)
	defer cleanup()

	t.Run("successful notification", func(t *testing.T) {
		// Setup
		ctx := t.Context()
		exchange := "test-exchange"
		routingKey := "test-key"
		queueName := "test-queue"

		setupQueue(t, client, exchange, routingKey, queueName)

		notifier := client.AsWalletwatchTransactionNotifier(exchange, routingKey)

		network := "ethereum"
		wallet := "0x123"
		txs := []walletwatch.Transaction{
			{Hash: "0xabc", To: "0x456", From: "0x123"},
			{Hash: "0xdef", To: "0x123", From: "0x789"},
		}

		// Execute
		err := notifier.NotifyTransactions(ctx, network, wallet, txs)

		// Assert
		require.NoError(t, err)

		// Verify the message was published
		msg, ok, err := client.channel.Get(queueName, true)
		require.NoError(t, err)
		require.True(t, ok)

		var publishedMsg walletwatchNotifyTransactionsMessage
		err = json.Unmarshal(msg.Body, &publishedMsg)
		require.NoError(t, err)

		expectedMsg := makeNotifyTransactionsMessage(network, wallet, txs)
		assert.Equal(t, expectedMsg, publishedMsg)
	})

	t.Run("handles empty transactions", func(t *testing.T) {
		// Setup
		ctx := t.Context()
		exchange := "test-exchange-empty"
		routingKey := "test-key-empty"
		queueName := "test-queue-empty"

		setupQueue(t, client, exchange, routingKey, queueName)

		notifier := client.AsWalletwatchTransactionNotifier(exchange, routingKey)

		network := "ethereum"
		wallet := "0x123"
		var txs []walletwatch.Transaction

		// Execute
		err := notifier.NotifyTransactions(ctx, network, wallet, txs)

		// Assert
		require.NoError(t, err)

		// Verify the message was published
		msg, ok, err := client.channel.Get(queueName, true)
		require.NoError(t, err)
		require.True(t, ok)

		var publishedMsg walletwatchNotifyTransactionsMessage
		err = json.Unmarshal(msg.Body, &publishedMsg)
		require.NoError(t, err)

		expectedMsg := makeNotifyTransactionsMessage(network, wallet, txs)
		assert.Equal(t, expectedMsg, publishedMsg)
	})

	t.Run("handles closed channel", func(t *testing.T) {
		// Setup
		ctx := t.Context()
		exchange := "test-exchange-closed"
		routingKey := "test-key-closed"

		// Create a new client and close its channel
		closedClient, cleanup := setupRabbitMQContainer(t)
		defer cleanup()
		notifier := closedClient.AsWalletwatchTransactionNotifier(exchange, routingKey)
		closedClient.channel.Close()

		network := "ethereum"
		wallet := "0x123"
		txs := []walletwatch.Transaction{{Hash: "0xabc"}}

		// Execute
		err := notifier.NotifyTransactions(ctx, network, wallet, txs)

		// Assert
		require.Error(t, err)
		assert.ErrorIs(t, err, amqp.ErrClosed)
	})
}
