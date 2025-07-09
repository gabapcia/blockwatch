package walletregistry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("creates service with provided wallet storage", func(t *testing.T) {
		walletStorage := NewWalletStorageMock(t)

		svc := New(walletStorage)

		require.NotNil(t, svc)
		assert.Equal(t, walletStorage, svc.walletStorage)
	})

	t.Run("creates service with nil wallet storage", func(t *testing.T) {
		svc := New(nil)

		require.NotNil(t, svc)
		assert.Nil(t, svc.walletStorage)
	})
}
