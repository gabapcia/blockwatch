package postgresql

import (
	"context"
	"strings"

	"github.com/gabapcia/blockwatch/internal/infra/storage/postgresql/querier"
	"github.com/gabapcia/blockwatch/internal/walletwatch"
)

// FilterWatchedWallets returns the subset of given addresses that are currently
// being monitored for a specific blockchain network.
//
// It normalizes the network name to uppercase to ensure consistency with the
// values stored in the database (which are uppercased via trigger or application).
//
// Parameters:
//   - ctx: Context for cancellation and tracing.
//   - network: The name of the blockchain network (e.g., "ETHEREUM").
//   - addresses: List of wallet addresses to filter.
//
// Returns:
//   - A list of addresses that are currently monitored under the given network.
//   - An error if the query fails or context is canceled.
func (c *client) FilterWatchedWallets(ctx context.Context, network string, addresses []string) ([]string, error) {
	return c.queries.FilterMonitoredWallets(ctx, querier.FilterMonitoredWalletsParams{
		Network:   strings.ToUpper(network),
		Addresses: addresses,
	})
}

// Ensure the client implements the walletwatch.WalletStorage interface at compile-time.
var _ walletwatch.WalletStorage = new(client)
