package bootstrap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBootstrap_CLI(t *testing.T) {
	// Save original os.Args to restore after tests
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	t.Run("successfully calls cli.Run with help command", func(t *testing.T) {
		b := &bootstrap{}

		// Set os.Args to simulate help command to avoid test flag conflicts
		os.Args = []string{"blockwatch", "--help"}

		// Act
		err := b.CLI(t.Context())

		// Assert
		// The function should delegate to cli.Run without error
		// Help command should exit successfully
		assert.NoError(t, err)
	})
}
