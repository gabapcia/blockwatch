package blockchain

import "github.com/gabapcia/blockwatch/internal/pkg/config/pkg"

type Networks struct {
	Ethereum *pkg.JsonRPC `validate:"omitempty"`
}
