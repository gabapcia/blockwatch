package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Run("uses default configuration when no options are provided", func(t *testing.T) {
		client := NewClient()

		assert.NotNil(t, client, "NewClient should return a non-nil client")
		assert.Equal(t, 5*time.Second, client.HTTPClient.Timeout, "default HTTP timeout should be 5s")
		assert.Equal(t, 1*time.Second, client.RetryWaitMin, "default RetryWaitMin should be 1s")
		assert.Equal(t, 5*time.Second, client.RetryWaitMax, "default RetryWaitMax should be 5s")
		assert.Equal(t, 2, client.RetryMax, "default RetryMax should be 2")
	})

	t.Run("applies provided options correctly", func(t *testing.T) {
		customTimeout := 10 * time.Second
		customMin := 200 * time.Millisecond
		customMax := 10 * time.Second
		customRetries := 5

		client := NewClient(
			WithTimeout(customTimeout),
			WithRetryWaitMin(customMin),
			WithRetryWaitMax(customMax),
			WithRetryMax(customRetries),
		)

		assert.Equal(t, customTimeout, client.HTTPClient.Timeout, "custom HTTP timeout should be applied")
		assert.Equal(t, customMin, client.RetryWaitMin, "custom RetryWaitMin should be applied")
		assert.Equal(t, customMax, client.RetryWaitMax, "custom RetryWaitMax should be applied")
		assert.Equal(t, customRetries, client.RetryMax, "custom RetryMax should be applied")
	})
}

func TestWithTimeout(t *testing.T) {
	cfg := &config{}
	timeout := 10 * time.Second

	opt := WithTimeout(timeout)
	require.NotNil(t, opt)

	opt(cfg)
	assert.Equal(t, timeout, cfg.timeout, "timeout should be set correctly")
}

func TestWithRetryWaitMin(t *testing.T) {
	cfg := &config{}
	min := 500 * time.Millisecond

	opt := WithRetryWaitMin(min)
	require.NotNil(t, opt)

	opt(cfg)
	assert.Equal(t, min, cfg.retryWaitMin, "retryWaitMin should be set correctly")
}

func TestWithRetryWaitMax(t *testing.T) {
	cfg := &config{}
	max := 8 * time.Second

	opt := WithRetryWaitMax(max)
	require.NotNil(t, opt)

	opt(cfg)
	assert.Equal(t, max, cfg.retryWaitMax, "retryWaitMax should be set correctly")
}

func TestWithRetryMax(t *testing.T) {
	cfg := &config{}
	retries := 5

	opt := WithRetryMax(retries)
	require.NotNil(t, opt)

	opt(cfg)
	assert.Equal(t, retries, cfg.retryMax, "retryMax should be set correctly")
}
