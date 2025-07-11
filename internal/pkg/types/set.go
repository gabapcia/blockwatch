package types

import (
	"iter"
	"maps"
	"slices"
)

// Set is a generic hash set implementation for comparable types.
//
// It provides efficient membership tests, insertion, and deletion
// using a map[T]struct{} internally. This type is mutable:
// methods like Add and Delete modify the set in place.
type Set[T comparable] map[T]struct{}

// NewSet creates a new Set and optionally inserts the provided elements.
//
// Parameters:
//   - data: zero or more elements to initialize the set with.
//
// Returns:
//   - A Set containing the provided elements.
func NewSet[T comparable](data ...T) Set[T] {
	set := make(Set[T])
	for _, d := range data {
		set[d] = struct{}{}
	}
	return set
}

// Add inserts one or more elements into the set.
//
// This method modifies the Set in place.
//
// Parameters:
//   - values: elements to add to the set.
func (s Set[T]) Add(values ...T) {
	for _, val := range values {
		s[val] = struct{}{}
	}
}

// Delete removes one or more elements from the set.
//
// This method modifies the Set in place.
//
// Parameters:
//   - values: elements to remove from the set.
func (s Set[T]) Delete(values ...T) {
	for _, val := range values {
		delete(s, val)
	}
}

// ToIter returns an iterator over all elements in the set.
//
// Useful for use with the "iter" package's range-based iteration.
//
// Returns:
//   - A Seq[T] iterator that yields all elements in the set.
func (s Set[T]) ToIter() iter.Seq[T] {
	return maps.Keys(s)
}

// ToSlice returns a slice containing all elements in the set.
//
// The order of elements is not guaranteed.
//
// Returns:
//   - A slice of all elements in the set.
func (s Set[T]) ToSlice() []T {
	return slices.Collect(s.ToIter())
}
