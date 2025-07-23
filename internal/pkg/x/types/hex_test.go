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

func TestHexFromInt(t *testing.T) {
	t.Run("zero", func(t *testing.T) {
		result := HexFromInt(0)
		assert.Equal(t, Hex("0x0"), result)
	})

	t.Run("positive single digit", func(t *testing.T) {
		result := HexFromInt(5)
		assert.Equal(t, Hex("0x5"), result)
	})

	t.Run("positive double digit", func(t *testing.T) {
		result := HexFromInt(15)
		assert.Equal(t, Hex("0xf"), result)
	})

	t.Run("positive large number", func(t *testing.T) {
		result := HexFromInt(255)
		assert.Equal(t, Hex("0xff"), result)
	})

	t.Run("positive very large number", func(t *testing.T) {
		result := HexFromInt(65535)
		assert.Equal(t, Hex("0xffff"), result)
	})

	t.Run("negative small number", func(t *testing.T) {
		result := HexFromInt(-1)
		assert.Equal(t, Hex("0x-1"), result)
	})

	t.Run("negative large number", func(t *testing.T) {
		result := HexFromInt(-255)
		assert.Equal(t, Hex("0x-ff"), result)
	})

	t.Run("power of 2", func(t *testing.T) {
		result := HexFromInt(1024)
		assert.Equal(t, Hex("0x400"), result)
	})

	t.Run("decimal 256", func(t *testing.T) {
		result := HexFromInt(256)
		assert.Equal(t, Hex("0x100"), result)
	})

	t.Run("round trip conversion preserves value", func(t *testing.T) {
		testValues := []int64{0, 1, -1, 42, -42, 255, -255, 1024, -1024}

		for _, original := range testValues {
			hex := HexFromInt(original)
			converted := hex.Int()
			assert.Equal(t, original, converted, "Round trip failed for value %d", original)
		}
	})

	t.Run("produces valid hex format for positive numbers", func(t *testing.T) {
		testValues := []int64{0, 1, 255, 1024}

		for _, value := range testValues {
			hex := HexFromInt(value)

			// Test that the result can be parsed back using HexFromString
			_, err := HexFromString(string(hex))
			assert.NoError(t, err, "HexFromInt(%d) produced invalid hex string %s", value, string(hex))
		}
	})

	t.Run("negative numbers produce hex with minus sign", func(t *testing.T) {
		testValues := []int64{-1, -255, -1024}

		for _, value := range testValues {
			hex := HexFromInt(value)
			hexStr := string(hex)

			// Negative numbers should start with "0x-"
			assert.True(t, len(hexStr) > 3, "Hex string should be longer than 3 characters")
			assert.Equal(t, "0x-", hexStr[:3], "Negative hex should start with '0x-'")
		}
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

func TestHex_Add(t *testing.T) {
	t.Run("add to valid hex", func(t *testing.T) {
		h := Hex("0x0a") // 10
		result := h.Add(5)
		assert.Equal(t, Hex("0xf"), result) // 15
	})

	t.Run("add zero", func(t *testing.T) {
		h := Hex("0x1f") // 31
		result := h.Add(0)
		assert.Equal(t, Hex("0x1f"), result)
	})

	t.Run("add negative", func(t *testing.T) {
		h := Hex("0x0a") // 10
		result := h.Add(-3)
		assert.Equal(t, Hex("0x7"), result) // 7
	})

	t.Run("add to invalid hex should treat as 0", func(t *testing.T) {
		h := Hex("0xZZ")
		result := h.Add(8)
		assert.Equal(t, Hex("0x8"), result)
	})

	t.Run("add resulting in zero", func(t *testing.T) {
		h := Hex("0x05")
		result := h.Add(-5)
		assert.Equal(t, Hex("0x0"), result)
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

func TestHex_IsEmpty(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		assert.True(t, Hex("").IsEmpty(), "IsEmpty should return true for empty string")
	})

	t.Run("whitespace only", func(t *testing.T) {
		assert.True(t, Hex("  \n\t ").IsEmpty(), "IsEmpty should return true for whitespace-only string")
	})

	t.Run("valid hex", func(t *testing.T) {
		assert.False(t, Hex("0x1a").IsEmpty(), "IsEmpty should return false for a valid hex value")
	})

	t.Run("hex with surrounding whitespace", func(t *testing.T) {
		assert.False(t, Hex("  0x1a  ").IsEmpty(), "IsEmpty should return false even if hex has surrounding whitespace")
	})

	t.Run("non-hex non-empty", func(t *testing.T) {
		assert.False(t, Hex(" abc ").IsEmpty(), "IsEmpty should return false for a non-empty non-hex string")
	})
}

func TestHex_String(t *testing.T) {
	t.Run("valid lowercase hex", func(t *testing.T) {
		h := Hex("0x1a")
		result := h.String()
		assert.Equal(t, "0x1a", result)
	})

	t.Run("valid uppercase hex", func(t *testing.T) {
		h := Hex("0XFF")
		result := h.String()
		assert.Equal(t, "0XFF", result)
	})

	t.Run("zero value", func(t *testing.T) {
		h := Hex("0x0")
		result := h.String()
		assert.Equal(t, "0x0", result)
	})

	t.Run("large hex value", func(t *testing.T) {
		h := Hex("0xdeadbeef")
		result := h.String()
		assert.Equal(t, "0xdeadbeef", result)
	})

	t.Run("empty hex", func(t *testing.T) {
		h := Hex("")
		result := h.String()
		assert.Equal(t, "", result)
	})

	t.Run("hex from int conversion", func(t *testing.T) {
		h := HexFromInt(255)
		result := h.String()
		assert.Equal(t, "0xff", result)
	})

	t.Run("hex from int zero", func(t *testing.T) {
		h := HexFromInt(0)
		result := h.String()
		assert.Equal(t, "0x0", result)
	})

	t.Run("hex from negative int", func(t *testing.T) {
		h := HexFromInt(-1)
		result := h.String()
		assert.Equal(t, "0x-1", result)
	})

	t.Run("implements fmt.Stringer interface", func(t *testing.T) {
		h := Hex("0x42")

		// Test that String() method implements fmt.Stringer interface
		var stringer interface{} = h
		_, ok := stringer.(interface{ String() string })
		assert.True(t, ok, "Hex should implement fmt.Stringer interface")
	})

	t.Run("preserves original format", func(t *testing.T) {
		testCases := []string{
			"0x1a",
			"0X1A",
			"0xdeadbeef",
			"0XDEADBEEF",
			"0x0",
			"0X0",
		}

		for _, original := range testCases {
			h := Hex(original)
			result := h.String()
			assert.Equal(t, original, result, "String() should preserve original format")
		}
	})
}
