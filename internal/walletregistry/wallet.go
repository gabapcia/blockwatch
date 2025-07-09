package walletregistry

import (
	"context"

	"github.com/gabapcia/blockwatch/internal/pkg/validator"
)

// WalletIdentifier uniquely identifies a wallet to be monitored,
// using a combination of blockchain network and address.
//
// Both fields are required and validated upon creation.
type WalletIdentifier struct {
	Network string `validate:"required"` // Blockchain network (e.g., "ethereum", "solana")
	Address string `validate:"required"` // Wallet address to be watched
}

// WalletStorage defines the persistence interface for storing and removing
// wallet identifiers that have opted into monitoring.
//
// This interface allows different storage backends to manage which wallets
// are being actively watched for transaction activity.
type WalletStorage interface {
	// RegisterWallet adds the given WalletIdentifier to the list of watched wallets.
	//
	// This method should be idempotent and safe to call multiple times with the same ID.
	RegisterWallet(ctx context.Context, id WalletIdentifier) error

	// UnregisterWallet removes the given WalletIdentifier from the list of watched wallets.
	//
	// After this call, the wallet should no longer receive transaction notifications.
	UnregisterWallet(ctx context.Context, id WalletIdentifier) error
}

// buildWalletIdentifier constructs and validates a WalletIdentifier using the
// given network and address. It returns an error if validation fails.
//
// This is a utility function used to enforce correct input before persistence.
func buildWalletIdentifier(network, address string) (WalletIdentifier, error) {
	id := WalletIdentifier{
		Network: network,
		Address: address,
	}

	return id, validator.Validate(id)
}

// StartWatching registers a wallet for monitoring based on its network and address.
//
// It validates the input, constructs a WalletIdentifier, and persists it using WalletStorage.
func (s *service) StartWatching(ctx context.Context, network, address string) error {
	id, err := buildWalletIdentifier(network, address)
	if err != nil {
		return err
	}

	return s.walletStorage.RegisterWallet(ctx, id)
}

// StopWatching unregisters a wallet from monitoring based on its network and address.
//
// It validates the input, constructs a WalletIdentifier, and removes it using WalletStorage.
func (s *service) StopWatching(ctx context.Context, network, address string) error {
	id, err := buildWalletIdentifier(network, address)
	if err != nil {
		return err
	}

	return s.walletStorage.UnregisterWallet(ctx, id)
}
