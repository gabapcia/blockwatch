package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultMap(t *testing.T) {
	t.Run("creates empty map with default function", func(t *testing.T) {
		defaultFunc := func() int { return 42 }
		dm := NewDefaultMap[string](defaultFunc)

		// Check that the underlying map is empty
		assert.Empty(t, dm.data, "expected empty map")

		// Check that default function works
		assert.Equal(t, 42, dm.defaultFunc(), "default function should return 42")
	})

	t.Run("works with different types", func(t *testing.T) {
		// Test with string values
		stringMap := NewDefaultMap[int](func() string { return "default" })
		assert.Equal(t, "default", stringMap.defaultFunc(), "string default function should return 'default'")

		// Test with slice values
		sliceMap := NewDefaultMap[string](func() []int { return []int{1, 2, 3} })
		defaultSlice := sliceMap.defaultFunc()
		require.Len(t, defaultSlice, 3, "default slice should have 3 elements")
		assert.Equal(t, 1, defaultSlice[0], "first element should be 1")
	})
}

func TestDefaultMap_Get(t *testing.T) {
	t.Run("returns existing value", func(t *testing.T) {
		dm := NewDefaultMap[string](func() int { return 0 })
		dm.Set("existing", 100)

		value := dm.Get("existing")
		assert.Equal(t, 100, value, "should return existing value")
	})

	t.Run("returns default value for missing key", func(t *testing.T) {
		dm := NewDefaultMap[string](func() int { return 42 })

		value := dm.Get("missing")
		assert.Equal(t, 42, value, "should return default value for missing key")
	})

	t.Run("stores default value in map after first access", func(t *testing.T) {
		dm := NewDefaultMap[string](func() int { return 99 })

		// First access should create and store the default value
		value1 := dm.Get("new_key")
		assert.Equal(t, 99, value1, "first access should return default value")

		// Check that the value is now stored in the underlying map
		storedValue, exists := dm.data["new_key"]
		require.True(t, exists, "value should be stored in map after first access")
		assert.Equal(t, 99, storedValue, "stored value should match default value")

		// Second access should return the stored value
		value2 := dm.Get("new_key")
		assert.Equal(t, 99, value2, "second access should return cached value")
	})

	t.Run("default function called only once per key", func(t *testing.T) {
		callCount := 0
		dm := NewDefaultMap[string](func() int {
			callCount++
			return callCount * 10
		})

		// First access should call default function
		value1 := dm.Get("test_key")
		assert.Equal(t, 10, value1, "first access should call default function")

		// Second access should not call default function again
		value2 := dm.Get("test_key")
		assert.Equal(t, 10, value2, "second access should return cached value")

		assert.Equal(t, 1, callCount, "default function should be called exactly once per key")
	})

	t.Run("works with complex types", func(t *testing.T) {
		type Person struct {
			Name string
			Age  int
		}

		dm := NewDefaultMap[string](func() Person {
			return Person{Name: "Unknown", Age: 0}
		})

		person := dm.Get("john")
		assert.Equal(t, "Unknown", person.Name, "default person name should be 'Unknown'")
		assert.Equal(t, 0, person.Age, "default person age should be 0")
	})
}

func TestDefaultMap_Set(t *testing.T) {
	t.Run("sets value for new key", func(t *testing.T) {
		dm := NewDefaultMap[string](func() int { return 0 })

		dm.Set("new_key", 123)

		value, exists := dm.data["new_key"]
		require.True(t, exists, "key should exist after Set")
		assert.Equal(t, 123, value, "value should be set correctly")
	})

	t.Run("overwrites existing value", func(t *testing.T) {
		dm := NewDefaultMap[string](func() int { return 0 })
		dm.Set("key", 100)

		dm.Set("key", 200)

		value := dm.Get("key")
		assert.Equal(t, 200, value, "value should be overwritten")
	})

	t.Run("works with different types", func(t *testing.T) {
		dm := NewDefaultMap[int](func() string { return "" })

		dm.Set(1, "first")
		dm.Set(2, "second")

		assert.Equal(t, "first", dm.Get(1), "should set and retrieve string value for key 1")
		assert.Equal(t, "second", dm.Get(2), "should set and retrieve string value for key 2")
	})
}

func TestDefaultMap_ToMap(t *testing.T) {
	t.Run("returns empty map initially", func(t *testing.T) {
		dm := NewDefaultMap[string](func() int { return 0 })

		result := dm.ToMap()
		assert.Empty(t, result, "should return empty map initially")
	})

	t.Run("returns map with set values", func(t *testing.T) {
		dm := NewDefaultMap[string](func() int { return 0 })
		dm.Set("a", 1)
		dm.Set("b", 2)

		result := dm.ToMap()

		require.Len(t, result, 2, "map should contain 2 elements")
		assert.Equal(t, 1, result["a"], "result['a'] should be 1")
		assert.Equal(t, 2, result["b"], "result['b'] should be 2")
	})

	t.Run("includes values created by Get", func(t *testing.T) {
		dm := NewDefaultMap[string](func() int { return 99 })

		// Access a key that doesn't exist, which should create it
		dm.Get("auto_created")
		dm.Set("manual", 50)

		result := dm.ToMap()

		require.Len(t, result, 2, "map should contain 2 elements")
		assert.Equal(t, 99, result["auto_created"], "auto-created value should be 99")
		assert.Equal(t, 50, result["manual"], "manually set value should be 50")
	})

	t.Run("returned map is the same reference as internal data", func(t *testing.T) {
		dm := NewDefaultMap[string](func() int { return 0 })
		dm.Set("test", 42)

		result := dm.ToMap()

		// Modify the returned map
		result["new_key"] = 100

		// Check that the internal data was also modified
		assert.Equal(t, 100, dm.data["new_key"], "internal data should be modified when returned map is modified")
	})
}
