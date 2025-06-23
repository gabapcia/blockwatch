package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHexFromString(t *testing.T) {
	t.Run("valid lowercase hex", func(t *testing.T) {
		h, err := HexFromString("0x1a")
		require.NoError(t, err)
		assert.Equal(t, Hex("0x1a"), h)
	})

	t.Run("valid uppercase hex", func(t *testing.T) {
		h, err := HexFromString("0XFF")
		require.NoError(t, err)
		assert.Equal(t, Hex("0XFF"), h)
	})

	t.Run("invalid: missing 0x prefix", func(t *testing.T) {
		h, err := HexFromString("1a")
		assert.Error(t, err)
		assert.Equal(t, Hex(""), h)
	})

	t.Run("invalid: only 0x", func(t *testing.T) {
		h, err := HexFromString("0x")
		assert.Error(t, err)
		assert.Equal(t, Hex(""), h)
	})

	t.Run("invalid: bad hex characters", func(t *testing.T) {
		h, err := HexFromString("0xZZZ")
		assert.Error(t, err)
		assert.Equal(t, Hex(""), h)
	})
}

func TestValidateHex(t *testing.T) {
	t.Run("valid lowercase hex", func(t *testing.T) {
		err := validateHex("0x1a")
		require.NoError(t, err)
	})

	t.Run("valid uppercase hex", func(t *testing.T) {
		err := validateHex("0XFF")
		require.NoError(t, err)
	})

	t.Run("only 0x prefix", func(t *testing.T) {
		err := validateHex("0x")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid hexadecimal value")
	})

	t.Run("missing 0x prefix", func(t *testing.T) {
		err := validateHex("123abc")
		assert.Error(t, err)
		assert.EqualError(t, err, "hex string must start with 0x")
	})

	t.Run("invalid characters", func(t *testing.T) {
		err := validateHex("0xGHIJK")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid hexadecimal value")
	})
}

func TestHex_MarshalJSON(t *testing.T) {
	t.Run("valid hex marshals to quoted JSON string", func(t *testing.T) {
		h := Hex("0x2a")

		data, err := json.Marshal(h)
		require.NoError(t, err)
		assert.Equal(t, `"0x2a"`, string(data))
	})

	t.Run("empty hex marshals to empty quoted string", func(t *testing.T) {
		h := Hex("")

		data, err := json.Marshal(h)
		require.NoError(t, err)
		assert.Equal(t, `""`, string(data))
	})

	t.Run("uppercase hex remains unchanged", func(t *testing.T) {
		h := Hex("0XFF")

		data, err := json.Marshal(h)
		require.NoError(t, err)
		assert.Equal(t, `"0XFF"`, string(data))
	})
}

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
