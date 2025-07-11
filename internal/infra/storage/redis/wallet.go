package redis

import (
	"context"
	"fmt"

	"github.com/gabapcia/blockwatch/internal/walletregistry"
	"github.com/gabapcia/blockwatch/internal/walletwatch"
)

// walletStoragePrefix defines the base key prefix used for storing
// watched wallet addresses in Redis.
const walletStoragePrefix = "wallet"

// walletStorageKey returns the Redis key under which watched wallet addresses
// are stored for the specified blockchain network.
//
// Format: "wallet:storage:{network}"
func walletStorageKey(network string) string {
	return fmt.Sprintf("%s:storage:%s", walletStoragePrefix, network)
}

// RegisterWallet adds the wallet address to the Redis set for the given network.
//
// If the wallet address is already registered (i.e., already present in the set),
// it returns walletregistry.ErrWalletAlreadyRegistered.
//
// Parameters:
//   - ctx: context for timeout and cancellation.
//   - id: the WalletIdentifier containing the network and address.
//
// Returns:
//   - nil on success.
//   - walletregistry.ErrWalletAlreadyRegistered if the wallet was already registered.
//   - Any Redis-related error if the operation fails.
func (c *client) RegisterWallet(ctx context.Context, id walletregistry.WalletIdentifier) error {
	key := walletStorageKey(id.Network)

	count, err := c.conn.SAdd(ctx, key, id.Address).Result()
	if err != nil {
		return err
	}

	if count == 0 {
		return walletregistry.ErrWalletAlreadyRegistered
	}

	return nil
}

// UnregisterWallet removes the wallet address from the Redis set for the given network.
//
// If the address is not found in the set, it returns walletregistry.ErrWalletNotFound.
//
// Parameters:
//   - ctx: context for timeout and cancellation.
//   - id: the WalletIdentifier containing the network and address.
//
// Returns:
//   - nil on success.
//   - walletregistry.ErrWalletNotFound if the wallet was not registered.
//   - Any Redis-related error if the operation fails.
func (c *client) UnregisterWallet(ctx context.Context, id walletregistry.WalletIdentifier) error {
	key := walletStorageKey(id.Network)

	count, err := c.conn.SRem(ctx, key, id.Address).Result()
	if err != nil {
		return err
	}

	if count == 0 {
		return walletregistry.ErrWalletNotFound
	}

	return nil
}

// Compile-time assertion to ensure *client satisfies the walletregistry.WalletStorage interface
var _ walletregistry.WalletStorage = new(client)

// FilterWatchedWallets implements the walletwatch.WalletStorage interface using Redis sets.
//
// It checks which wallet addresses from the provided list are currently being monitored
// for the given blockchain network. Internally, this uses the SMISMEMBER Redis command
// for efficient multi-member existence checks.
//
// Parameters:
//   - ctx: context used for cancellation and timeout control.
//   - network: blockchain network identifier (e.g., "ethereum").
//   - addresses: list of wallet addresses to check.
//
// Returns:
//   - A slice containing only the wallet addresses that are actively being watched.
//   - An error if the Redis query fails or cannot be completed.
func (c *client) FilterWatchedWallets(ctx context.Context, network string, addresses []string) ([]string, error) {
	key := walletStorageKey(network)

	// Convert []string to []any, required by SMIsMember Redis command
	addressesAsAny := make([]any, len(addresses))
	for i, addr := range addresses {
		addressesAsAny[i] = addr
	}

	// Perform bulk membership check in Redis set
	matchResult, err := c.conn.SMIsMember(ctx, key, addressesAsAny...).Result()
	if err != nil {
		return nil, err
	}

	// Filter the input addresses based on membership result
	matched := make([]string, 0, len(addresses))
	for i, isMember := range matchResult {
		if isMember {
			matched = append(matched, addresses[i])
		}
	}

	return matched, nil
}

// Compile-time assertion to ensure *client satisfies the walletwatch.WalletStorage interface
var _ walletwatch.WalletStorage = new(client)
