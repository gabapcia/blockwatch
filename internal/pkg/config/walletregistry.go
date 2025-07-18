package config

import "github.com/gabapcia/blockwatch/internal/pkg/config/storage"

// WalletRegistry defines configuration options for the wallet registry use case.
type WalletRegistry struct {
	// WalletStorage selects the storage backend used to persist wallet registration data.
	WalletStorage storage.Picker `env:", prefix=WALLET_STORAGE_" validate:"required"`
}
