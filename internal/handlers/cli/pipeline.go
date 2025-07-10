package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gabapcia/blockwatch/internal/blockproc"

	"github.com/urfave/cli/v3"
)

// startPipelineCommand returns a CLI command that starts the full block processing pipeline,
// including chainstream ingestion, wallet activity monitoring, and other processing stages.
//
// Usage example:
//
//	blockwatch start
//
// The process runs indefinitely until it receives an interrupt (SIGINT or SIGTERM).
func startPipelineCommand(bp blockproc.Service) *cli.Command {
	return &cli.Command{
		Name:        "start",
		Description: "Starts the block processing pipeline including chain ingestion and wallet monitoring.",
		Usage:       "Initializes and runs the full pipeline. Terminates gracefully on Ctrl+C or termination signals.",
		Action: func(ctx context.Context, c *cli.Command) error {
			quit := make(chan os.Signal, 1)
			defer close(quit)

			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

			if err := bp.Start(ctx); err != nil {
				return err
			}
			defer bp.Close()

			<-quit
			return nil
		},
	}
}
