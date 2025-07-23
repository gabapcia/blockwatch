package bootstrap

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/handlers/cli"
)

// CLI executes the command-line interface (CLI) entrypoint.
//
// It delegates the execution to the `cli.Run` handler, passing the initialized
// wallet registry and block processor services.
//
// This method is typically called when the application is run in CLI mode.
//
// Parameters:
//   - ctx: request-scoped context for cancellation and lifecycle control.
//
// Returns:
//   - An error if the CLI execution fails.
func (b *bootstrap) CLI(ctx context.Context) error {
	return cli.Run(ctx, b.walletregistry, b.blockproc)
}
