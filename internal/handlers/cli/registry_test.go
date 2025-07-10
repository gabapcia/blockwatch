package cli

import (
	"context"
	"errors"
	"testing"

	walletregistrytest "github.com/gabapcia/blockwatch/internal/walletregistry/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/urfave/cli/v3"
)

func TestStartWatchingWalletCommand(t *testing.T) {
	t.Run("should create command with correct metadata", func(t *testing.T) {
		// Arrange
		mockService := walletregistrytest.NewService(t)

		// Act
		cmd := startWatchingWalletCommand(mockService)

		// Assert
		assert.Equal(t, "watch", cmd.Name)
		assert.Equal(t, "Register a wallet to be monitored for transaction activity on a specific network.", cmd.Description)
		assert.Equal(t, "Registers a wallet address for watching. Must provide both network and address.", cmd.Usage)
		assert.Len(t, cmd.Flags, 2)

		// Check network flag
		networkFlag := cmd.Flags[0].(*cli.StringFlag)
		assert.Equal(t, "network", networkFlag.Name)
		assert.Equal(t, "Blockchain network name (e.g., ethereum, solana)", networkFlag.Usage)
		assert.True(t, networkFlag.Required)

		// Check address flag
		addressFlag := cmd.Flags[1].(*cli.StringFlag)
		assert.Equal(t, "address", addressFlag.Name)
		assert.Equal(t, "Wallet address to start watching", addressFlag.Usage)
		assert.True(t, addressFlag.Required)
	})

	t.Run("should execute action successfully with valid flags", func(t *testing.T) {
		// Arrange
		mockService := walletregistrytest.NewService(t)
		ctx := t.Context()
		network := "ethereum"
		address := "0x1234567890abcdef1234567890abcdef12345678"

		mockService.EXPECT().StartWatching(mock.Anything, network, address).Return(nil).Once()

		cmd := startWatchingWalletCommand(mockService)

		// Create a mock CLI context with flags
		app := &cli.Command{
			Commands: []*cli.Command{cmd},
		}

		// Act & Assert
		err := app.Run(ctx, []string{"test", "watch", "--network", network, "--address", address})
		assert.NoError(t, err)
	})

	t.Run("should return error when service fails", func(t *testing.T) {
		// Arrange
		mockService := walletregistrytest.NewService(t)
		ctx := t.Context()
		network := "ethereum"
		address := "0x1234567890abcdef1234567890abcdef12345678"
		expectedError := errors.New("service error")

		mockService.EXPECT().StartWatching(mock.Anything, network, address).Return(expectedError).Once()

		cmd := startWatchingWalletCommand(mockService)

		// Create a mock CLI context with flags
		app := &cli.Command{
			Commands: []*cli.Command{cmd},
		}

		// Act
		err := app.Run(ctx, []string{"test", "watch", "--network", network, "--address", address})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service error")
	})

	t.Run("should fail when network flag is missing", func(t *testing.T) {
		// Arrange
		mockService := walletregistrytest.NewService(t)
		ctx := t.Context()
		address := "0x1234567890abcdef1234567890abcdef12345678"

		cmd := startWatchingWalletCommand(mockService)

		// Create a mock CLI context with only address flag
		app := &cli.Command{
			Commands: []*cli.Command{cmd},
		}

		// Act
		err := app.Run(ctx, []string{"test", "watch", "--address", address})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network")
	})

	t.Run("should fail when address flag is missing", func(t *testing.T) {
		// Arrange
		mockService := walletregistrytest.NewService(t)
		ctx := t.Context()
		network := "ethereum"

		cmd := startWatchingWalletCommand(mockService)

		// Create a mock CLI context with only network flag
		app := &cli.Command{
			Commands: []*cli.Command{cmd},
		}

		// Act
		err := app.Run(ctx, []string{"test", "watch", "--network", network})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "address")
	})

	t.Run("should fail when both flags are missing", func(t *testing.T) {
		// Arrange
		mockService := walletregistrytest.NewService(t)
		ctx := t.Context()

		cmd := startWatchingWalletCommand(mockService)

		// Create a mock CLI context with no flags
		app := &cli.Command{
			Commands: []*cli.Command{cmd},
		}

		// Act
		err := app.Run(ctx, []string{"test", "watch"})

		// Assert
		assert.Error(t, err)
	})

	t.Run("should handle different network types", func(t *testing.T) {
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
				mockService := walletregistrytest.NewService(t)
				ctx := t.Context()

				mockService.EXPECT().StartWatching(mock.Anything, tc.network, tc.address).Return(nil).Once()

				cmd := startWatchingWalletCommand(mockService)

				// Create a mock CLI context with flags
				app := &cli.Command{
					Commands: []*cli.Command{cmd},
				}

				// Act
				err := app.Run(ctx, []string{"test", "watch", "--network", tc.network, "--address", tc.address})

				// Assert
				assert.NoError(t, err)
			})
		}
	})

	t.Run("should handle context cancellation", func(t *testing.T) {
		// Arrange
		mockService := walletregistrytest.NewService(t)
		ctx, cancel := context.WithCancel(t.Context())
		network := "ethereum"
		address := "0x1234567890abcdef1234567890abcdef12345678"
		expectedError := context.Canceled

		// Cancel the context before the service call
		cancel()

		mockService.EXPECT().StartWatching(mock.Anything, network, address).Return(expectedError).Once()

		cmd := startWatchingWalletCommand(mockService)

		// Create a mock CLI context with flags
		app := &cli.Command{
			Commands: []*cli.Command{cmd},
		}

		// Act
		err := app.Run(ctx, []string{"test", "watch", "--network", network, "--address", address})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

func TestStopWatchingWalletCommand(t *testing.T) {
	t.Run("should create command with correct metadata", func(t *testing.T) {
		// Arrange
		mockService := walletregistrytest.NewService(t)

		// Act
		cmd := stopWatchingWalletCommand(mockService)

		// Assert
		assert.Equal(t, "unwatch", cmd.Name)
		assert.Equal(t, "Unregister a wallet from being monitored on a specific network.", cmd.Description)
		assert.Equal(t, "Stops watching a wallet address. Must provide both network and address.", cmd.Usage)
		assert.Len(t, cmd.Flags, 2)

		// Check network flag
		networkFlag := cmd.Flags[0].(*cli.StringFlag)
		assert.Equal(t, "network", networkFlag.Name)
		assert.Equal(t, "Blockchain network name (e.g., ethereum, solana)", networkFlag.Usage)
		assert.True(t, networkFlag.Required)

		// Check address flag
		addressFlag := cmd.Flags[1].(*cli.StringFlag)
		assert.Equal(t, "address", addressFlag.Name)
		assert.Equal(t, "Wallet address to stop watching", addressFlag.Usage)
		assert.True(t, addressFlag.Required)
	})

	t.Run("should execute action successfully with valid flags", func(t *testing.T) {
		// Arrange
		mockService := walletregistrytest.NewService(t)
		ctx := t.Context()
		network := "ethereum"
		address := "0x1234567890abcdef1234567890abcdef12345678"

		mockService.EXPECT().StopWatching(mock.Anything, network, address).Return(nil).Once()

		cmd := stopWatchingWalletCommand(mockService)

		// Create a mock CLI context with flags
		app := &cli.Command{
			Commands: []*cli.Command{cmd},
		}

		// Act & Assert
		err := app.Run(ctx, []string{"test", "unwatch", "--network", network, "--address", address})
		assert.NoError(t, err)
	})

	t.Run("should return error when service fails", func(t *testing.T) {
		// Arrange
		mockService := walletregistrytest.NewService(t)
		ctx := t.Context()
		network := "ethereum"
		address := "0x1234567890abcdef1234567890abcdef12345678"
		expectedError := errors.New("service error")

		mockService.EXPECT().StopWatching(mock.Anything, network, address).Return(expectedError).Once()

		cmd := stopWatchingWalletCommand(mockService)

		// Create a mock CLI context with flags
		app := &cli.Command{
			Commands: []*cli.Command{cmd},
		}

		// Act
		err := app.Run(ctx, []string{"test", "unwatch", "--network", network, "--address", address})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "service error")
	})

	t.Run("should fail when network flag is missing", func(t *testing.T) {
		// Arrange
		mockService := walletregistrytest.NewService(t)
		ctx := t.Context()
		address := "0x1234567890abcdef1234567890abcdef12345678"

		cmd := stopWatchingWalletCommand(mockService)

		// Create a mock CLI context with only address flag
		app := &cli.Command{
			Commands: []*cli.Command{cmd},
		}

		// Act
		err := app.Run(ctx, []string{"test", "unwatch", "--address", address})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network")
	})

	t.Run("should fail when address flag is missing", func(t *testing.T) {
		// Arrange
		mockService := walletregistrytest.NewService(t)
		ctx := t.Context()
		network := "ethereum"

		cmd := stopWatchingWalletCommand(mockService)

		// Create a mock CLI context with only network flag
		app := &cli.Command{
			Commands: []*cli.Command{cmd},
		}

		// Act
		err := app.Run(ctx, []string{"test", "unwatch", "--network", network})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "address")
	})

	t.Run("should fail when both flags are missing", func(t *testing.T) {
		// Arrange
		mockService := walletregistrytest.NewService(t)
		ctx := t.Context()

		cmd := stopWatchingWalletCommand(mockService)

		// Create a mock CLI context with no flags
		app := &cli.Command{
			Commands: []*cli.Command{cmd},
		}

		// Act
		err := app.Run(ctx, []string{"test", "unwatch"})

		// Assert
		assert.Error(t, err)
	})

	t.Run("should handle different network types", func(t *testing.T) {
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
				mockService := walletregistrytest.NewService(t)
				ctx := t.Context()

				mockService.EXPECT().StopWatching(mock.Anything, tc.network, tc.address).Return(nil).Once()

				cmd := stopWatchingWalletCommand(mockService)

				// Create a mock CLI context with flags
				app := &cli.Command{
					Commands: []*cli.Command{cmd},
				}

				// Act
				err := app.Run(ctx, []string{"test", "unwatch", "--network", tc.network, "--address", tc.address})

				// Assert
				assert.NoError(t, err)
			})
		}
	})

	t.Run("should handle context cancellation", func(t *testing.T) {
		// Arrange
		mockService := walletregistrytest.NewService(t)
		ctx, cancel := context.WithCancel(t.Context())
		network := "ethereum"
		address := "0x1234567890abcdef1234567890abcdef12345678"
		expectedError := context.Canceled

		// Cancel the context before the service call
		cancel()

		mockService.EXPECT().StopWatching(mock.Anything, network, address).Return(expectedError).Once()

		cmd := stopWatchingWalletCommand(mockService)

		// Create a mock CLI context with flags
		app := &cli.Command{
			Commands: []*cli.Command{cmd},
		}

		// Act
		err := app.Run(ctx, []string{"test", "unwatch", "--network", network, "--address", address})

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}
