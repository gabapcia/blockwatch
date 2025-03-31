package blockwatch

import (
	"context"
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		var (
			blockNumber int64 = 100

			mock = &BlockchainMock{
				GetLatestBlockNumberFunc: func(ctx context.Context) (int64, error) { return blockNumber, nil },
			}
		)

		monitor, err := New(t.Context(), mock)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if monitor.currentBlockNumber != 100 {
			t.Errorf("expected currentBlockNumber to be 100, got %d", monitor.currentBlockNumber)
		}

		if monitor.blockchain != mock {
			t.Error("expected monitor.blockchain to be the dummy instance")
		}

		if len(monitor.transactiosByAddress) != 0 {
			t.Errorf("expected empty transactiosByAddress map, got %d items", len(monitor.transactiosByAddress))
		}
	})

	t.Run("Failure", func(t *testing.T) {
		var (
			mockedErr = errors.New("failed to get block number")

			mock = &BlockchainMock{
				GetLatestBlockNumberFunc: func(ctx context.Context) (int64, error) { return 0, mockedErr },
			}
		)

		monitor, err := New(t.Context(), mock)
		if !errors.Is(err, mockedErr) {
			t.Fatal("expected error, got nil")
		}

		if monitor != nil {
			t.Errorf("expected monitor to be nil on error, got %+v", monitor)
		}
	})
}
