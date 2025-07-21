package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_ChainStream(t *testing.T) {
	t.Run("successful processing and validation with required fields", func(t *testing.T) {
		// Set up required environment variables
		t.Setenv("NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://mainnet.infura.io/v3/key")

		var chainstream ChainStream
		ctx := t.Context()

		// Test process function
		err := process(ctx, &chainstream)
		require.NoError(t, err)

		// Verify values were loaded
		require.NotNil(t, chainstream.Networks.Ethereum)
		assert.Equal(t, "https://mainnet.infura.io/v3/key", chainstream.Networks.Ethereum.ProviderEndpoint)
		// Verify default values for HttpClient
		assert.Equal(t, 5*time.Second, chainstream.Networks.Ethereum.Timeout)
		assert.Equal(t, 1*time.Second, chainstream.Networks.Ethereum.RetryWaitMin)
		assert.Equal(t, 5*time.Second, chainstream.Networks.Ethereum.RetryWaitMax)
		assert.Equal(t, 2, chainstream.Networks.Ethereum.RetryMax)

		// Test validate function
		err = validate(chainstream)
		assert.NoError(t, err)
	})

	t.Run("successful processing with optional fields", func(t *testing.T) {
		// Set up environment variables including optional fields
		t.Setenv("NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://mainnet.infura.io/v3/key")
		t.Setenv("NETWORKS_ETHEREUM_TIMEOUT", "10s")
		t.Setenv("CHECKPOINT_STORAGE_ENGINE", "REDIS")
		t.Setenv("RETRY_ATTEMPTS", "5")
		t.Setenv("RETRY_DELAY", "2s")
		t.Setenv("RETRY_MAX_DELAY", "10s")
		t.Setenv("DISPATCH_FAILURE_HANDLER_ENGINE", "RABBITMQ")
		t.Setenv("DISPATCH_FAILURE_HANDLER_RABBITMQ_ROUTING_KEY", "dispatch.failures")

		var chainstream ChainStream
		ctx := t.Context()

		// Test process function
		err := process(ctx, &chainstream)
		require.NoError(t, err)

		// Verify required fields were loaded
		require.NotNil(t, chainstream.Networks.Ethereum)
		assert.Equal(t, "https://mainnet.infura.io/v3/key", chainstream.Networks.Ethereum.ProviderEndpoint)
		assert.Equal(t, 10*time.Second, chainstream.Networks.Ethereum.Timeout)

		// Verify optional fields were loaded
		assert.NotNil(t, chainstream.CheckpointStorage)
		assert.Equal(t, "REDIS", chainstream.CheckpointStorage.Engine)

		assert.NotNil(t, chainstream.Retry)
		assert.Equal(t, uint(5), chainstream.Retry.Attempts)
		assert.Equal(t, 2*time.Second, chainstream.Retry.Delay)
		assert.Equal(t, 10*time.Second, chainstream.Retry.MaxDelay)

		assert.NotNil(t, chainstream.DispatchFailureHandler)
		assert.Equal(t, "RABBITMQ", chainstream.DispatchFailureHandler.Engine)
		assert.Equal(t, "dispatch.failures", chainstream.DispatchFailureHandler.MessagePublisher.RabbitMQ.RoutingKey)

		// Test validate function
		err = validate(chainstream)
		assert.NoError(t, err)
	})

	t.Run("validation fails without required networks", func(t *testing.T) {
		var chainstream ChainStream
		ctx := t.Context()

		// Test process function with no env vars
		err := process(ctx, &chainstream)
		require.NoError(t, err)

		// Validation should fail due to missing required networks
		err = validate(chainstream)
		assert.Error(t, err)
	})

	t.Run("validation fails with invalid provider endpoint", func(t *testing.T) {
		// Set up invalid environment variables
		t.Setenv("NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "") // Invalid empty endpoint

		var chainstream ChainStream
		ctx := t.Context()

		err := process(ctx, &chainstream)
		require.NoError(t, err)

		// Validation should fail due to invalid provider endpoint
		err = validate(chainstream)
		assert.Error(t, err)
	})

	t.Run("successful processing with custom http client settings", func(t *testing.T) {
		// Set up environment variables with custom HTTP client settings
		t.Setenv("NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://mainnet.infura.io/v3/key")
		t.Setenv("NETWORKS_ETHEREUM_TIMEOUT", "30s")
		t.Setenv("NETWORKS_ETHEREUM_RETRY_WAIT_MIN", "2s")
		t.Setenv("NETWORKS_ETHEREUM_RETRY_WAIT_MAX", "10s")
		t.Setenv("NETWORKS_ETHEREUM_RETRY_MAX", "5")

		var chainstream ChainStream
		ctx := t.Context()

		// Test process function
		err := process(ctx, &chainstream)
		require.NoError(t, err)

		// Verify custom HTTP client values were loaded
		require.NotNil(t, chainstream.Networks.Ethereum)
		assert.Equal(t, "https://mainnet.infura.io/v3/key", chainstream.Networks.Ethereum.ProviderEndpoint)
		assert.Equal(t, 30*time.Second, chainstream.Networks.Ethereum.Timeout)
		assert.Equal(t, 2*time.Second, chainstream.Networks.Ethereum.RetryWaitMin)
		assert.Equal(t, 10*time.Second, chainstream.Networks.Ethereum.RetryWaitMax)
		assert.Equal(t, 5, chainstream.Networks.Ethereum.RetryMax)

		// Test validate function
		err = validate(chainstream)
		assert.NoError(t, err)
	})

	t.Run("successful validation with inline storage config", func(t *testing.T) {
		// Set up configuration with inline storage config instead of engine reference
		t.Setenv("NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")
		t.Setenv("CHECKPOINT_STORAGE_REDIS_ADDRESS", "localhost:6379")

		var chainstream ChainStream
		ctx := t.Context()

		err := process(ctx, &chainstream)
		require.NoError(t, err)

		// Verify inline config was loaded
		require.NotNil(t, chainstream.CheckpointStorage)
		assert.Empty(t, chainstream.CheckpointStorage.Engine)
		require.NotNil(t, chainstream.CheckpointStorage.InlineConfig.Redis)
		assert.Equal(t, "localhost:6379", chainstream.CheckpointStorage.InlineConfig.Redis.Address)

		// Validation should pass with inline config
		err = validate(chainstream)
		assert.NoError(t, err)
	})

	t.Run("successful validation with inline messaging config", func(t *testing.T) {
		// Set up configuration with inline messaging config instead of engine reference
		t.Setenv("NETWORKS_ETHEREUM_PROVIDER_ENDPOINT", "https://eth.example.com")
		t.Setenv("DISPATCH_FAILURE_HANDLER_REDIS_ADDRESS", "localhost:6379")
		t.Setenv("DISPATCH_FAILURE_HANDLER_REDIS_STREAM", "failures")

		var chainstream ChainStream
		ctx := t.Context()

		err := process(ctx, &chainstream)
		require.NoError(t, err)

		// Verify inline config was loaded
		require.NotNil(t, chainstream.DispatchFailureHandler)
		assert.Empty(t, chainstream.DispatchFailureHandler.Engine)
		require.NotNil(t, chainstream.DispatchFailureHandler.InlineConfig.Redis)
		assert.Equal(t, "localhost:6379", chainstream.DispatchFailureHandler.InlineConfig.Redis.Address)
		assert.Equal(t, "failures", chainstream.DispatchFailureHandler.MessagePublisher.Redis.Stream)

		// Validation should pass with inline config
		err = validate(chainstream)
		assert.NoError(t, err)
	})
}
