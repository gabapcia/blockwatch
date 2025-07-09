package walletregistry

import (
	"errors"
	"testing"

	"github.com/gabapcia/blockwatch/internal/pkg/validator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildWalletIdentifier(t *testing.T) {
	validator.Init()

	t.Run("should build and validate a correct identifier", func(t *testing.T) {
		id, err := buildWalletIdentifier("ethereum", "0x123")
		require.NoError(t, err)
		assert.Equal(t, "ethereum", id.Network)
		assert.Equal(t, "0x123", id.Address)
	})

	t.Run("should return a validation error if network is missing", func(t *testing.T) {
		_, err := buildWalletIdentifier("", "0x123")
		require.Error(t, err)
		assert.ErrorIs(t, err, validator.ErrValidation)
	})

	t.Run("should return a validation error if address is missing", func(t *testing.T) {
		_, err := buildWalletIdentifier("ethereum", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, validator.ErrValidation)
	})
}

func TestService_StartWatching(t *testing.T) {
	validator.Init()

	t.Run("should register a wallet for monitoring", func(t *testing.T) {
		ctx := t.Context()
		storage := NewWalletStorageMock(t)
		s := &service{walletStorage: storage}

		id := WalletIdentifier{
			Network: "ethereum",
			Address: "0x123",
		}

		storage.EXPECT().RegisterWallet(ctx, id).Return(nil).Once()

		err := s.StartWatching(ctx, "ethereum", "0x123")
		require.NoError(t, err)
	})
	t.Run("should return an error if wallet identifier build fails", func(t *testing.T) {
		ctx := t.Context()
		storage := NewWalletStorageMock(t)
		s := &service{walletStorage: storage}

		err := s.StartWatching(ctx, "", "0x123")
		require.Error(t, err)
		assert.ErrorIs(t, err, validator.ErrValidation)
	})

	t.Run("should return an error if wallet storage fails", func(t *testing.T) {
		ctx := t.Context()
		storage := NewWalletStorageMock(t)
		s := &service{walletStorage: storage}

		id := WalletIdentifier{
			Network: "ethereum",
			Address: "0x123",
		}

		expectedErr := errors.New("storage error")
		storage.EXPECT().RegisterWallet(ctx, id).Return(expectedErr).Once()

		err := s.StartWatching(ctx, "ethereum", "0x123")
		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("should return an error if wallet is already registered", func(t *testing.T) {
		ctx := t.Context()
		storage := NewWalletStorageMock(t)
		s := &service{walletStorage: storage}

		id := WalletIdentifier{
			Network: "ethereum",
			Address: "0x123",
		}

		storage.EXPECT().RegisterWallet(ctx, id).Return(ErrWalletAlreadyRegistered).Once()

		err := s.StartWatching(ctx, "ethereum", "0x123")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrWalletAlreadyRegistered)
	})
}

func TestService_StopWatching(t *testing.T) {
	validator.Init()

	t.Run("should unregister a wallet from monitoring", func(t *testing.T) {
		ctx := t.Context()
		storage := NewWalletStorageMock(t)
		s := &service{walletStorage: storage}

		id := WalletIdentifier{
			Network: "ethereum",
			Address: "0x123",
		}

		storage.EXPECT().UnregisterWallet(ctx, id).Return(nil).Once()

		err := s.StopWatching(ctx, "ethereum", "0x123")
		require.NoError(t, err)
	})
	t.Run("should return an error if wallet identifier build fails", func(t *testing.T) {
		ctx := t.Context()
		storage := NewWalletStorageMock(t)
		s := &service{walletStorage: storage}

		err := s.StopWatching(ctx, "ethereum", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, validator.ErrValidation)
	})

	t.Run("should return an error if wallet storage fails", func(t *testing.T) {
		ctx := t.Context()
		storage := NewWalletStorageMock(t)
		s := &service{walletStorage: storage}

		id := WalletIdentifier{
			Network: "ethereum",
			Address: "0x123",
		}

		expectedErr := errors.New("storage error")
		storage.EXPECT().UnregisterWallet(ctx, id).Return(expectedErr).Once()

		err := s.StopWatching(ctx, "ethereum", "0x123")
		require.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("should return an error if wallet not found", func(t *testing.T) {
		ctx := t.Context()
		storage := NewWalletStorageMock(t)
		s := &service{walletStorage: storage}

		id := WalletIdentifier{
			Network: "ethereum",
			Address: "0x456",
		}

		storage.EXPECT().UnregisterWallet(ctx, id).Return(ErrWalletNotFound).Once()

		err := s.StopWatching(ctx, "ethereum", "0x456")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrWalletNotFound)
	})
}
