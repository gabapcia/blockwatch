package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Hex represents a hexadecimal-encoded number as a string (e.g., "0x1a").
// It provides validation, JSON marshaling/unmarshaling, and conversion to int64.
type Hex string

// HexFromString validates the input string and returns a Hex value if valid.
func HexFromString(s string) (Hex, error) {
	if err := validateHex(s); err != nil {
		return "", err
	}
	return Hex(s), nil
}

// validateHex checks whether a string is a valid hexadecimal number starting with "0x" or "0X".
func validateHex(s string) error {
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return fmt.Errorf("hex string must start with 0x")
	}

	if _, err := strconv.ParseUint(s[2:], 16, 64); err != nil {
		return fmt.Errorf("invalid hexadecimal value: %w", err)
	}

	return nil
}

// MarshalJSON encodes the Hex as a JSON string.
func (h Hex) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(h))
}

// UnmarshalJSON parses and validates a JSON-encoded hexadecimal string.
func (h *Hex) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("invalid hex string: %w", err)
	}

	if err := validateHex(s); err != nil {
		return err
	}

	*h = Hex(s)
	return nil
}

// Add returns a new Hex representing the result of adding n to the current value.
// If the original value is invalid, it treats it as zero.
func (h Hex) Add(n int64) Hex {
	current := h.Int()
	sum := current + n
	return Hex(fmt.Sprintf("0x%x", sum))
}

// Int returns the decoded int64 value from the hexadecimal string.
// If parsing fails, it returns zero.
func (h Hex) Int() int64 {
	v, _ := strconv.ParseInt(string(h)[2:], 16, 64)
	return v
}
