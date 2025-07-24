package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/gabapcia/blockwatch/internal/blockproc"
	blockprocmocks "github.com/gabapcia/blockwatch/internal/blockproc/mocks"
	"github.com/gabapcia/blockwatch/internal/walletregistry"
	walletregistrymocks "github.com/gabapcia/blockwatch/internal/walletregistry/mocks"

	"github.com/stretchr/testify/assert"
)

func TestBootstrap_CLI(t *testing.T) {
	t.Run("should return error when cli run fails", func(t *testing.T) {
		// arrange
		ctx := context.Background()

		expectedErr := errors.New("some error")

		walletRegistrySvc := new(walletregistrymocks.Service)
		blockProcSvc := new(blockprocmocks.Service)

		cliRun = func(ctx context.Context, wr walletregistry.Service, bp blockproc.Service) error {
			return expectedErr
		}

		b := &bootstrap{
			walletregistry: walletRegistrySvc,
			blockproc:      blockProcSvc,
		}

		// act
		err := b.CLI(ctx)

		// assert
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("should return nil when cli run succeeds", func(t *testing.T) {
		// arrange
		ctx := context.Background()

		walletRegistrySvc := new(walletregistrymocks.Service)
		blockProcSvc := new(blockprocmocks.Service)

		cliRun = func(ctx context.Context, wr walletregistry.Service, bp blockproc.Service) error {
			return nil
		}

		b := &bootstrap{
			walletregistry: walletRegistrySvc,
			blockproc:      blockProcSvc,
		}

		// act
		err := b.CLI(ctx)

		// assert
		assert.NoError(t, err)
	})
}
