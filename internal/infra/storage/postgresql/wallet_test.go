package postgresql

import (
	"context"
	"testing"
	"time"

	"github.com/gabapcia/blockwatch/internal/walletregistry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterWallet(t *testing.T) {
	client, cleanup := setupPostgreSQLContainer(t)
	defer cleanup()

	t.Run("successfully registers new wallet", func(t *testing.T) {
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0x1234567890abcdef1234567890abcdef12345678",
		}

		err := client.RegisterWallet(t.Context(), id)
		require.NoError(t, err)

		var count int
		query := `SELECT COUNT(*) FROM monitored_wallets WHERE network = $1 AND address = $2`
		err = client.pool.QueryRow(t.Context(), query, "ETHEREUM", id.Address).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("returns ErrWalletAlreadyRegistered when wallet already exists", func(t *testing.T) {
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0xabcdef1234567890abcdef1234567890abcdef12",
		}

		err := client.RegisterWallet(t.Context(), id)
		require.NoError(t, err)

		err = client.RegisterWallet(t.Context(), id)
		require.Error(t, err)
		assert.Equal(t, walletregistry.ErrWalletAlreadyRegistered, err)
	})

	t.Run("network name is automatically uppercased", func(t *testing.T) {
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0x1111111111111111111111111111111111111111",
		}

		err := client.RegisterWallet(t.Context(), id)
		require.NoError(t, err)

		var storedNetwork string
		query := `SELECT network FROM monitored_wallets WHERE address = $1`
		err = client.pool.QueryRow(t.Context(), query, id.Address).Scan(&storedNetwork)
		require.NoError(t, err)
		assert.Equal(t, "ETHEREUM", storedNetwork)
	})

	t.Run("handles different networks independently", func(t *testing.T) {
		address := "0x2222222222222222222222222222222222222222"

		id1 := walletregistry.WalletIdentifier{Network: "ethereum", Address: address}
		id2 := walletregistry.WalletIdentifier{Network: "polygon", Address: address}

		err1 := client.RegisterWallet(t.Context(), id1)
		err2 := client.RegisterWallet(t.Context(), id2)

		require.NoError(t, err1)
		require.NoError(t, err2)

		var count int
		query := `SELECT COUNT(*) FROM monitored_wallets WHERE address = $1`
		err := client.pool.QueryRow(t.Context(), query, address).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("handles different addresses on same network", func(t *testing.T) {
		network := "ethereum"

		id1 := walletregistry.WalletIdentifier{Network: network, Address: "0x3333333333333333333333333333333333333333"}
		id2 := walletregistry.WalletIdentifier{Network: network, Address: "0x4444444444444444444444444444444444444444"}

		err1 := client.RegisterWallet(t.Context(), id1)
		err2 := client.RegisterWallet(t.Context(), id2)

		require.NoError(t, err1)
		require.NoError(t, err2)
	})

	t.Run("generates unique UUID for each wallet", func(t *testing.T) {
		id1 := walletregistry.WalletIdentifier{Network: "ethereum", Address: "0x5555555555555555555555555555555555555555"}
		id2 := walletregistry.WalletIdentifier{Network: "ethereum", Address: "0x6666666666666666666666666666666666666666"}

		err1 := client.RegisterWallet(t.Context(), id1)
		err2 := client.RegisterWallet(t.Context(), id2)

		require.NoError(t, err1)
		require.NoError(t, err2)

		var uuids []string
		rows, err := client.pool.Query(t.Context(), `SELECT id FROM monitored_wallets WHERE address IN ($1, $2) ORDER BY address`, id1.Address, id2.Address)
		require.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var uuid string
			err := rows.Scan(&uuid)
			require.NoError(t, err)
			uuids = append(uuids, uuid)
		}

		require.Len(t, uuids, 2)
		assert.NotEqual(t, uuids[0], uuids[1])
	})

	t.Run("sets created at timestamp", func(t *testing.T) {
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0x7777777777777777777777777777777777777777",
		}

		beforeRegistration := time.Now().Add(-time.Second) // Add buffer for timing
		err := client.RegisterWallet(t.Context(), id)
		require.NoError(t, err)
		afterRegistration := time.Now().Add(time.Second) // Add buffer for timing

		var createdAt time.Time
		query := `SELECT created_at FROM monitored_wallets WHERE network = $1 AND address = $2`
		err = client.pool.QueryRow(t.Context(), query, "ETHEREUM", id.Address).Scan(&createdAt)
		require.NoError(t, err)

		assert.True(t, createdAt.After(beforeRegistration), "created_at should be after beforeRegistration")
		assert.True(t, createdAt.Before(afterRegistration), "created_at should be before afterRegistration")
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(t.Context())
		cancel()

		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0x8888888888888888888888888888888888888888",
		}

		err := client.RegisterWallet(cancelCtx, id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("handles empty network name", func(t *testing.T) {
		id := walletregistry.WalletIdentifier{
			Network: "",
			Address: "0x9999999999999999999999999999999999999999",
		}

		err := client.RegisterWallet(t.Context(), id)
		require.NoError(t, err)

		var storedNetwork string
		query := `SELECT network FROM monitored_wallets WHERE address = $1`
		err = client.pool.QueryRow(t.Context(), query, id.Address).Scan(&storedNetwork)
		require.NoError(t, err)
		assert.Equal(t, "", storedNetwork)
	})

	t.Run("handles empty address", func(t *testing.T) {
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "",
		}

		err := client.RegisterWallet(t.Context(), id)
		require.NoError(t, err)

		var count int
		query := `SELECT COUNT(*) FROM monitored_wallets WHERE network = $1 AND address = $2`
		err = client.pool.QueryRow(t.Context(), query, "ETHEREUM", "").Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("concurrent registrations of same wallet", func(t *testing.T) {
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}

		errCh := make(chan error, 2)

		go func() {
			errCh <- client.RegisterWallet(t.Context(), id)
		}()

		go func() {
			errCh <- client.RegisterWallet(t.Context(), id)
		}()

		err1 := <-errCh
		err2 := <-errCh

		var successCount, alreadyRegisteredCount int
		for _, err := range []error{err1, err2} {
			switch err {
			case walletregistry.ErrWalletAlreadyRegistered:
				alreadyRegisteredCount++
			case nil:
				successCount++
			default:
				t.Fatalf("unexpected error: %v", err)
			}
		}

		assert.Equal(t, 1, successCount)
		assert.Equal(t, 1, alreadyRegisteredCount)
	})

	t.Run("handles mixed case network names consistently", func(t *testing.T) {
		address := "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"

		id1 := walletregistry.WalletIdentifier{Network: "ethereum", Address: address}
		err := client.RegisterWallet(t.Context(), id1)
		require.NoError(t, err)

		id2 := walletregistry.WalletIdentifier{Network: "ETHEREUM", Address: address}
		err = client.RegisterWallet(t.Context(), id2)
		assert.Equal(t, walletregistry.ErrWalletAlreadyRegistered, err)

		id3 := walletregistry.WalletIdentifier{Network: "Ethereum", Address: address}
		err = client.RegisterWallet(t.Context(), id3)
		assert.Equal(t, walletregistry.ErrWalletAlreadyRegistered, err)
	})
}

func TestUnregisterWallet(t *testing.T) {
	client, cleanup := setupPostgreSQLContainer(t)
	defer cleanup()

	t.Run("successfully unregisters existing wallet", func(t *testing.T) {
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0x1234567890abcdef1234567890abcdef12345678",
		}

		err := client.RegisterWallet(t.Context(), id)
		require.NoError(t, err)

		err = client.UnregisterWallet(t.Context(), id)
		require.NoError(t, err)

		var count int
		query := `SELECT COUNT(*) FROM monitored_wallets WHERE network = $1 AND address = $2`
		err = client.pool.QueryRow(t.Context(), query, "ETHEREUM", id.Address).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("returns ErrWalletNotFound when wallet does not exist", func(t *testing.T) {
		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0xnonexistent1234567890abcdef1234567890ab",
		}

		err := client.UnregisterWallet(t.Context(), id)
		require.Error(t, err)
		assert.Equal(t, walletregistry.ErrWalletNotFound, err)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(t.Context())
		cancel()

		id := walletregistry.WalletIdentifier{
			Network: "ethereum",
			Address: "0x5555555555555555555555555555555555555555",
		}

		err := client.UnregisterWallet(cancelCtx, id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

func TestFilterWatchedWallets(t *testing.T) {
	client, cleanup := setupPostgreSQLContainer(t)
	defer cleanup()

	network := "ethereum"

	t.Run("filters watched wallets correctly", func(t *testing.T) {
		watchedAddresses := []string{
			"0x1111111111111111111111111111111111111111",
			"0x3333333333333333333333333333333333333333",
			"0x5555555555555555555555555555555555555555",
		}

		for _, address := range watchedAddresses {
			id := walletregistry.WalletIdentifier{Network: network, Address: address}
			err := client.RegisterWallet(t.Context(), id)
			require.NoError(t, err)
		}

		testAddresses := []string{
			"0x1111111111111111111111111111111111111111", // watched
			"0x2222222222222222222222222222222222222222", // not watched
			"0x3333333333333333333333333333333333333333", // watched
			"0x4444444444444444444444444444444444444444", // not watched
			"0x5555555555555555555555555555555555555555", // watched
		}

		filtered, err := client.FilterWatchedWallets(t.Context(), network, testAddresses)
		require.NoError(t, err)
		assert.Len(t, filtered, 3)
		assert.Contains(t, filtered, "0x1111111111111111111111111111111111111111")
		assert.Contains(t, filtered, "0x3333333333333333333333333333333333333333")
		assert.Contains(t, filtered, "0x5555555555555555555555555555555555555555")
	})

	t.Run("returns empty slice when no wallets are watched", func(t *testing.T) {
		testAddresses := []string{
			"0x7777777777777777777777777777777777777777",
			"0x8888888888888888888888888888888888888888",
		}

		filtered, err := client.FilterWatchedWallets(t.Context(), network, testAddresses)
		require.NoError(t, err)
		assert.Empty(t, filtered)
	})

	t.Run("returns empty slice when input is empty", func(t *testing.T) {
		filtered, err := client.FilterWatchedWallets(t.Context(), network, []string{})
		require.NoError(t, err)
		assert.Empty(t, filtered)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(t.Context())
		cancel()

		testAddresses := []string{"0x1111111111111111111111111111111111111111"}

		_, err := client.FilterWatchedWallets(cancelCtx, "ethereum", testAddresses)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}
