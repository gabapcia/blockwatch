package cli

import (
	"context"
	"os"

	"github.com/gabapcia/blockwatch/internal/blockproc"
	"github.com/gabapcia/blockwatch/internal/walletregistry"

	"github.com/urfave/cli/v3"
)

// Run initializes and executes the blockwatch CLI application.
//
// It registers all available commands, including:
//
//   - `start`: Starts the full block processing pipeline.
//   - `watch`: Registers a wallet for monitoring.
//   - `unwatch`: Unregisters a wallet from monitoring.
//
// Parameters:
//   - ctx: Context used to control the lifecycle of the CLI application.
//   - wr: The walletregistry service implementation used by wallet commands.
//   - bp: The blockproc service implementation used by the pipeline command.
//
// This function sets up shell completion and invokes the CLI framework to parse and run commands.
func Run(ctx context.Context, wr walletregistry.Service, bp blockproc.Service) error {
	app := &cli.Command{
		EnableShellCompletion: true,
		Name:                  "blockwatch",
		Description:           "Command-line interface for managing and running the Blockwatch pipeline.",
		Usage:                 "blockwatch [command] [flags]",
		Commands: []*cli.Command{
			startPipelineCommand(bp),
			startWatchingWalletCommand(wr),
			stopWatchingWalletCommand(wr),
		},
	}

	return app.Run(ctx, os.Args)
}
