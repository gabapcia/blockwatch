package blockwatch

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type BlockchainMock struct {
	GetLatestBlockNumberFunc func(ctx context.Context) (int64, error)
	ListBlocksSinceFunc      func(ctx context.Context, blockNumber int64) ([]Block, error)
}

func (m *BlockchainMock) GetLatestBlockNumber(ctx context.Context) (int64, error) {
	return m.GetLatestBlockNumberFunc(ctx)
}

func (m *BlockchainMock) ListBlocksSince(ctx context.Context, blockNumber int64) ([]Block, error) {
	return m.ListBlocksSinceFunc(ctx, blockNumber)
}

func NewBlockchainMock(initialBlock int64, initialErr error) *BlockchainMock {
	return &BlockchainMock{
		GetLatestBlockNumberFunc: func(ctx context.Context) (int64, error) { return initialBlock, initialErr },
	}
}

func TestSync(t *testing.T) {
	t.Run("No New Blocks", func(t *testing.T) {
		mock := NewBlockchainMock(100, nil)
		mock.ListBlocksSinceFunc = func(ctx context.Context, blockNumber int64) ([]Block, error) { return make([]Block, 0), nil }

		monitor, err := New(t.Context(), mock)
		if err != nil {
			t.Fatalf("unexpected error creating monitor: %v", err)
		}

		monitor.Subscribe("0x1")

		if err := monitor.Sync(t.Context()); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if monitor.currentBlockNumber != 100 {
			t.Errorf("expected currentBlockNumber to remain 100, got %d", monitor.currentBlockNumber)
		}

		if txs, ok := monitor.transactiosByAddress["0x1"]; !ok || len(txs) != 0 {
			t.Errorf("expected address '0x1' to have empty transactions, got %v", monitor.transactiosByAddress["0x1"])
		}
	})

	t.Run("Sync With Blocks", func(t *testing.T) {
		blocks := []Block{
			{
				Number: 101,
				Transactions: []Transaction{
					{
						BlockNumber: 101,
						Hash:        "tx1",
						From:        "0x1",
						To:          "0x3",
						Value:       "10",
					},
					{
						BlockNumber: 101,
						Hash:        "tx2",
						From:        "0x3",
						To:          "0x2",
						Value:       "20",
					},
				},
			},
			{
				Number: 102,
				Transactions: []Transaction{
					{
						BlockNumber: 102,
						Hash:        "tx3",
						From:        "0x1",
						To:          "0x2",
						Value:       "30",
					},
					{
						BlockNumber: 102,
						Hash:        "tx4",
						From:        "0x4",
						To:          "0x1",
						Value:       "40",
					},
				},
			},
		}

		mock := NewBlockchainMock(100, nil)
		mock.ListBlocksSinceFunc = func(ctx context.Context, blockNumber int64) ([]Block, error) { return blocks, nil }

		monitor, err := New(t.Context(), mock)
		if err != nil {
			t.Fatalf("unexpected error creating monitor: %v", err)
		}

		monitor.Subscribe("0x1")
		monitor.Subscribe("0x2")

		if err := monitor.Sync(t.Context()); err != nil {
			t.Fatalf("expected no error during Sync, got %v", err)
		}

		if monitor.currentBlockNumber != 102 {
			t.Errorf("expected currentBlockNumber to be 102, got %d", monitor.currentBlockNumber)
		}

		expectedTxs0x1 := []Transaction{
			blocks[0].Transactions[0], // tx1
			blocks[1].Transactions[0], // tx3
			blocks[1].Transactions[1], // tx4
		}
		if got := monitor.transactiosByAddress["0x1"]; !reflect.DeepEqual(got, expectedTxs0x1) {
			t.Errorf("for address '0x1', expected transactions %v, got %v", expectedTxs0x1, got)
		}

		expectedTxs0x2 := []Transaction{
			blocks[0].Transactions[1], // tx2
			blocks[1].Transactions[0], // tx3
		}
		if got := monitor.transactiosByAddress["0x2"]; !reflect.DeepEqual(got, expectedTxs0x2) {
			t.Errorf("for address '0x2', expected transactions %v, got %v", expectedTxs0x2, got)
		}
	})

	t.Run("List Blocks Since Error", func(t *testing.T) {
		mock := NewBlockchainMock(100, nil)
		mock.ListBlocksSinceFunc = func(ctx context.Context, blockNumber int64) ([]Block, error) { return nil, errors.New("any error") }

		monitor, err := New(t.Context(), mock)
		if err != nil {
			t.Fatalf("unexpected error creating monitor: %v", err)
		}

		if err := monitor.Sync(t.Context()); err == nil {
			t.Fatal("expected error from Sync due to ListBlocksSince error, got nil")
		}
	})
}
