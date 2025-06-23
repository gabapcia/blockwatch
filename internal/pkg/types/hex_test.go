package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHex_UnmarshalJSON(t *testing.T) {
	t.Run("valid lowercase hex", func(t *testing.T) {
		input := `"0x1a"`
		var h Hex

		err := json.Unmarshal([]byte(input), &h)
		require.NoError(t, err)
		assert.Equal(t, Hex("0x1a"), h)
	})

	t.Run("valid uppercase hex", func(t *testing.T) {
		input := `"0X2F"`
		var h Hex

		err := json.Unmarshal([]byte(input), &h)
		require.NoError(t, err)
		assert.Equal(t, Hex("0X2F"), h)
	})

	t.Run("missing 0x prefix", func(t *testing.T) {
		input := `"1a"`
		var h Hex

		err := json.Unmarshal([]byte(input), &h)
		require.Error(t, err)
	})

	t.Run("invalid hex characters", func(t *testing.T) {
		input := `"0xZZZ"`
		var h Hex

		err := json.Unmarshal([]byte(input), &h)
		require.Error(t, err)
	})

	t.Run("not a string", func(t *testing.T) {
		input := `42`
		var h Hex

		err := json.Unmarshal([]byte(input), &h)
		require.Error(t, err)
	})
}

func TestHex_Int(t *testing.T) {
	t.Run("0x0a should be 10", func(t *testing.T) {
		var h Hex = "0x0a"
		assert.Equal(t, int64(10), h.Int())
	})

	t.Run("0xff should be 255", func(t *testing.T) {
		var h Hex = "0xff"
		assert.Equal(t, int64(255), h.Int())
	})

	t.Run("0X10 should be 16", func(t *testing.T) {
		var h Hex = "0X10"
		assert.Equal(t, int64(16), h.Int())
	})

	t.Run("invalid hex returns 0", func(t *testing.T) {
		var h Hex = "0xZZZ"
		assert.Equal(t, int64(0), h.Int())
	})
}
