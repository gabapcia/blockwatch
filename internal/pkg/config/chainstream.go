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
	CheckpointStorage *storage.Picker `validate:"omitempty" split_words:"true"`

	// Retry configures the retry strategy for failed block dispatch attempts.
	Retry *pkg.Retry `validate:"omitempty"`

	// Networks contains the JSON-RPC configurations for supported blockchain networks.
	Networks blockchain.Networks `validate:"required" split_words:"true"`

	// DispatchFailureHandler selects the messaging backend used to report dispatch failures.
	DispatchFailureHandler *messaging.Picker `validate:"omitempty" split_words:"true"`
}
