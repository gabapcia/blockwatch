package walletregistry

import "context"

// Service defines the interface for registering and unregistering
// wallets that should be actively monitored for transaction activity.
//
// Implementations are responsible for validating input and delegating
// persistence to the configured WalletStorage.
type Service interface {
	// StartWatching registers a wallet for transaction monitoring.
	//
	// Parameters:
	//   - ctx: controls cancellation and timeout.
	//   - network: the name of the blockchain network (e.g., "ethereum", "solana").
	//   - address: the wallet address to monitor.
	//
	// Returns:
	//   - An error if validation fails or the registration cannot be completed.
	StartWatching(ctx context.Context, network, address string) error

	// StopWatching unregisters a wallet from transaction monitoring.
	//
	// Parameters:
	//   - ctx: controls cancellation and timeout.
	//   - network: the name of the blockchain network (e.g., "ethereum", "solana").
	//   - address: the wallet address to stop monitoring.
	//
	// Returns:
	//   - An error if validation fails or the unregistration cannot be completed.
	StopWatching(ctx context.Context, network, address string) error
}

// service is the concrete implementation of the Service interface.
// It uses a WalletStorage backend to persist registered wallets.
type service struct {
	walletStorage WalletStorage
}

// Ensure compile-time compliance with the Service interface.
var _ Service = (*service)(nil)

// New creates a new instance of the walletregistry service using the
// provided WalletStorage implementation.
//
// This constructor is intended to be used by dependency injection
// during application wiring.
func New(ws WalletStorage) *service {
	return &service{
		walletStorage: ws,
	}
}
