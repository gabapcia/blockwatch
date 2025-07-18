package config

import (
	"time"

	"github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	"github.com/gabapcia/blockwatch/internal/pkg/config/storage"
)

// WalletWatch defines configuration options for the wallet transaction watcher use case.
type WalletWatch struct {
	// MaxProcessingTime sets the maximum allowed time for processing a single block.
	// Default: 5m
	MaxProcessingTime time.Duration `env:"MAX_PROCESSING_TIME, default=5m"`

	// IdempotencyGuard selects the storage backend used to prevent duplicate block processing.
	IdempotencyGuard *storage.Picker `env:", prefix=IDEMPOTENCY_GUARD_" validate:"omitempty"`

	// WalletStorage selects the storage backend used to retrieve registered wallets.
	WalletStorage storage.Picker `env:", prefix=WALLET_STORAGE_" validate:"required"`

	// TransactionNotifier selects the messaging backend used to publish detected transactions.
	TransactionNotifier messaging.Picker `env:", prefix=TRANSACTION_NOTIFIER_" validate:"required"`
}
