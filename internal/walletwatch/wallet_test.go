package walletwatch

import (
	"github.com/gabapcia/blockwatch/internal/pkg/logger"
)

func init() {
	// Initialize logger for tests to prevent nil pointer dereference
	_ = logger.Init("error")
}
