package memory

import "testing"

func TestSetLatestBlockNumber(t *testing.T) {
	t.Run("Set And Retrieve Block Number", func(t *testing.T) {
		s := New()
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
		s := New()
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

func TestGetLatestBlockNumber(t *testing.T) {
	t.Run("Initial Value", func(t *testing.T) {
		s := New()
		value, err := s.GetLatestBlockNumber(t.Context())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if value != 0 {
			t.Errorf("expected initial block number 0, got %d", value)
		}
	})

	t.Run("After Setting", func(t *testing.T) {
		s := New()
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
