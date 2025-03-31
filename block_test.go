package blockwatch

import "testing"

func TestGetCurrentBlock(t *testing.T) {
	var (
		expected int64 = 100
		monitor        = &Monitor{currentBlockNumber: expected}
	)

	got := monitor.GetCurrentBlock()
	if got != expected {
		t.Errorf("expected current block %d, got %d", expected, got)
	}
}
