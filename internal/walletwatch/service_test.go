package walletwatch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("creates service with default configuration", func(t *testing.T) {
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		svc := New(walletStorage, transactionNotifier)

		require.NotNil(t, svc)
		assert.Equal(t, defaultMaxProcessingTime, svc.maxProcessingTime)
		assert.Equal(t, walletStorage, svc.walletStorage)
		assert.Equal(t, transactionNotifier, svc.transactionNotifier)

		// Verify that the default idempotency guard is nopIdempotencyGuard
		_, ok := svc.idempotencyGuard.(nopIdempotencyGuard)
		assert.True(t, ok, "expected default idempotency guard to be nopIdempotencyGuard")
	})

	t.Run("creates service with custom max processing time", func(t *testing.T) {
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		customTimeout := 10 * time.Minute

		svc := New(walletStorage, transactionNotifier, WithMaxProcessingTime(customTimeout))

		require.NotNil(t, svc)
		assert.Equal(t, customTimeout, svc.maxProcessingTime)
		assert.Equal(t, walletStorage, svc.walletStorage)
		assert.Equal(t, transactionNotifier, svc.transactionNotifier)

		// Verify that the default idempotency guard is still nopIdempotencyGuard
		_, ok := svc.idempotencyGuard.(nopIdempotencyGuard)
		assert.True(t, ok, "expected default idempotency guard to be nopIdempotencyGuard")
	})

	t.Run("creates service with custom idempotency guard", func(t *testing.T) {
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		idempotencyGuard := NewIdempotencyGuardMock(t)

		svc := New(walletStorage, transactionNotifier, WithIdempotencyGuard(idempotencyGuard))

		require.NotNil(t, svc)
		assert.Equal(t, defaultMaxProcessingTime, svc.maxProcessingTime)
		assert.Equal(t, walletStorage, svc.walletStorage)
		assert.Equal(t, transactionNotifier, svc.transactionNotifier)
		assert.Equal(t, idempotencyGuard, svc.idempotencyGuard)
	})

	t.Run("creates service with multiple options", func(t *testing.T) {
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		idempotencyGuard := NewIdempotencyGuardMock(t)
		customTimeout := 15 * time.Minute

		svc := New(
			walletStorage,
			transactionNotifier,
			WithMaxProcessingTime(customTimeout),
			WithIdempotencyGuard(idempotencyGuard),
		)

		require.NotNil(t, svc)
		assert.Equal(t, customTimeout, svc.maxProcessingTime)
		assert.Equal(t, walletStorage, svc.walletStorage)
		assert.Equal(t, transactionNotifier, svc.transactionNotifier)
		assert.Equal(t, idempotencyGuard, svc.idempotencyGuard)
	})

	t.Run("creates service with options applied in order", func(t *testing.T) {
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		firstTimeout := 8 * time.Minute
		secondTimeout := 12 * time.Minute

		// Apply two WithMaxProcessingTime options - the last one should win
		svc := New(
			walletStorage,
			transactionNotifier,
			WithMaxProcessingTime(firstTimeout),
			WithMaxProcessingTime(secondTimeout),
		)

		require.NotNil(t, svc)
		assert.Equal(t, secondTimeout, svc.maxProcessingTime)
		assert.Equal(t, walletStorage, svc.walletStorage)
		assert.Equal(t, transactionNotifier, svc.transactionNotifier)
	})

	t.Run("creates service with nil wallet storage", func(t *testing.T) {
		transactionNotifier := NewTransactionNotifierMock(t)

		svc := New(nil, transactionNotifier)

		require.NotNil(t, svc)
		assert.Nil(t, svc.walletStorage)
		assert.Equal(t, transactionNotifier, svc.transactionNotifier)
		assert.Equal(t, defaultMaxProcessingTime, svc.maxProcessingTime)
	})

	t.Run("creates service with nil transaction notifier", func(t *testing.T) {
		walletStorage := NewWalletStorageMock(t)

		svc := New(walletStorage, nil)

		require.NotNil(t, svc)
		assert.Equal(t, walletStorage, svc.walletStorage)
		assert.Nil(t, svc.transactionNotifier)
		assert.Equal(t, defaultMaxProcessingTime, svc.maxProcessingTime)
	})

	t.Run("creates service with both nil dependencies", func(t *testing.T) {
		svc := New(nil, nil)

		require.NotNil(t, svc)
		assert.Nil(t, svc.walletStorage)
		assert.Nil(t, svc.transactionNotifier)
		assert.Equal(t, defaultMaxProcessingTime, svc.maxProcessingTime)

		// Verify that the default idempotency guard is nopIdempotencyGuard
		_, ok := svc.idempotencyGuard.(nopIdempotencyGuard)
		assert.True(t, ok, "expected default idempotency guard to be nopIdempotencyGuard")
	})

	t.Run("creates service with zero max processing time", func(t *testing.T) {
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)

		svc := New(walletStorage, transactionNotifier, WithMaxProcessingTime(0))

		require.NotNil(t, svc)
		assert.Equal(t, time.Duration(0), svc.maxProcessingTime)
		assert.Equal(t, walletStorage, svc.walletStorage)
		assert.Equal(t, transactionNotifier, svc.transactionNotifier)
	})

	t.Run("creates service with negative max processing time", func(t *testing.T) {
		walletStorage := NewWalletStorageMock(t)
		transactionNotifier := NewTransactionNotifierMock(t)
		negativeTimeout := -5 * time.Minute

		svc := New(walletStorage, transactionNotifier, WithMaxProcessingTime(negativeTimeout))

		require.NotNil(t, svc)
		assert.Equal(t, negativeTimeout, svc.maxProcessingTime)
		assert.Equal(t, walletStorage, svc.walletStorage)
		assert.Equal(t, transactionNotifier, svc.transactionNotifier)
	})
}

func TestWithMaxProcessingTime(t *testing.T) {
	t.Run("sets max processing time correctly", func(t *testing.T) {
		customTimeout := 20 * time.Minute
		cfg := config{
			maxProcessingTime: defaultMaxProcessingTime,
			idempotencyGuard:  nopIdempotencyGuard{},
		}

		option := WithMaxProcessingTime(customTimeout)
		option(&cfg)

		assert.Equal(t, customTimeout, cfg.maxProcessingTime)
		// Verify other fields are unchanged
		_, ok := cfg.idempotencyGuard.(nopIdempotencyGuard)
		assert.True(t, ok)
	})

	t.Run("sets zero max processing time", func(t *testing.T) {
		cfg := config{
			maxProcessingTime: defaultMaxProcessingTime,
			idempotencyGuard:  nopIdempotencyGuard{},
		}

		option := WithMaxProcessingTime(0)
		option(&cfg)

		assert.Equal(t, time.Duration(0), cfg.maxProcessingTime)
	})

	t.Run("sets negative max processing time", func(t *testing.T) {
		negativeTimeout := -10 * time.Second
		cfg := config{
			maxProcessingTime: defaultMaxProcessingTime,
			idempotencyGuard:  nopIdempotencyGuard{},
		}

		option := WithMaxProcessingTime(negativeTimeout)
		option(&cfg)

		assert.Equal(t, negativeTimeout, cfg.maxProcessingTime)
	})

	t.Run("overwrites previous max processing time", func(t *testing.T) {
		firstTimeout := 5 * time.Minute
		secondTimeout := 10 * time.Minute
		cfg := config{
			maxProcessingTime: firstTimeout,
			idempotencyGuard:  nopIdempotencyGuard{},
		}

		option := WithMaxProcessingTime(secondTimeout)
		option(&cfg)

		assert.Equal(t, secondTimeout, cfg.maxProcessingTime)
	})
}

func TestWithIdempotencyGuard(t *testing.T) {
	t.Run("sets idempotency guard correctly", func(t *testing.T) {
		mockGuard := NewIdempotencyGuardMock(t)
		cfg := config{
			maxProcessingTime: defaultMaxProcessingTime,
			idempotencyGuard:  nopIdempotencyGuard{},
		}

		option := WithIdempotencyGuard(mockGuard)
		option(&cfg)

		assert.Equal(t, mockGuard, cfg.idempotencyGuard)
		// Verify other fields are unchanged
		assert.Equal(t, defaultMaxProcessingTime, cfg.maxProcessingTime)
	})

	t.Run("sets nil idempotency guard", func(t *testing.T) {
		cfg := config{
			maxProcessingTime: defaultMaxProcessingTime,
			idempotencyGuard:  nopIdempotencyGuard{},
		}

		option := WithIdempotencyGuard(nil)
		option(&cfg)

		assert.Nil(t, cfg.idempotencyGuard)
	})

	t.Run("overwrites previous idempotency guard", func(t *testing.T) {
		firstGuard := NewIdempotencyGuardMock(t)
		secondGuard := NewIdempotencyGuardMock(t)
		cfg := config{
			maxProcessingTime: defaultMaxProcessingTime,
			idempotencyGuard:  firstGuard,
		}

		option := WithIdempotencyGuard(secondGuard)
		option(&cfg)

		assert.Equal(t, secondGuard, cfg.idempotencyGuard)
	})

	t.Run("replaces nop guard with custom guard", func(t *testing.T) {
		mockGuard := NewIdempotencyGuardMock(t)
		cfg := config{
			maxProcessingTime: defaultMaxProcessingTime,
			idempotencyGuard:  nopIdempotencyGuard{},
		}

		option := WithIdempotencyGuard(mockGuard)
		option(&cfg)

		assert.Equal(t, mockGuard, cfg.idempotencyGuard)
		// Verify it's no longer the nop guard
		_, ok := cfg.idempotencyGuard.(nopIdempotencyGuard)
		assert.False(t, ok)
	})
}
