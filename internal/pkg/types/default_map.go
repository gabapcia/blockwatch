package types

// DefaultMap is a generic map wrapper that returns default values for missing keys.
//
// It is useful when you want to avoid key existence checks and automatically
// initialize map entries with default values provided by a user-defined function.
//
// Example use case:
//
//	m := NewDefaultMap[string, int](func() int { return 0 })
//	count := m.Get("key") // returns 0 if "key" is not yet in the map
type DefaultMap[K comparable, V any] struct {
	data        map[K]V  // underlying map storing the key-value pairs
	defaultFunc func() V // function used to generate default values for missing keys
}

// NewDefaultMap creates a new DefaultMap with a user-defined default function.
//
// Parameters:
//   - defaultFunc: function that produces a default value for the map's value type.
//
// Returns:
//   - A DefaultMap instance with an empty underlying map and the given default function.
func NewDefaultMap[K comparable, V any](defaultFunc func() V) DefaultMap[K, V] {
	return DefaultMap[K, V]{
		data:        make(map[K]V),
		defaultFunc: defaultFunc,
	}
}

// Get retrieves the value associated with the given key.
//
// If the key is not present, it invokes the defaultFunc to generate a default value,
// stores it in the map, and then returns it.
//
// Parameters:
//   - key: the key to retrieve.
//
// Returns:
//   - The value associated with the key, or a default-initialized value if absent.
func (d *DefaultMap[K, V]) Get(key K) V {
	val, ok := d.data[key]
	if ok {
		return val
	}

	val = d.defaultFunc()
	d.data[key] = val
	return val
}

// Set manually assigns a value to the given key in the map.
//
// Parameters:
//   - key: the map key to assign.
//   - val: the value to associate with the key.
func (d *DefaultMap[K, V]) Set(key K, val V) {
	d.data[key] = val
}

// ToMap returns the underlying map used by the DefaultMap.
//
// This allows external access to the internal data for iteration, serialization,
// or bulk operations.
//
// Returns:
//   - A map[K]V containing all key-value pairs in the DefaultMap.
func (d *DefaultMap[K, V]) ToMap() map[K]V {
	return d.data
}
