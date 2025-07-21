package postgresql

import (
	"context"
	"strings"

	"github.com/gabapcia/blockwatch/internal/infra/storage/postgresql/querier"
	"github.com/gabapcia/blockwatch/internal/walletwatch"
)

func (c *client) FilterWatchedWallets(ctx context.Context, network string, addresses []string) ([]string, error) {
	return c.queries.FilterMonitoredWallets(ctx, querier.FilterMonitoredWalletsParams{
		Network:   strings.ToUpper(network),
		Addresses: addresses,
	})
}

var _ walletwatch.WalletStorage = new(client)
