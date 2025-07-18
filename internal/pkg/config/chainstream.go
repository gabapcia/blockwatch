package config

import (
	"github.com/gabapcia/blockwatch/internal/pkg/config/blockchain"
	"github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	"github.com/gabapcia/blockwatch/internal/pkg/config/pkg"
	"github.com/gabapcia/blockwatch/internal/pkg/config/storage"
)

// ChainStream defines configuration options for the chainstream use case.
type ChainStream struct {
	// CheckpointStorage specifies which storage backend to use for checkpoint persistence.
	CheckpointStorage *storage.Picker `env:", prefix=CHECKPOINT_STORAGE_" validate:"omitempty"`

	// Retry configures the retry strategy for failed block dispatch attempts.
	Retry *pkg.Retry `env:", prefix=RETRY_" validate:"omitempty"`

	// Networks contains the JSON-RPC configurations for supported blockchain networks.
	Networks blockchain.Networks `env:", prefix=NETWORKS_" validate:"required"`

	// DispatchFailureHandler selects the messaging backend used to report dispatch failures.
	DispatchFailureHandler *messaging.Picker `env:", prefix=DISPATCH_FAILURE_HANDLER_" validate:"omitempty"`
}
