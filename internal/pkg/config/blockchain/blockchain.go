package blockchain

import "github.com/gabapcia/blockwatch/internal/pkg/config/pkg"

// Networks defines the supported blockchain network configurations.
type Networks struct {
	Ethereum *pkg.JsonRPC `validate:"omitempty"` // Ethereum holds the JSON-RPC configuration for the Ethereum network.
}
