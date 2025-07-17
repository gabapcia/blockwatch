package config

import (
	"github.com/gabapcia/blockwatch/internal/pkg/config/blockchain"
	"github.com/gabapcia/blockwatch/internal/pkg/config/messaging"
	"github.com/gabapcia/blockwatch/internal/pkg/config/pkg"
	"github.com/gabapcia/blockwatch/internal/pkg/config/storage"
)

type ChainStream struct {
	CheckpointStorage      *storage.Picker     `validate:"omitempty" split_words:"true"`
	Retry                  *pkg.Retry          `validate:"omitempty" split_words:"true"`
	Networks               blockchain.Networks `validate:"required" split_words:"true"`
	DispatchFailureHandler *messaging.Picker   `validate:"omitempty" split_words:"true"`
}
