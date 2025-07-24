package redis

import (
	"context"
	"errors"
	"testing"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/pkg/x/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainstreamDispatchFailureNotifier_NotifyDispatchFailure(t *testing.T) {
	client, cleanup := setupRedisContainer(t)
	defer cleanup()

	t.Run("successful notification", func(t *testing.T) {
		// Setup
		ctx := t.Context()
		stream := "test-stream-failure"
		notifier := client.AsChainstreamDispatchFailureNotifier(stream)

		failure := chainstream.BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x3039"), // 12345
			Errors:  []error{errors.New("error 1"), errors.New("error 2")},
		}

		// Execute
		err := notifier.NotifyDispatchFailure(ctx, failure)

		// Assert
		require.NoError(t, err)

		// Verify the message was published to the stream
		result, err := client.conn.XRange(ctx, stream, "-", "+").Result()
		require.NoError(t, err)
		require.Len(t, result, 1)

		msg := result[0].Values
		assert.Equal(t, failure.Network, msg["network"])
		assert.Equal(t, "0x3039", msg["height"]) // Redis values are strings

		expectedErrors := `["error 1","error 2"]`
		assert.JSONEq(t, expectedErrors, msg["errors"].(string))
	})

	t.Run("handles no errors", func(t *testing.T) {
		// Setup
		ctx := t.Context()
		stream := "test-stream-failure-no-errors"
		notifier := client.AsChainstreamDispatchFailureNotifier(stream)

		failure := chainstream.BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0xd431"), // 54321
			Errors:  []error{},
		}

		// Execute
		err := notifier.NotifyDispatchFailure(ctx, failure)

		// Assert
		require.NoError(t, err)

		// Verify the message was published
		result, err := client.conn.XRange(ctx, stream, "-", "+").Result()
		require.NoError(t, err)
		require.Len(t, result, 1)

		msg := result[0].Values
		assert.Equal(t, failure.Network, msg["network"])
		assert.Equal(t, "0xd431", msg["height"])
		assert.JSONEq(t, "[]", msg["errors"].(string))
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Setup
		ctx, cancel := context.WithCancel(t.Context())
		stream := "test-stream-failure-cancel"
		notifier := client.AsChainstreamDispatchFailureNotifier(stream)

		failure := chainstream.BlockDispatchFailure{
			Network: "ethereum",
			Height:  types.Hex("0x7b"), // 123
			Errors:  []error{errors.New("some error")},
		}

		// Cancel context before execution
		cancel()

		// Execute
		err := notifier.NotifyDispatchFailure(ctx, failure)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}
