package types

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSet(t *testing.T) {
	t.Run("empty set", func(t *testing.T) {
		set := NewSet[int]()
		assert.Empty(t, set)
	})

	t.Run("single element", func(t *testing.T) {
		set := NewSet(42)
		assert.Len(t, set, 1)
		assert.Contains(t, set, 42)
	})

	t.Run("multiple elements", func(t *testing.T) {
		set := NewSet(1, 2, 3, 4, 5)
		assert.Len(t, set, 5)
		for i := 1; i <= 5; i++ {
			assert.Contains(t, set, i)
		}
	})

	t.Run("duplicate elements", func(t *testing.T) {
		set := NewSet(1, 2, 2, 3, 3, 3)
		assert.Len(t, set, 3)
		for i := 1; i <= 3; i++ {
			assert.Contains(t, set, i)
		}
	})

	t.Run("string elements", func(t *testing.T) {
		set := NewSet("hello", "world", "test")
		assert.Len(t, set, 3)
		expected := []string{"hello", "world", "test"}
		for _, str := range expected {
			assert.Contains(t, set, str)
		}
	})
}

func TestSet_Add(t *testing.T) {
	t.Run("add to empty set", func(t *testing.T) {
		set := NewSet[int]()
		set.Add(42)

		assert.Len(t, set, 1)
		assert.Contains(t, set, 42)
	})

	t.Run("add multiple elements", func(t *testing.T) {
		set := NewSet[int]()
		set.Add(1, 2, 3)

		assert.Len(t, set, 3)
		for i := 1; i <= 3; i++ {
			assert.Contains(t, set, i)
		}
	})

	t.Run("add to existing set", func(t *testing.T) {
		set := NewSet(1, 2, 3)
		set.Add(4, 5)

		assert.Len(t, set, 5)
		for i := 1; i <= 5; i++ {
			assert.Contains(t, set, i)
		}
	})

	t.Run("add duplicate elements", func(t *testing.T) {
		set := NewSet(1, 2, 3)
		set.Add(2, 3, 4)

		assert.Len(t, set, 4)
		for i := 1; i <= 4; i++ {
			assert.Contains(t, set, i)
		}
	})

	t.Run("add no elements", func(t *testing.T) {
		set := NewSet(1, 2, 3)
		originalLen := len(set)
		set.Add()

		assert.Len(t, set, originalLen)
	})
}

func TestSet_Delete(t *testing.T) {
	t.Run("delete from empty set", func(t *testing.T) {
		set := NewSet[int]()
		set.Delete(42)

		assert.Empty(t, set)
	})

	t.Run("delete existing element", func(t *testing.T) {
		set := NewSet(1, 2, 3, 4, 5)
		set.Delete(3)

		assert.Len(t, set, 4)
		assert.NotContains(t, set, 3)
		// Check remaining elements
		for _, i := range []int{1, 2, 4, 5} {
			assert.Contains(t, set, i)
		}
	})

	t.Run("delete multiple elements", func(t *testing.T) {
		set := NewSet(1, 2, 3, 4, 5)
		set.Delete(2, 4)

		assert.Len(t, set, 3)
		for _, i := range []int{2, 4} {
			assert.NotContains(t, set, i)
		}
		// Check remaining elements
		for _, i := range []int{1, 3, 5} {
			assert.Contains(t, set, i)
		}
	})

	t.Run("delete non-existing element", func(t *testing.T) {
		set := NewSet(1, 2, 3)
		originalLen := len(set)
		set.Delete(99)

		assert.Len(t, set, originalLen)
		// Check all original elements still exist
		for i := 1; i <= 3; i++ {
			assert.Contains(t, set, i)
		}
	})

	t.Run("delete no elements", func(t *testing.T) {
		set := NewSet(1, 2, 3)
		originalLen := len(set)
		set.Delete()

		assert.Len(t, set, originalLen)
	})
}

func TestSet_ToIter(t *testing.T) {
	t.Run("empty set", func(t *testing.T) {
		set := NewSet[int]()
		iter := set.ToIter()

		count := 0
		for range iter {
			count++
		}

		assert.Equal(t, 0, count)
	})

	t.Run("non-empty set", func(t *testing.T) {
		expected := []int{1, 2, 3, 4, 5}
		set := NewSet(expected...)
		iter := set.ToIter()

		var collected []int
		for val := range iter {
			collected = append(collected, val)
		}

		require.Len(t, collected, len(expected))

		// Sort both slices for comparison since set iteration order is not guaranteed
		slices.Sort(collected)
		slices.Sort(expected)

		assert.Equal(t, expected, collected)
	})

	t.Run("string set", func(t *testing.T) {
		expected := []string{"apple", "banana", "cherry"}
		set := NewSet(expected...)
		iter := set.ToIter()

		var collected []string
		for val := range iter {
			collected = append(collected, val)
		}

		require.Len(t, collected, len(expected))

		// Sort both slices for comparison
		slices.Sort(collected)
		slices.Sort(expected)

		assert.Equal(t, expected, collected)
	})
}

func TestSet_ToSlice(t *testing.T) {
	t.Run("empty set", func(t *testing.T) {
		set := NewSet[int]()
		slice := set.ToSlice()

		assert.Empty(t, slice)
	})

	t.Run("non-empty set", func(t *testing.T) {
		expected := []int{1, 2, 3, 4, 5}
		set := NewSet(expected...)
		slice := set.ToSlice()

		require.Len(t, slice, len(expected))

		// Sort both slices for comparison since set order is not guaranteed
		slices.Sort(slice)
		slices.Sort(expected)

		assert.Equal(t, expected, slice)
	})

	t.Run("string set", func(t *testing.T) {
		expected := []string{"apple", "banana", "cherry"}
		set := NewSet(expected...)
		slice := set.ToSlice()

		require.Len(t, slice, len(expected))

		// Sort both slices for comparison
		slices.Sort(slice)
		slices.Sort(expected)

		assert.Equal(t, expected, slice)
	})

	t.Run("slice independence", func(t *testing.T) {
		set := NewSet(1, 2, 3)
		slice := set.ToSlice()

		// Modify the slice
		slice[0] = 999

		// Verify the set is unchanged
		assert.NotContains(t, set, 999)
		assert.Contains(t, set, 1)
	})
}
