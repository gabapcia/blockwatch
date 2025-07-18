package blockchain

import "github.com/gabapcia/blockwatch/internal/pkg/config/pkg"

// Supported blockchain providers.
const (
	// ProviderEthereum identifies the Ethereum network in configuration or selection logic.
	ProviderEthereum = "ETHEREUM"
)

// Networks defines the supported blockchain network configurations.
type Networks struct {
	Ethereum *pkg.JsonRPC `env:", prefix=ETHEREUM_" validate:"omitempty"` // Ethereum holds the JSON-RPC configuration for the Ethereum network.
}
