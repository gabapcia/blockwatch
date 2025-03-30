package memory

import (
	"reflect"
	"testing"

	"github.com/gabapcia/blockwatch"
)

func TestSaveAddressTransactions(t *testing.T) {
	t.Run("New Address Transactions", func(t *testing.T) {
		s := New()
		address := "0xABC123"
		txs := []blockwatch.Transaction{
			{
				BlockNumber: 1,
				Hash:        "0x1",
				From:        "0xFrom1",
				To:          "0xTo1",
				Value:       "100",
			},
			{
				BlockNumber: 1,
				Hash:        "0x2",
				From:        "0xFrom2",
				To:          "0xTo2",
				Value:       "200",
			},
		}

		err := s.SaveAddressTransactions(t.Context(), address, txs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		savedTxs, err := s.ListAddressTransactions(t.Context(), address)
		if err != nil {
			t.Fatalf("unexpected error listing transactions: %v", err)
		}

		if !reflect.DeepEqual(savedTxs, txs) {
			t.Errorf("expected transactions %v, got %v", txs, savedTxs)
		}
	})

	t.Run("Append Transactions", func(t *testing.T) {
		s := New()
		address := "0xDEF456"
		initialTxs := []blockwatch.Transaction{
			{
				BlockNumber: 2,
				Hash:        "0x3",
				From:        "0xFrom3",
				To:          "0xTo3",
				Value:       "300",
			},
		}

		if err := s.SaveAddressTransactions(t.Context(), address, initialTxs); err != nil {
			t.Fatalf("unexpected error saving initial transactions: %v", err)
		}

		newTxs := []blockwatch.Transaction{
			{
				BlockNumber: 2,
				Hash:        "0x4",
				From:        "0xFrom4",
				To:          "0xTo4",
				Value:       "400",
			},
		}
		if err := s.SaveAddressTransactions(t.Context(), address, newTxs); err != nil {
			t.Fatalf("unexpected error appending transactions: %v", err)
		}

		expectedTxs := append(initialTxs, newTxs...)
		savedTxs, err := s.ListAddressTransactions(t.Context(), address)
		if err != nil {
			t.Fatalf("unexpected error listing transactions: %v", err)
		}
		if !reflect.DeepEqual(savedTxs, expectedTxs) {
			t.Errorf("expected transactions %v, got %v", expectedTxs, savedTxs)
		}
	})

	t.Run("Empty Transactions", func(t *testing.T) {
		s := New()
		address := "0xGHI789"
		if err := s.SaveAddressTransactions(t.Context(), address, []blockwatch.Transaction{}); err != nil {
			t.Fatalf("unexpected error saving empty transactions: %v", err)
		}

		savedTxs, err := s.ListAddressTransactions(t.Context(), address)
		if err != nil {
			t.Fatalf("unexpected error listing transactions: %v", err)
		}
		if len(savedTxs) != 0 {
			t.Errorf("expected 0 transactions, got %d", len(savedTxs))
		}
	})
}

func TestListAddressTransactions(t *testing.T) {
	t.Run("Address Not Saved", func(t *testing.T) {
		s := New()
		address := "0xABC"
		txs, err := s.ListAddressTransactions(t.Context(), address)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(txs) != 0 {
			t.Errorf("expected nil or empty slice, got %v", txs)
		}
	})

	t.Run("Address Saved With No Transactions", func(t *testing.T) {
		s := New()
		address := "0xDEF"

		if err := s.SaveAddressTransactions(t.Context(), address, []blockwatch.Transaction{}); err != nil {
			t.Fatalf("unexpected error when saving empty transactions: %v", err)
		}
		txs, err := s.ListAddressTransactions(t.Context(), address)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(txs) != 0 {
			t.Errorf("expected an empty slice, got %v", txs)
		}
	})

	t.Run("Address Saved With Transactions", func(t *testing.T) {
		s := New()
		address := "0xGHI"
		expectedTxs := []blockwatch.Transaction{
			{
				BlockNumber: 1,
				Hash:        "0x1",
				From:        "0xFrom",
				To:          "0xTo",
				Value:       "10",
			},
			{
				BlockNumber: 1,
				Hash:        "0x2",
				From:        "0xFrom2",
				To:          "0xTo2",
				Value:       "20",
			},
		}

		err := s.SaveAddressTransactions(t.Context(), address, expectedTxs)
		if err != nil {
			t.Fatalf("unexpected error when saving transactions: %v", err)
		}
		txs, err := s.ListAddressTransactions(t.Context(), address)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(txs, expectedTxs) {
			t.Errorf("expected transactions %v, got %v", expectedTxs, txs)
		}
	})
}
