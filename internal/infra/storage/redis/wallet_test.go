package redis

import (
	"context"
	"testing"

	"github.com/gabapcia/blockwatch/internal/walletregistry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalletStorageKey(t *testing.T) {
	t.Run("generates correct key format", func(t *testing.T) {
		// Execute
		key := walletStorageKey("ethereum")

		// Assert
		expected := "wallet:storage:ethereum"
		assert.Equal(t, expected, key)
	})

	t.Run("handles empty network", func(t *testing.T) {
		// Execute
		key := walletStorageKey("")

		// Assert
		expected := "wallet:storage:"
		assert.Equal(t, expected, key)
	})

	t.Run("handles special characters in network", func(t *testing.T) {
		// Execute
		key := walletStorageKey("test:network-v2")

		// Assert
		expected := "wallet:storage:test:network-v2"
		assert.Equal(t, expected, key)
	})
}

func TestRegisterWallet(t *testing.T) {
	t.Run("successfully registers new wallet", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0x1234567890abcdef1234567890abcdef12345678",
		}

		// Execute
		err := client.RegisterWallet(ctx, id)

		// Assert
		require.NoError(t, err)

		// Verify the wallet was added to the Redis set
		key := walletStorageKey(id.Network)
		isMember, err := client.conn.SIsMember(ctx, key, id.Address).Result()
		require.NoError(t, err)
		assert.True(t, isMember, "wallet should be in the Redis set")
	})

	t.Run("returns ErrWalletAlreadyRegistered when wallet already exists", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0xabcdef1234567890abcdef1234567890abcdef12",
		}

		// First registration should succeed
		err := client.RegisterWallet(ctx, id)
		require.NoError(t, err)

		// Execute - second registration should fail
		err = client.RegisterWallet(ctx, id)

		// Assert
		require.Error(t, err)
		assert.Equal(t, walletregistry.ErrWalletAlreadyRegistered, err)

		// Verify the wallet is still in the set (only once)
		key := walletStorageKey(id.Network)
		count, err := client.conn.SCard(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "set should contain exactly one member")
	})

	t.Run("handles different networks independently", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		address := "0x1111111111111111111111111111111111111111"

		id1 := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: address,
		}
		id2 := walletregistry.WalletIdentifier{
			Network: "polygon",
			Address: address,
		}

		// Execute - register same address on different networks
		err1 := client.RegisterWallet(ctx, id1)
		err2 := client.RegisterWallet(ctx, id2)

		// Assert - both should succeed
		require.NoError(t, err1)
		require.NoError(t, err2)

		// Verify both wallets are in their respective sets
		key1 := walletStorageKey(id1.Network)
		key2 := walletStorageKey(id2.Network)

		isMember1, err := client.conn.SIsMember(ctx, key1, address).Result()
		require.NoError(t, err)
		assert.True(t, isMember1, "wallet should be in ethereum set")

		isMember2, err := client.conn.SIsMember(ctx, key2, address).Result()
		require.NoError(t, err)
		assert.True(t, isMember2, "wallet should be in polygon set")
	})

	t.Run("handles different addresses on same network", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "ethereum"

		id1 := walletregistry.WalletIdentifier{
			Network: network,
			Address: "0x2222222222222222222222222222222222222222",
		}
		id2 := walletregistry.WalletIdentifier{
			Network: network,
			Address: "0x3333333333333333333333333333333333333333",
		}

		// Execute - register different addresses on same network
		err1 := client.RegisterWallet(ctx, id1)
		err2 := client.RegisterWallet(ctx, id2)

		// Assert - both should succeed
		require.NoError(t, err1)
		require.NoError(t, err2)

		// Verify both wallets are in the same set
		key := walletStorageKey(network)

		isMember1, err := client.conn.SIsMember(ctx, key, id1.Address).Result()
		require.NoError(t, err)
		assert.True(t, isMember1, "first wallet should be in the set")

		isMember2, err := client.conn.SIsMember(ctx, key, id2.Address).Result()
		require.NoError(t, err)
		assert.True(t, isMember2, "second wallet should be in the set")

		// Verify set contains exactly 2 members
		count, err := client.conn.SCard(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(2), count, "set should contain exactly two members")
	})

	t.Run("handles empty network name", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		id := walletregistry.WalletIdentifier{
			Network: "",
			Address: "0x4444444444444444444444444444444444444444",
		}

		// Execute
		err := client.RegisterWallet(ctx, id)

		// Assert - should still work with empty network
		require.NoError(t, err)

		// Verify the wallet was added to the Redis set with empty network key
		key := walletStorageKey(id.Network)
		isMember, err := client.conn.SIsMember(ctx, key, id.Address).Result()
		require.NoError(t, err)
		assert.True(t, isMember, "wallet should be in the Redis set")
	})

	t.Run("handles empty address", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "",
		}

		// Execute
		err := client.RegisterWallet(ctx, id)

		// Assert - should still work with empty address
		require.NoError(t, err)

		// Verify the empty address was added to the Redis set
		key := walletStorageKey(id.Network)
		isMember, err := client.conn.SIsMember(ctx, key, id.Address).Result()
		require.NoError(t, err)
		assert.True(t, isMember, "empty address should be in the Redis set")
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0x5555555555555555555555555555555555555555",
		}

		// Execute
		err := client.RegisterWallet(ctx, id)

		// Assert - should return context error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("concurrent registrations of same wallet", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0x6666666666666666666666666666666666666666",
		}

		// Execute concurrent registrations
		errCh := make(chan error, 2)

		go func() {
			errCh <- client.RegisterWallet(ctx, id)
		}()

		go func() {
			errCh <- client.RegisterWallet(ctx, id)
		}()

		// Collect results
		err1 := <-errCh
		err2 := <-errCh

		// Assert - one should succeed, one should fail with ErrWalletAlreadyRegistered
		var successCount, alreadyRegisteredCount int
		for _, err := range []error{err1, err2} {
			if err == nil {
				successCount++
			} else if err == walletregistry.ErrWalletAlreadyRegistered {
				alreadyRegisteredCount++
			} else {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		assert.Equal(t, 1, successCount, "exactly one registration should succeed")
		assert.Equal(t, 1, alreadyRegisteredCount, "exactly one registration should fail with ErrWalletAlreadyRegistered")

		// Verify the wallet is in the set exactly once
		key := walletStorageKey(id.Network)
		count, err := client.conn.SCard(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "set should contain exactly one member")
	})

	t.Run("handles special characters in address", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0x7777:7777-7777_7777@7777#7777$7777%7777",
		}

		// Execute
		err := client.RegisterWallet(ctx, id)

		// Assert
		require.NoError(t, err)

		// Verify the wallet with special characters was added
		key := walletStorageKey(id.Network)
		isMember, err := client.conn.SIsMember(ctx, key, id.Address).Result()
		require.NoError(t, err)
		assert.True(t, isMember, "wallet with special characters should be in the Redis set")
	})

	t.Run("handles very long address", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		longAddress := "0x" + string(make([]byte, 1000)) // Very long address
		for i := range longAddress[2:] {
			longAddress = longAddress[:2+i] + "a" + longAddress[2+i+1:]
		}

		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: longAddress,
		}

		// Execute
		err := client.RegisterWallet(ctx, id)

		// Assert
		require.NoError(t, err)

		// Verify the long address was added
		key := walletStorageKey(id.Network)
		isMember, err := client.conn.SIsMember(ctx, key, id.Address).Result()
		require.NoError(t, err)
		assert.True(t, isMember, "long address should be in the Redis set")
	})

	t.Run("multiple wallets on same network maintain separate entries", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "ethereum"
		addresses := []string{
			"0x1111111111111111111111111111111111111111",
			"0x2222222222222222222222222222222222222222",
			"0x3333333333333333333333333333333333333333",
			"0x4444444444444444444444444444444444444444",
			"0x5555555555555555555555555555555555555555",
		}

		// Execute - register multiple wallets
		for _, address := range addresses {
			id := walletregistry.WalletIdentifier{
				Network: network,
				Address: address,
			}
			err := client.RegisterWallet(ctx, id)
			require.NoError(t, err, "registration should succeed for address %s", address)
		}

		// Assert - verify all wallets are in the set
		key := walletStorageKey(network)
		count, err := client.conn.SCard(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, int64(len(addresses)), count, "set should contain all registered addresses")

		// Verify each address is in the set
		for _, address := range addresses {
			isMember, err := client.conn.SIsMember(ctx, key, address).Result()
			require.NoError(t, err)
			assert.True(t, isMember, "address %s should be in the set", address)
		}

		// Verify attempting to register any existing address fails
		for _, address := range addresses {
			id := walletregistry.WalletIdentifier{
				Network: network,
				Address: address,
			}
			err := client.RegisterWallet(ctx, id)
			assert.Equal(t, walletregistry.ErrWalletAlreadyRegistered, err, "re-registration should fail for address %s", address)
		}
	})
}

func TestUnregisterWallet(t *testing.T) {
	t.Run("successfully unregisters existing wallet", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0x1234567890abcdef1234567890abcdef12345678",
		}

		// First register the wallet
		err := client.RegisterWallet(ctx, id)
		require.NoError(t, err)

		// Execute - unregister the wallet
		err = client.UnregisterWallet(ctx, id)

		// Assert
		require.NoError(t, err)

		// Verify the wallet was removed from the Redis set
		key := walletStorageKey(id.Network)
		isMember, err := client.conn.SIsMember(ctx, key, id.Address).Result()
		require.NoError(t, err)
		assert.False(t, isMember, "wallet should not be in the Redis set")
	})

	t.Run("returns ErrWalletNotFound when wallet does not exist", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0xnonexistent1234567890abcdef1234567890ab",
		}

		// Execute - try to unregister non-existent wallet
		err := client.UnregisterWallet(ctx, id)

		// Assert
		require.Error(t, err)
		assert.Equal(t, walletregistry.ErrWalletNotFound, err)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0x5555555555555555555555555555555555555555",
		}

		// Execute
		err := client.UnregisterWallet(ctx, id)

		// Assert - should return context error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

func TestFilterWatchedWallets(t *testing.T) {
	t.Run("filters watched wallets correctly", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "ethereum"

		// Register some wallets
		watchedAddresses := []string{
			"0x1111111111111111111111111111111111111111",
			"0x3333333333333333333333333333333333333333",
			"0x5555555555555555555555555555555555555555",
		}

		for _, address := range watchedAddresses {
			id := walletregistry.WalletIdentifier{
				Network: network,
				Address: address,
			}
			err := client.RegisterWallet(ctx, id)
			require.NoError(t, err)
		}

		// Prepare test addresses (mix of watched and unwatched)
		testAddresses := []string{
			"0x1111111111111111111111111111111111111111", // watched
			"0x2222222222222222222222222222222222222222", // not watched
			"0x3333333333333333333333333333333333333333", // watched
			"0x4444444444444444444444444444444444444444", // not watched
			"0x5555555555555555555555555555555555555555", // watched
		}

		// Execute
		filtered, err := client.FilterWatchedWallets(ctx, network, testAddresses)

		// Assert
		require.NoError(t, err)
		assert.Len(t, filtered, 3, "should return 3 watched wallets")
		assert.Contains(t, filtered, "0x1111111111111111111111111111111111111111")
		assert.Contains(t, filtered, "0x3333333333333333333333333333333333333333")
		assert.Contains(t, filtered, "0x5555555555555555555555555555555555555555")
		assert.NotContains(t, filtered, "0x2222222222222222222222222222222222222222")
		assert.NotContains(t, filtered, "0x4444444444444444444444444444444444444444")
	})

	t.Run("returns empty slice when no wallets are watched", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "ethereum"

		testAddresses := []string{
			"0x1111111111111111111111111111111111111111",
			"0x2222222222222222222222222222222222222222",
		}

		// Execute
		filtered, err := client.FilterWatchedWallets(ctx, network, testAddresses)

		// Assert
		require.NoError(t, err)
		assert.Empty(t, filtered, "should return empty slice when no wallets are watched")
	})

	t.Run("returns empty slice when input is empty", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx := context.Background()
		network := "ethereum"

		// Execute
		filtered, err := client.FilterWatchedWallets(ctx, network, []string{})

		// Assert
		require.NoError(t, err)
		assert.Empty(t, filtered, "should return empty slice when input is empty")
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Setup
		client, cleanup := setupRedisContainer(t)
		defer cleanup()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		testAddresses := []string{"0x1111111111111111111111111111111111111111"}

		// Execute
		_, err := client.FilterWatchedWallets(ctx, "ethereum", testAddresses)

		// Assert - should return context error
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}
