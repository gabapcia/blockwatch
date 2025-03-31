package blockwatch

import (
	"reflect"
	"testing"
)

func TestGetTransactions(t *testing.T) {
	monitor, err := New(t.Context(), NewBlockchainMock(0, nil))
	if err != nil {
		t.Fatalf("unexpected error creating monitor: %v", err)
	}

	t.Run("Address Not Present", func(t *testing.T) {
		txs := monitor.GetTransactions("0xNotPresent")
		if txs != nil {
			t.Errorf("expected nil for non-existent address, got %v", txs)
		}
	})

	t.Run("Address Present Empty", func(t *testing.T) {
		address := "0xEmpty"
		monitor.transactiosByAddress[address] = nil

		txs := monitor.GetTransactions(address)
		if len(txs) != 0 {
			t.Errorf("expected nil or empty slice for address with no transactions, got %v", txs)
		}
	})

	t.Run("Address With Transactions", func(t *testing.T) {
		address := "0xWithTx"
		expected := []Transaction{
			{
				BlockNumber: 1,
				Hash:        "0xabc",
				From:        "0xFrom1",
				To:          "0xTo1",
				Value:       "100",
			},
			{
				BlockNumber: 2,
				Hash:        "0xdef",
				From:        "0xFrom2",
				To:          "0xTo2",
				Value:       "200",
			},
		}

		monitor.transactiosByAddress[address] = expected

		txs := monitor.GetTransactions(address)
		if !reflect.DeepEqual(txs, expected) {
			t.Errorf("expected transactions %v, got %v", expected, txs)
		}
	})
}
