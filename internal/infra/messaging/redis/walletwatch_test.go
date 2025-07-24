package redis

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gabapcia/blockwatch/internal/walletwatch"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalletwatchTransactionNotifier_NotifyTransactions(t *testing.T) {
	client, cleanup := setupRedisContainer(t)
	defer cleanup()

	t.Run("successful notification", func(t *testing.T) {
		// Setup
		ctx := t.Context()
		stream := "test-stream"
		notifier := client.AsWalletwatchTransactionNotifier(stream)

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

		// Verify the message was published to the stream
		result, err := client.conn.XRange(ctx, stream, "-", "+").Result()
		require.NoError(t, err)
		require.Len(t, result, 1)

		msg := result[0].Values
		assert.Equal(t, network, msg["network"])
		assert.Equal(t, wallet, msg["wallet"])

		var publishedTxs []map[string]any
		err = json.Unmarshal([]byte(msg["transactions"].(string)), &publishedTxs)
		require.NoError(t, err)

		expectedTxs := []map[string]any{
			{"hash": "0xabc", "to": "0x456", "from": "0x123"},
			{"hash": "0xdef", "to": "0x123", "from": "0x789"},
		}
		assert.Equal(t, expectedTxs, publishedTxs)
	})

	t.Run("handles empty transactions", func(t *testing.T) {
		// Setup
		ctx := t.Context()
		stream := "test-stream-empty"
		notifier := client.AsWalletwatchTransactionNotifier(stream)

		network := "ethereum"
		wallet := "0x123"
		var txs []walletwatch.Transaction

		// Execute
		err := notifier.NotifyTransactions(ctx, network, wallet, txs)

		// Assert
		require.NoError(t, err)

		// Verify the message was published
		result, err := client.conn.XRange(ctx, stream, "-", "+").Result()
		require.NoError(t, err)
		require.Len(t, result, 1)

		msg := result[0].Values
		assert.Equal(t, network, msg["network"])
		assert.Equal(t, wallet, msg["wallet"])

		var publishedTxs []map[string]any
		err = json.Unmarshal([]byte(msg["transactions"].(string)), &publishedTxs)
		require.NoError(t, err)
		assert.Empty(t, publishedTxs)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Setup
		ctx, cancel := context.WithCancel(t.Context())
		stream := "test-stream-cancel"
		notifier := client.AsWalletwatchTransactionNotifier(stream)

		network := "ethereum"
		wallet := "0x123"
		txs := []walletwatch.Transaction{{Hash: "0xabc"}}

		// Cancel context before execution
		cancel()

		// Execute
		err := notifier.NotifyTransactions(ctx, network, wallet, txs)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}
