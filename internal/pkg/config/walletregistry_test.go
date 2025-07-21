package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_WalletRegistry(t *testing.T) {
	t.Run("successful processing and validation with redis engine", func(t *testing.T) {
		// Set up required environment variables
		t.Setenv("WALLET_STORAGE_ENGINE", "REDIS")

		var walletRegistry WalletRegistry
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletRegistry)
		require.NoError(t, err)

		// Verify values were loaded
		assert.Equal(t, "REDIS", walletRegistry.WalletStorage.Engine)
		// Verify InlineConfig is empty when using Engine
		assert.Empty(t, walletRegistry.WalletStorage.InlineConfig.Redis)
		assert.Empty(t, walletRegistry.WalletStorage.InlineConfig.PostgreSQL)

		// Test validate function
		err = validate(walletRegistry)
		assert.NoError(t, err)
	})

	t.Run("successful processing and validation with postgresql engine", func(t *testing.T) {
		// Set up required environment variables
		t.Setenv("WALLET_STORAGE_ENGINE", "POSTGRESQL")

		var walletRegistry WalletRegistry
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletRegistry)
		require.NoError(t, err)

		// Verify values were loaded
		assert.Equal(t, "POSTGRESQL", walletRegistry.WalletStorage.Engine)
		// Verify InlineConfig is empty when using Engine
		assert.Empty(t, walletRegistry.WalletStorage.InlineConfig.Redis)
		assert.Empty(t, walletRegistry.WalletStorage.InlineConfig.PostgreSQL)

		// Test validate function
		err = validate(walletRegistry)
		assert.NoError(t, err)
	})

	t.Run("successful processing and validation with inline redis config", func(t *testing.T) {
		// Set up environment variables for inline Redis config
		t.Setenv("WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLET_STORAGE_REDIS_USERNAME", "testuser")
		t.Setenv("WALLET_STORAGE_REDIS_PASSWORD", "testpass")
		t.Setenv("WALLET_STORAGE_REDIS_DB", "1")

		var walletRegistry WalletRegistry
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletRegistry)
		require.NoError(t, err)

		// Verify values were loaded
		assert.Empty(t, walletRegistry.WalletStorage.Engine)
		require.NotNil(t, walletRegistry.WalletStorage.InlineConfig.Redis)
		assert.Equal(t, "localhost:6379", walletRegistry.WalletStorage.InlineConfig.Redis.Address)
		assert.Equal(t, "testuser", walletRegistry.WalletStorage.InlineConfig.Redis.Username)
		assert.Equal(t, "testpass", walletRegistry.WalletStorage.InlineConfig.Redis.Password)
		assert.Equal(t, 1, walletRegistry.WalletStorage.InlineConfig.Redis.DB)
		assert.Nil(t, walletRegistry.WalletStorage.InlineConfig.PostgreSQL)

		// Test validate function
		err = validate(walletRegistry)
		assert.NoError(t, err)
	})

	t.Run("successful processing and validation with inline postgresql config", func(t *testing.T) {
		// Set up environment variables for inline PostgreSQL config
		t.Setenv("WALLET_STORAGE_POSTGRESQL_DSN", "postgres://user:pass@localhost:5432/testdb")

		var walletRegistry WalletRegistry
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletRegistry)
		require.NoError(t, err)

		// Verify values were loaded
		assert.Empty(t, walletRegistry.WalletStorage.Engine)
		require.NotNil(t, walletRegistry.WalletStorage.InlineConfig.PostgreSQL)
		assert.Equal(t, "postgres://user:pass@localhost:5432/testdb", walletRegistry.WalletStorage.InlineConfig.PostgreSQL.DSN)
		assert.Nil(t, walletRegistry.WalletStorage.InlineConfig.Redis)

		// Test validate function
		err = validate(walletRegistry)
		assert.NoError(t, err)
	})

	t.Run("validation fails without any storage configuration", func(t *testing.T) {
		var walletRegistry WalletRegistry
		ctx := t.Context()

		// Test process function with no env vars
		err := process(ctx, &walletRegistry)
		require.NoError(t, err)

		// Verify no values were loaded
		assert.Empty(t, walletRegistry.WalletStorage.Engine)
		assert.Nil(t, walletRegistry.WalletStorage.InlineConfig.Redis)
		assert.Nil(t, walletRegistry.WalletStorage.InlineConfig.PostgreSQL)

		// Validation should fail due to missing required storage configuration
		err = validate(walletRegistry)
		assert.Error(t, err)
	})

	t.Run("validation fails with empty engine", func(t *testing.T) {
		// Set up environment variables with empty engine
		t.Setenv("WALLET_STORAGE_ENGINE", "")

		var walletRegistry WalletRegistry
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletRegistry)
		require.NoError(t, err)

		// Verify empty engine was loaded
		assert.Equal(t, "", walletRegistry.WalletStorage.Engine)

		// Validation should fail due to empty engine
		err = validate(walletRegistry)
		assert.Error(t, err)
	})

	t.Run("validation fails with invalid engine", func(t *testing.T) {
		// Set up environment variables with invalid engine
		t.Setenv("WALLET_STORAGE_ENGINE", "INVALID_ENGINE")

		var walletRegistry WalletRegistry
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletRegistry)
		require.NoError(t, err)

		// Verify invalid engine was loaded
		assert.Equal(t, "INVALID_ENGINE", walletRegistry.WalletStorage.Engine)

		// Validation should fail due to invalid engine
		err = validate(walletRegistry)
		assert.Error(t, err)
	})

	t.Run("validation fails with both engine and inline config", func(t *testing.T) {
		// Set up environment variables with both engine and inline config
		t.Setenv("WALLET_STORAGE_ENGINE", "REDIS")
		t.Setenv("WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")

		var walletRegistry WalletRegistry
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletRegistry)
		require.NoError(t, err)

		// Verify both values were loaded
		assert.Equal(t, "REDIS", walletRegistry.WalletStorage.Engine)
		require.NotNil(t, walletRegistry.WalletStorage.InlineConfig.Redis)
		assert.Equal(t, "localhost:6379", walletRegistry.WalletStorage.InlineConfig.Redis.Address)

		// Validation should fail due to mutually exclusive fields
		err = validate(walletRegistry)
		assert.Error(t, err)
	})

	t.Run("validation fails with incomplete inline redis config", func(t *testing.T) {
		// Set up environment variables with incomplete Redis config (missing required address)
		t.Setenv("WALLET_STORAGE_REDIS_USERNAME", "testuser")
		t.Setenv("WALLET_STORAGE_REDIS_PASSWORD", "testpass")

		var walletRegistry WalletRegistry
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletRegistry)
		require.NoError(t, err)

		// Verify partial values were loaded
		require.NotNil(t, walletRegistry.WalletStorage.InlineConfig.Redis)
		assert.Empty(t, walletRegistry.WalletStorage.InlineConfig.Redis.Address) // Missing required field
		assert.Equal(t, "testuser", walletRegistry.WalletStorage.InlineConfig.Redis.Username)
		assert.Equal(t, "testpass", walletRegistry.WalletStorage.InlineConfig.Redis.Password)

		// Validation should fail due to missing required address
		err = validate(walletRegistry)
		assert.Error(t, err)
	})

	t.Run("validation fails with incomplete inline postgresql config", func(t *testing.T) {
		// Set up environment variables with incomplete PostgreSQL config (missing required DSN)
		// Note: We can't easily test this since PostgreSQL only has one required field (DSN)
		// But we can test with empty DSN
		t.Setenv("WALLET_STORAGE_POSTGRESQL_DSN", "")

		var walletRegistry WalletRegistry
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletRegistry)
		require.NoError(t, err)

		// With empty DSN, the struct MUST not be created
		assert.Nil(t, walletRegistry.WalletStorage.InlineConfig.PostgreSQL)

		// Validation should fail due to missing storage configuration
		err = validate(walletRegistry)
		assert.Error(t, err)
	})

	t.Run("validation fails with multiple inline configs", func(t *testing.T) {
		// Set up environment variables with both Redis and PostgreSQL inline configs
		t.Setenv("WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("WALLET_STORAGE_POSTGRESQL_DSN", "postgres://user:pass@localhost:5432/testdb")

		var walletRegistry WalletRegistry
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletRegistry)
		require.NoError(t, err)

		// Verify both inline configs were loaded
		require.NotNil(t, walletRegistry.WalletStorage.InlineConfig.Redis)
		require.NotNil(t, walletRegistry.WalletStorage.InlineConfig.PostgreSQL)
		assert.Equal(t, "localhost:6379", walletRegistry.WalletStorage.InlineConfig.Redis.Address)
		assert.Equal(t, "postgres://user:pass@localhost:5432/testdb", walletRegistry.WalletStorage.InlineConfig.PostgreSQL.DSN)

		// Validation should fail due to multiple inline configs (required_alone validation)
		err = validate(walletRegistry)
		assert.Error(t, err)
	})

	t.Run("successful processing with minimal inline redis config", func(t *testing.T) {
		// Set up environment variables with minimal Redis config (only required field)
		t.Setenv("WALLET_STORAGE_REDIS_ADDRESS", "localhost:6379")

		var walletRegistry WalletRegistry
		ctx := t.Context()

		// Test process function
		err := process(ctx, &walletRegistry)
		require.NoError(t, err)

		// Verify minimal values were loaded
		require.NotNil(t, walletRegistry.WalletStorage.InlineConfig.Redis)
		assert.Equal(t, "localhost:6379", walletRegistry.WalletStorage.InlineConfig.Redis.Address)
		assert.Empty(t, walletRegistry.WalletStorage.InlineConfig.Redis.Username)
		assert.Empty(t, walletRegistry.WalletStorage.InlineConfig.Redis.Password)
		assert.Equal(t, 0, walletRegistry.WalletStorage.InlineConfig.Redis.DB) // Default value

		// Test validate function
		err = validate(walletRegistry)
		assert.NoError(t, err)
	})
}
