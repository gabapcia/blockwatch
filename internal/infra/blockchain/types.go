package blockchain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// HexNumber represents a hexadecimal-encoded number as a string (e.g., "0x1a").
// It implements JSON unmarshaling with validation and provides conversion to int64.
type HexNumber string

// UnmarshalJSON parses and validates a JSON-encoded hexadecimal string.
// It ensures the string starts with "0x" or "0X" and contains a valid hex number.
func (h *HexNumber) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("invalid hex string: %w", err)
	}

	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return fmt.Errorf("hex string must start with 0x")
	}

	if _, err := strconv.ParseInt(s[2:], 16, 64); err != nil {
		return fmt.Errorf("invalid hexadecimal value: %w", err)
	}

	*h = HexNumber(s)
	return nil
}

// Int returns the decoded int64 value from the hexadecimal string.
// If parsing fails, it returns zero.
func (h HexNumber) Int() int64 {
	v, _ := strconv.ParseInt(string(h)[2:], 16, 64)
	return v
}
