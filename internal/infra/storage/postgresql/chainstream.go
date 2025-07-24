package postgresql

import (
	"context"
	"strings"

	"github.com/gabapcia/blockwatch/internal/chainstream"
	"github.com/gabapcia/blockwatch/internal/infra/storage/postgresql/querier/chainstreamcheckpoint"
	"github.com/gabapcia/blockwatch/internal/pkg/x/types"

	"github.com/google/uuid"
)

// SaveCheckpoint stores a new checkpoint for the given blockchain network and block height.
//
// If a checkpoint with the same network and height already exists, it will not be inserted
// (due to a unique constraint). The height is stored as a plain integer.
//
// Parameters:
//   - ctx: context for cancellation and timeouts.
//   - network: name of the blockchain network (e.g., "ETHEREUM").
//   - height: block height in hexadecimal format.
//
// Returns:
//   - error: nil on success, or an error if the operation fails.
func (c *client) SaveCheckpoint(ctx context.Context, network string, height types.Hex) error {
	return c.chainstreamCheckpoint.InsertCheckpointIfNotExists(ctx, chainstreamcheckpoint.InsertCheckpointIfNotExistsParams{
		ID:      uuid.Must(uuid.NewV7()),
		Network: strings.ToUpper(network),
		Height:  height.Int(),
	})
}

// LoadLatestCheckpoint retrieves the most recent checkpoint for the given blockchain network.
//
// This function queries the latest block height checkpointed for a specific network,
// ordering by height in descending order and limiting to one result.
//
// Parameters:
//   - ctx: context for cancellation and timeouts.
//   - network: name of the blockchain network (e.g., "ETHEREUM").
//
// Returns:
//   - types.Hex: the latest checkpointed block height as a hexadecimal string.
//   - error: ErrNoCheckpointFound if no checkpoint is available, or another error on failure.
func (c *client) LoadLatestCheckpoint(ctx context.Context, network string) (types.Hex, error) {
	height, err := c.chainstreamCheckpoint.GetLatestCheckpointByNetwork(ctx, strings.ToUpper(network))
	if err != nil {
		if isNotFoundError(err) {
			err = chainstream.ErrNoCheckpointFound
		}

		return "", err
	}

	return types.HexFromInt(height), nil
}

// Ensure that client implements the chainstream.CheckpointStorage interface.
var _ chainstream.CheckpointStorage = (*client)(nil)
