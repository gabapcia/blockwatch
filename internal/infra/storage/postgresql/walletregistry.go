package postgresql

import (
	"context"
	"strings"

	"github.com/gabapcia/blockwatch/internal/infra/storage/postgresql/querier"
	"github.com/gabapcia/blockwatch/internal/walletregistry"

	"github.com/google/uuid"
)

// RegisterWallet inserts a new wallet into the monitored_wallets table.
// It automatically uppercases the network name to ensure consistency.
//
// If a wallet with the same (network, address) pair already exists,
// it returns walletregistry.ErrWalletAlreadyRegistered.
//
// Parameters:
//   - ctx: context for cancellation and timeout control.
//   - id: wallet identifier containing the network and address.
//
// Returns:
//   - nil on success.
//   - walletregistry.ErrWalletAlreadyRegistered if the wallet already exists.
//   - or another error if the database operation fails.
func (c *client) RegisterWallet(ctx context.Context, id walletregistry.WalletIdentifier) error {
	err := c.queries.InsertMonitoredWallet(ctx, querier.InsertMonitoredWalletParams{
		ID:      uuid.Must(uuid.NewV7()),
		Network: strings.ToUpper(id.Network),
		Address: id.Address,
	})

	if err != nil && isUniqueViolation(err) {
		err = walletregistry.ErrWalletAlreadyRegistered
	}

	return err
}

// UnregisterWallet deletes a wallet from the monitored_wallets table based on network and address.
// The network name is uppercased before comparison to ensure consistency.
//
// If no matching wallet is found, it returns walletregistry.ErrWalletNotFound.
//
// Parameters:
//   - ctx: context for cancellation and timeout control.
//   - id: wallet identifier containing the network and address.
//
// Returns:
//   - nil on success.
//   - walletregistry.ErrWalletNotFound if the wallet does not exist.
//   - or another error if the database operation fails.
func (c *client) UnregisterWallet(ctx context.Context, id walletregistry.WalletIdentifier) error {
	rowsAffected, err := c.queries.DeleteMonitoredWalletByAddress(ctx, querier.DeleteMonitoredWalletByAddressParams{
		Network: strings.ToUpper(id.Network),
		Address: id.Address,
	})
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return walletregistry.ErrWalletNotFound
	}

	return nil
}

// Compile-time assertion that *client implements walletregistry.WalletStorage.
var _ walletregistry.WalletStorage = new(client)
