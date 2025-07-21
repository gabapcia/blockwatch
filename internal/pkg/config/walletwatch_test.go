package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_WalletWatch(t *testing.T) {
	t.Run("successful processing and validation with redis storage and rabbitmq notifier", func(t *testing.T) {
		// Set up required environment variables
		t.Setenv("WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("TRANSACTION_NOTIFIER_ENGINE", "RABBITMQ")
		t.Setenv("TRANSACTION_NOTIFIER_RABBITMQ_ROUTING_KEY", "wallet.transactions")

		var walletWatch WalletWatch
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletWatch)
		require.NoError(t, err)

		// Verify values were loaded
		assert.Equal(t, 5*time.Minute, walletWatch.MaxProcessingTime) // Default value
		assert.Equal(t, "REDIS", walletWatch.WalletStorage.Engine)
		assert.Equal(t, "RABBITMQ", walletWatch.TransactionNotifier.Engine)
		assert.Equal(t, "wallet.transactions", walletWatch.TransactionNotifier.MessagePublisher.RabbitMQ.RoutingKey)
		assert.Empty(t, walletWatch.TransactionNotifier.MessagePublisher.RabbitMQ.Exchange) // Optional field
		assert.Nil(t, walletWatch.IdempotencyGuard)                                         // Optional field

		// Test validate function
		err = validate(walletWatch)
		assert.NoError(t, err)
	})

	t.Run("successful processing with custom max processing time", func(t *testing.T) {
		// Set up environment variables with custom max processing time
		t.Setenv("MAX_PROCESSING_TIME", "10m")
		t.Setenv("WALLET_STORAGE_ENGINE", "POSTGRESQL")
		t.Setenv("TRANSACTION_NOTIFIER_ENGINE", "REDIS")
		t.Setenv("TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-transactions")

		var walletWatch WalletWatch
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletWatch)
		require.NoError(t, err)

		// Verify values were loaded
		assert.Equal(t, 10*time.Minute, walletWatch.MaxProcessingTime)
		assert.Equal(t, "POSTGRESQL", walletWatch.WalletStorage.Engine)
		assert.Equal(t, "REDIS", walletWatch.TransactionNotifier.Engine)
		assert.Equal(t, "wallet-transactions", walletWatch.TransactionNotifier.MessagePublisher.Redis.Stream)

		// Test validate function
		err = validate(walletWatch)
		assert.NoError(t, err)
	})

	t.Run("successful processing with optional idempotency guard", func(t *testing.T) {
		// Set up environment variables including optional idempotency guard
		t.Setenv("WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("TRANSACTION_NOTIFIER_ENGINE", "RABBITMQ")
		t.Setenv("TRANSACTION_NOTIFIER_RABBITMQ_ROUTING_KEY", "wallet.transactions")
		t.Setenv("IDEMPOTENCY_GUARD_ENGINE", "POSTGRESQL")

		var walletWatch WalletWatch
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletWatch)
		require.NoError(t, err)

		// Verify values were loaded
		assert.NotNil(t, walletWatch.IdempotencyGuard)
		assert.Equal(t, "POSTGRESQL", walletWatch.IdempotencyGuard.Engine)

		// Test validate function
		err = validate(walletWatch)
		assert.NoError(t, err)
	})

	t.Run("successful processing with inline storage config", func(t *testing.T) {
		// Set up environment variables with inline storage config
		t.Setenv("WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLET_STORAGE_REDIS_DB", "2")
		t.Setenv("TRANSACTION_NOTIFIER_ENGINE", "RABBITMQ")
		t.Setenv("TRANSACTION_NOTIFIER_RABBITMQ_ROUTING_KEY", "wallet.transactions")
		t.Setenv("TRANSACTION_NOTIFIER_RABBITMQ_EXCHANGE", "wallet-exchange")

		var walletWatch WalletWatch
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletWatch)
		require.NoError(t, err)

		// Verify values were loaded
		assert.Empty(t, walletWatch.WalletStorage.Engine)
		require.NotNil(t, walletWatch.WalletStorage.InlineConfig.Redis)
		assert.Equal(t, "localhost:6379", walletWatch.WalletStorage.InlineConfig.Redis.Address)
		assert.Equal(t, 2, walletWatch.WalletStorage.InlineConfig.Redis.DB)
		assert.Equal(t, "RABBITMQ", walletWatch.TransactionNotifier.Engine)
		assert.Equal(t, "wallet.transactions", walletWatch.TransactionNotifier.MessagePublisher.RabbitMQ.RoutingKey)
		assert.Equal(t, "wallet-exchange", walletWatch.TransactionNotifier.MessagePublisher.RabbitMQ.Exchange)

		// Test validate function
		err = validate(walletWatch)
		assert.NoError(t, err)
	})

	t.Run("successful processing with inline messaging config", func(t *testing.T) {
		// Set up environment variables with inline messaging config
		t.Setenv("WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("TRANSACTION_NOTIFIER_REDIS_USERNAME", "msguser")
		t.Setenv("TRANSACTION_NOTIFIER_REDIS_PASSWORD", "msgpass")
		t.Setenv("TRANSACTION_NOTIFIER_REDIS_DB", "3")
		t.Setenv("TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")

		var walletWatch WalletWatch
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletWatch)
		require.NoError(t, err)

		// Verify values were loaded
		assert.Equal(t, "REDIS", walletWatch.WalletStorage.Engine)
		assert.Empty(t, walletWatch.TransactionNotifier.Engine)
		require.NotNil(t, walletWatch.TransactionNotifier.InlineConfig.Redis)
		assert.Equal(t, "localhost:6379", walletWatch.TransactionNotifier.InlineConfig.Redis.Address)
		assert.Equal(t, "msguser", walletWatch.TransactionNotifier.InlineConfig.Redis.Username)
		assert.Equal(t, "msgpass", walletWatch.TransactionNotifier.InlineConfig.Redis.Password)
		assert.Equal(t, 3, walletWatch.TransactionNotifier.InlineConfig.Redis.DB)
		assert.Equal(t, "wallet-events", walletWatch.TransactionNotifier.MessagePublisher.Redis.Stream)

		// Test validate function
		err = validate(walletWatch)
		assert.NoError(t, err)
	})

	t.Run("validation fails without required wallet storage", func(t *testing.T) {
		// Set up environment variables missing wallet storage
		t.Setenv("TRANSACTION_NOTIFIER_ENGINE", "RABBITMQ")
		t.Setenv("TRANSACTION_NOTIFIER_RABBITMQ_ROUTING_KEY", "wallet.transactions")

		var walletWatch WalletWatch
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletWatch)
		require.NoError(t, err)

		// Validation should fail due to missing required wallet storage
		err = validate(walletWatch)
		assert.Error(t, err)
	})

	t.Run("validation fails without required transaction notifier", func(t *testing.T) {
		// Set up environment variables missing transaction notifier
		t.Setenv("WALLET_STORAGE_ENGINE", "REDIS")

		var walletWatch WalletWatch
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletWatch)
		require.NoError(t, err)

		// Validation should fail due to missing required transaction notifier
		err = validate(walletWatch)
		assert.Error(t, err)
	})

	t.Run("validation fails with incomplete messaging publisher config", func(t *testing.T) {
		// Set up environment variables with incomplete publisher config
		t.Setenv("WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("TRANSACTION_NOTIFIER_ENGINE", "RABBITMQ")
		// Missing required ROUTING_KEY

		var walletWatch WalletWatch
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletWatch)
		require.NoError(t, err)

		// Validation should fail due to missing required routing key
		err = validate(walletWatch)
		assert.Error(t, err)
	})

	t.Run("validation fails with mismatched engine and publisher", func(t *testing.T) {
		// Set up environment variables with mismatched engine and publisher
		t.Setenv("WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("TRANSACTION_NOTIFIER_ENGINE", "REDIS")
		t.Setenv("TRANSACTION_NOTIFIER_RABBITMQ_ROUTING_KEY", "wallet.transactions") // Wrong publisher for Redis engine

		var walletWatch WalletWatch
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletWatch)
		require.NoError(t, err)

		// Validation should fail due to mismatched engine and publisher
		err = validate(walletWatch)
		assert.Error(t, err)
	})

	t.Run("validation fails with both engine and inline config for messaging", func(t *testing.T) {
		// Set up environment variables with both engine and inline config
		t.Setenv("WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("TRANSACTION_NOTIFIER_ENGINE", "REDIS")
		t.Setenv("TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")

		var walletWatch WalletWatch
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletWatch)
		require.NoError(t, err)

		// Validation should fail due to mutually exclusive fields
		err = validate(walletWatch)
		assert.Error(t, err)
	})

	t.Run("validation fails with invalid max processing time", func(t *testing.T) {
		// Set up environment variables with invalid max processing time
		t.Setenv("MAX_PROCESSING_TIME", "invalid-duration")
		t.Setenv("WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("TRANSACTION_NOTIFIER_ENGINE", "RABBITMQ")
		t.Setenv("TRANSACTION_NOTIFIER_RABBITMQ_ROUTING_KEY", "wallet.transactions")

		var walletWatch WalletWatch
		ctx := t.Context()

		// Test process function should fail due to invalid duration
		err := process(ctx, &walletWatch)
		assert.Error(t, err)
	})

	t.Run("successful processing with zero max processing time", func(t *testing.T) {
		// Set up environment variables with zero max processing time
		t.Setenv("MAX_PROCESSING_TIME", "0s")
		t.Setenv("WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("TRANSACTION_NOTIFIER_ENGINE", "RABBITMQ")
		t.Setenv("TRANSACTION_NOTIFIER_RABBITMQ_ROUTING_KEY", "wallet.transactions")

		var walletWatch WalletWatch
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletWatch)
		require.NoError(t, err)

		// Verify zero duration was loaded
		assert.Equal(t, 0*time.Second, walletWatch.MaxProcessingTime)

		// Test validate function
		err = validate(walletWatch)
		assert.NoError(t, err)
	})

	t.Run("validation fails with multiple inline messaging configs", func(t *testing.T) {
		// Set up environment variables with both Redis and RabbitMQ inline configs
		t.Setenv("WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("TRANSACTION_NOTIFIER_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("TRANSACTION_NOTIFIER_RABBITMQ_URI", "amqp://localhost")
		t.Setenv("TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")

		var walletWatch WalletWatch
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletWatch)
		require.NoError(t, err)

		// Validation should fail due to multiple inline configs (required_alone validation)
		err = validate(walletWatch)
		assert.Error(t, err)
	})

	t.Run("validation fails with multiple message publishers", func(t *testing.T) {
		// Set up environment variables with both Redis and RabbitMQ publishers
		t.Setenv("WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("TRANSACTION_NOTIFIER_ENGINE", "REDIS")
		t.Setenv("TRANSACTION_NOTIFIER_REDIS_STREAM", "wallet-events")
		t.Setenv("TRANSACTION_NOTIFIER_RABBITMQ_ROUTING_KEY", "wallet.transactions")

		var walletWatch WalletWatch
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletWatch)
		require.NoError(t, err)

		// Validation should fail due to multiple publishers (required_alone validation)
		err = validate(walletWatch)
		assert.Error(t, err)
	})
}
