package memory

import (
	"testing"

	"github.com/gabapcia/blockwatch"
)

func TestSaveAddress(t *testing.T) {
	t.Run("New Address", func(t *testing.T) {
		s := New()
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
		s := New()
		address := "0xABC123"

		err := s.SaveAddress(t.Context(), address)
		if err != nil {
			t.Fatalf("expected no error on first save, got %v", err)
		}

		err = s.SaveAddress(t.Context(), address)
		if err == nil {
			t.Fatal("expected error when saving duplicate address, got nil")
		}

		if err != blockwatch.ErrAddressAlreadySaved {
			t.Errorf("expected error %q, got %q", blockwatch.ErrAddressAlreadySaved, err)
		}
	})
}

func TestListAddresses(t *testing.T) {
	t.Run("EmptyStorage", func(t *testing.T) {
		s := New()
		addresses, err := s.ListAddresses(t.Context())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(addresses) != 0 {
			t.Errorf("expected 0 addresses, got %d", len(addresses))
		}
	})

	t.Run("Multiple Addresses", func(t *testing.T) {
		s := New()
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
