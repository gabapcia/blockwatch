package postgresql

import (
	"context"
	"strings"

	"github.com/gabapcia/blockwatch/internal/infra/storage/postgresql/internal/sqlc"
	"github.com/gabapcia/blockwatch/internal/walletregistry"

	"github.com/google/uuid"
)

func (c *client) RegisterWallet(ctx context.Context, id walletregistry.WalletIdentifier) error {
	return c.queries.InsertMonitoredWallet(ctx, sqlc.InsertMonitoredWalletParams{
		ID:      uuid.Must(uuid.NewV7()),
		Network: strings.ToUpper(id.Network),
		Address: id.Address,
	})
}

func (c *client) UnregisterWallet(ctx context.Context, id walletregistry.WalletIdentifier) error {
	return c.queries.DeleteMonitoredWalletByAddress(ctx, sqlc.DeleteMonitoredWalletByAddressParams{
		Network: strings.ToUpper(id.Network),
		Address: id.Address,
	})
}

var _ walletregistry.WalletStorage = new(client)
