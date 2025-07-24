package bootstrap

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/handlers/cli"
)

// cliRun is an indirection over cli.Run used to allow test-time overrides.
//
// This enables tests to mock or intercept CLI execution without invoking
// the actual handler logic.
var cliRun = cli.Run

// CLI executes the application's command-line interface mode.
//
// This method delegates execution to the cli.Run handler, wiring in the
// necessary runtime dependencies such as walletregistry and blockproc
// services. It serves as the entrypoint when the application is invoked in
// standalone CLI mode
//
// Parameters:
//   - ctx: context for cancellation, timeout, or shutdown signaling.
//
// Returns:
//   - An error if CLI execution fails or terminates abnormally.
func (b *bootstrap) CLI(ctx context.Context) error {
	return cliRun(ctx, b.walletregistry, b.blockproc)
}
