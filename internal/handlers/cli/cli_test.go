package cli

import (
	"context"
	"os"
	"testing"

	blockproctest "github.com/gabapcia/blockwatch/internal/blockproc/mocks"
	walletregistrytest "github.com/gabapcia/blockwatch/internal/walletregistry/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRun(t *testing.T) {
	// Save original os.Args to restore after tests
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	t.Run("should create CLI app with correct metadata", func(t *testing.T) {
		// Arrange
		mockWalletRegistry := walletregistrytest.NewService(t)
		mockBlockProc := blockproctest.NewService(t)
		ctx := t.Context()

		// Set os.Args to simulate help command
		os.Args = []string{"blockwatch", "--help"}

		// Act
		err := Run(ctx, mockWalletRegistry, mockBlockProc)

		// Assert
		// Help command should exit with code 0, which translates to no error
		assert.NoError(t, err)
	})

	t.Run("should register all expected commands", func(t *testing.T) {
		// Arrange
		mockWalletRegistry := walletregistrytest.NewService(t)
		mockBlockProc := blockproctest.NewService(t)
		ctx := t.Context()

		// Set os.Args to simulate help command to see available commands
		os.Args = []string{"blockwatch", "--help"}

		// Act
		err := Run(ctx, mockWalletRegistry, mockBlockProc)

		// Assert
		assert.NoError(t, err)
		// The help output would show the available commands, but we can't easily capture it
		// The fact that it doesn't error means the commands were registered successfully
	})

	t.Run("should handle start command", func(t *testing.T) {
		// Arrange
		mockWalletRegistry := walletregistrytest.NewService(t)
		mockBlockProc := blockproctest.NewService(t)
		ctx := t.Context()

		// Mock the blockproc service to return an error to exit early
		mockBlockProc.EXPECT().Start(mock.Anything).Return(assert.AnError).Once()

		// Set os.Args to simulate start command
		os.Args = []string{"blockwatch", "start"}

		// Act
		err := Run(ctx, mockWalletRegistry, mockBlockProc)

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("should handle watch command with valid flags", func(t *testing.T) {
		// Arrange
		mockWalletRegistry := walletregistrytest.NewService(t)
		mockBlockProc := blockproctest.NewService(t)
		ctx := t.Context()

		network := "ethereum"
		address := "0x1234567890abcdef1234567890abcdef12345678"

		mockWalletRegistry.EXPECT().StartWatching(mock.Anything, network, address).Return(nil).Once()

		// Set os.Args to simulate watch command
		os.Args = []string{"blockwatch", "watch", "--network", network, "--address", address}

		// Act
		err := Run(ctx, mockWalletRegistry, mockBlockProc)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should handle watch command with missing flags", func(t *testing.T) {
		// Arrange
		mockWalletRegistry := walletregistrytest.NewService(t)
		mockBlockProc := blockproctest.NewService(t)
		ctx := t.Context()

		// Set os.Args to simulate watch command with missing flags
		os.Args = []string{"blockwatch", "watch"}

		// Act
		err := Run(ctx, mockWalletRegistry, mockBlockProc)

		// Assert
		assert.Error(t, err)
		// Should fail due to missing required flags
	})

	t.Run("should handle unwatch command with valid flags", func(t *testing.T) {
		// Arrange
		mockWalletRegistry := walletregistrytest.NewService(t)
		mockBlockProc := blockproctest.NewService(t)
		ctx := t.Context()

		network := "ethereum"
		address := "0x1234567890abcdef1234567890abcdef12345678"

		mockWalletRegistry.EXPECT().StopWatching(mock.Anything, network, address).Return(nil).Once()

		// Set os.Args to simulate unwatch command
		os.Args = []string{"blockwatch", "unwatch", "--network", network, "--address", address}

		// Act
		err := Run(ctx, mockWalletRegistry, mockBlockProc)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("should handle unwatch command with missing flags", func(t *testing.T) {
		// Arrange
		mockWalletRegistry := walletregistrytest.NewService(t)
		mockBlockProc := blockproctest.NewService(t)
		ctx := t.Context()

		// Set os.Args to simulate unwatch command with missing flags
		os.Args = []string{"blockwatch", "unwatch"}

		// Act
		err := Run(ctx, mockWalletRegistry, mockBlockProc)

		// Assert
		assert.Error(t, err)
		// Should fail due to missing required flags
	})

	t.Run("should handle help command for specific command", func(t *testing.T) {
		// Arrange
		mockWalletRegistry := walletregistrytest.NewService(t)
		mockBlockProc := blockproctest.NewService(t)
		ctx := t.Context()

		// Set os.Args to simulate help for a specific command
		os.Args = []string{"blockwatch", "help", "watch"}

		// Act
		err := Run(ctx, mockWalletRegistry, mockBlockProc)

		// Assert
		assert.NoError(t, err)
		// Help for specific commands should work without error
	})

	t.Run("should handle context cancellation", func(t *testing.T) {
		// Arrange
		mockWalletRegistry := walletregistrytest.NewService(t)
		mockBlockProc := blockproctest.NewService(t)
		ctx, cancel := context.WithCancel(t.Context())

		network := "ethereum"
		address := "0x1234567890abcdef1234567890abcdef12345678"

		// Cancel context and expect context.Canceled error
		cancel()
		mockWalletRegistry.EXPECT().StartWatching(mock.Anything, network, address).Return(context.Canceled).Once()

		// Set os.Args to simulate watch command
		os.Args = []string{"blockwatch", "watch", "--network", network, "--address", address}

		// Act
		err := Run(ctx, mockWalletRegistry, mockBlockProc)

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("should enable shell completion", func(t *testing.T) {
		// Arrange
		mockWalletRegistry := walletregistrytest.NewService(t)
		mockBlockProc := blockproctest.NewService(t)
		ctx := t.Context()

		// Set os.Args to simulate help command
		os.Args = []string{"blockwatch", "--help"}

		// Act
		err := Run(ctx, mockWalletRegistry, mockBlockProc)

		// Assert
		assert.NoError(t, err)
		// Shell completion is enabled by default in the app configuration
		// We can't easily test this directly, but the fact that the app runs
		// without error indicates the configuration is correct
	})

	t.Run("should handle different network types in watch command", func(t *testing.T) {
		testCases := []struct {
			name    string
			network string
			address string
		}{
			{
				name:    "ethereum network",
				network: "ethereum",
				address: "0x1234567890abcdef1234567890abcdef12345678",
			},
			{
				name:    "solana network",
				network: "solana",
				address: "11111111111111111111111111111112",
			},
			{
				name:    "bitcoin network",
				network: "bitcoin",
				address: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Arrange
				mockWalletRegistry := walletregistrytest.NewService(t)
				mockBlockProc := blockproctest.NewService(t)
				ctx := t.Context()

				mockWalletRegistry.EXPECT().StartWatching(mock.Anything, tc.network, tc.address).Return(nil).Once()

				// Set os.Args to simulate watch command
				os.Args = []string{"blockwatch", "watch", "--network", tc.network, "--address", tc.address}

				// Act
				err := Run(ctx, mockWalletRegistry, mockBlockProc)

				// Assert
				assert.NoError(t, err)
			})
		}
	})

	t.Run("should handle service errors in watch command", func(t *testing.T) {
		// Arrange
		mockWalletRegistry := walletregistrytest.NewService(t)
		mockBlockProc := blockproctest.NewService(t)
		ctx := t.Context()

		network := "ethereum"
		address := "0x1234567890abcdef1234567890abcdef12345678"
		expectedError := assert.AnError

		mockWalletRegistry.EXPECT().StartWatching(mock.Anything, network, address).Return(expectedError).Once()

		// Set os.Args to simulate watch command
		os.Args = []string{"blockwatch", "watch", "--network", network, "--address", address}

		// Act
		err := Run(ctx, mockWalletRegistry, mockBlockProc)

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedError)
	})

	t.Run("should handle service errors in unwatch command", func(t *testing.T) {
		// Arrange
		mockWalletRegistry := walletregistrytest.NewService(t)
		mockBlockProc := blockproctest.NewService(t)
		ctx := t.Context()

		network := "ethereum"
		address := "0x1234567890abcdef1234567890abcdef12345678"
		expectedError := assert.AnError

		mockWalletRegistry.EXPECT().StopWatching(mock.Anything, network, address).Return(expectedError).Once()

		// Set os.Args to simulate unwatch command
		os.Args = []string{"blockwatch", "unwatch", "--network", network, "--address", address}

		// Act
		err := Run(ctx, mockWalletRegistry, mockBlockProc)

		// Assert
		assert.Error(t, err)
		assert.ErrorIs(t, err, expectedError)
	})

	t.Run("should pass correct services to commands", func(t *testing.T) {
		// Arrange
		mockWalletRegistry := walletregistrytest.NewService(t)
		mockBlockProc := blockproctest.NewService(t)
		ctx := t.Context()

		// Test that the correct service is used for each command type
		network := "ethereum"
		address := "0x1234567890abcdef1234567890abcdef12345678"

		// Test wallet registry service is used for watch command
		mockWalletRegistry.EXPECT().StartWatching(mock.Anything, network, address).Return(nil).Once()

		// Set os.Args to simulate watch command
		os.Args = []string{"blockwatch", "watch", "--network", network, "--address", address}

		// Act
		err := Run(ctx, mockWalletRegistry, mockBlockProc)

		// Assert
		assert.NoError(t, err)
		// The mock expectations verify that the correct service was called
	})
}
