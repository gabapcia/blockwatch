package blockwatch

import (
	"reflect"
	"testing"
)

func TestMemoryStorage_SaveAddress(t *testing.T) {
	t.Run("New Address", func(t *testing.T) {
		s := NewMemoryStorage()
		address := "0xABC123"

		err := s.SaveAddress(t.Context(), address)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		addresses, err := s.ListAddresses(t.Context())
		if err != nil {
			t.Fatalf("expected no error from ListAddresses, got %v", err)
		}

		found := false
		for _, addr := range addresses {
			if addr == address {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("expected address %q to be saved", address)
		}
	})

	t.Run("Duplicate Address", func(t *testing.T) {
		s := NewMemoryStorage()
		address := "0xABC123"

		err := s.SaveAddress(t.Context(), address)
		if err != nil {
			t.Fatalf("expected no error on first save, got %v", err)
		}

		err = s.SaveAddress(t.Context(), address)
		if err == nil {
			t.Fatal("expected error when saving duplicate address, got nil")
		}

		if err != ErrAddressAlreadySaved {
			t.Errorf("expected error %q, got %q", ErrAddressAlreadySaved, err)
		}
	})
}

func TestMemoryStorage_ListAddresses(t *testing.T) {
	t.Run("EmptyStorage", func(t *testing.T) {
		s := NewMemoryStorage()
		addresses, err := s.ListAddresses(t.Context())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(addresses) != 0 {
			t.Errorf("expected 0 addresses, got %d", len(addresses))
		}
	})

	t.Run("Multiple Addresses", func(t *testing.T) {
		s := NewMemoryStorage()
		expectedAddresses := []string{"0x1", "0x2", "0x3"}
		for _, addr := range expectedAddresses {
			if err := s.SaveAddress(t.Context(), addr); err != nil {
				t.Fatalf("failed to save address %q: %v", addr, err)
			}
		}

		addresses, err := s.ListAddresses(t.Context())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(addresses) != len(expectedAddresses) {
			t.Errorf("expected %d addresses, got %d", len(expectedAddresses), len(addresses))
		}

		for _, expected := range expectedAddresses {
			found := false
			for _, actual := range addresses {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected address %q not found in list", expected)
			}
		}
	})
}

func TestMemoryStorage_SetLatestBlockNumber(t *testing.T) {
	t.Run("Set And Retrieve Block Number", func(t *testing.T) {
		s := NewMemoryStorage()
		expected := int64(100)

		if err := s.SetLatestBlockNumber(t.Context(), expected); err != nil {
			t.Fatalf("SetLatestBlockNumber returned an unexpected error: %v", err)
		}

		actual, err := s.GetLatestBlockNumber(t.Context())
		if err != nil {
			t.Fatalf("GetLatestBlockNumber returned an unexpected error: %v", err)
		}
		if actual != expected {
			t.Errorf("expected block number %d, got %d", expected, actual)
		}
	})

	t.Run("Overwrite Block Number", func(t *testing.T) {
		s := NewMemoryStorage()
		initial := int64(50)
		if err := s.SetLatestBlockNumber(t.Context(), initial); err != nil {
			t.Fatalf("SetLatestBlockNumber returned an unexpected error: %v", err)
		}

		newValue := int64(75)
		if err := s.SetLatestBlockNumber(t.Context(), newValue); err != nil {
			t.Fatalf("SetLatestBlockNumber returned an unexpected error when overwriting: %v", err)
		}

		actual, err := s.GetLatestBlockNumber(t.Context())
		if err != nil {
			t.Fatalf("GetLatestBlockNumber returned an unexpected error: %v", err)
		}
		if actual != newValue {
			t.Errorf("expected block number %d, got %d", newValue, actual)
		}
	})
}

func TestMemoryStorage_GetLatestBlockNumber(t *testing.T) {
	t.Run("Initial Value", func(t *testing.T) {
		s := NewMemoryStorage()
		value, err := s.GetLatestBlockNumber(t.Context())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if value != 0 {
			t.Errorf("expected initial block number 0, got %d", value)
		}
	})

	t.Run("After Setting", func(t *testing.T) {
		s := NewMemoryStorage()
		expected := int64(123)
		if err := s.SetLatestBlockNumber(t.Context(), expected); err != nil {
			t.Fatalf("unexpected error from SetLatestBlockNumber: %v", err)
		}
		value, err := s.GetLatestBlockNumber(t.Context())
		if err != nil {
			t.Fatalf("unexpected error from GetLatestBlockNumber: %v", err)
		}
		if value != expected {
			t.Errorf("expected block number %d, got %d", expected, value)
		}
	})
}

func TestMemoryStorage_SaveAddressTransactions(t *testing.T) {
	t.Run("New Address Transactions", func(t *testing.T) {
		s := NewMemoryStorage()
		address := "0xABC123"
		txs := []Transaction{
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
		s := NewMemoryStorage()
		address := "0xDEF456"
		initialTxs := []Transaction{
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

		newTxs := []Transaction{
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
		s := NewMemoryStorage()
		address := "0xGHI789"
		if err := s.SaveAddressTransactions(t.Context(), address, []Transaction{}); err != nil {
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

func TestMemoryStorage_ListAddressTransactions(t *testing.T) {
	t.Run("Address Not Saved", func(t *testing.T) {
		s := NewMemoryStorage()
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
		s := NewMemoryStorage()
		address := "0xDEF"

		if err := s.SaveAddressTransactions(t.Context(), address, []Transaction{}); err != nil {
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
		s := NewMemoryStorage()
		address := "0xGHI"
		expectedTxs := []Transaction{
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
