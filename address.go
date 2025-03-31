package blockwatch

// Subscribe adds the specified address to the Monitor's list of subscribed addresses.
// If the address is not already subscribed, it creates a new entry (with a nil transaction slice)
// and returns true, indicating that the subscription was successful.
// If the address is already present in the subscription list, it returns false.
func (m *Monitor) Subscribe(address string) bool {
	if _, ok := m.transactiosByAddress[address]; ok {
		return false
	}

	m.transactiosByAddress[address] = nil
	return true
}
