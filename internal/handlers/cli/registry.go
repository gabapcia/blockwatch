package cli

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/walletregistry"

	"github.com/urfave/cli/v3"
)

// startWatchingWalletCommand returns a CLI command that allows users to register
// a wallet address for activity monitoring on a specified blockchain network.
//
// Usage example:
//
//	blockwatch watch --network ethereum --address 0xABC123...
func startWatchingWalletCommand(wr walletregistry.Service) *cli.Command {
	return &cli.Command{
		Name:        "watch",
		Description: "Register a wallet to be monitored for transaction activity on a specific network.",
		Usage:       "Registers a wallet address for watching. Must provide both network and address.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "network",
				Usage:    "Blockchain network name (e.g., ethereum, solana)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "address",
				Usage:    "Wallet address to start watching",
				Required: true,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			var (
				network = c.String("network")
				address = c.String("address")
			)

			return wr.StartWatching(ctx, network, address)
		},
	}
}

// stopWatchingWalletCommand returns a CLI command that allows users to unregister
// a wallet address from being monitored on a specific blockchain network.
//
// Usage example:
//
//	blockwatch unwatch --network ethereum --address 0xABC123...
func stopWatchingWalletCommand(wr walletregistry.Service) *cli.Command {
	return &cli.Command{
		Name:        "unwatch",
		Description: "Unregister a wallet from being monitored on a specific network.",
		Usage:       "Stops watching a wallet address. Must provide both network and address.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "network",
				Usage:    "Blockchain network name (e.g., ethereum, solana)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "address",
				Usage:    "Wallet address to stop watching",
				Required: true,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			var (
				network = c.String("network")
				address = c.String("address")
			)

			return wr.StopWatching(ctx, network, address)
		},
	}
}
