package blockwatch

import "testing"

func TestSubscribe(t *testing.T) {
	monitor, err := New(t.Context(), NewBlockchainMock(0, nil))
	if err != nil {
		t.Fatalf("unexpected error creating monitor: %v", err)
	}

	t.Run("Subscribe New Address", func(t *testing.T) {
		address := "0x123"
		result := monitor.Subscribe(address)
		if !result {
			t.Errorf("expected Subscribe(%q) to return true for a new address, got false", address)
		}

		if _, exists := monitor.transactiosByAddress[address]; !exists {
			t.Errorf("address %q was not added to transactiosByAddress", address)
		}
	})

	t.Run("Subscribe Duplicate Address", func(t *testing.T) {
		address := "0xABC"
		if !monitor.Subscribe(address) {
			t.Fatalf("expected first subscription of %q to succeed", address)
		}

		result := monitor.Subscribe(address)
		if result {
			t.Errorf("expected Subscribe(%q) to return false for a duplicate address, got true", address)
		}
	})
}
