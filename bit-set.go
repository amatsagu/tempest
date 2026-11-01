package tempest

// Represents combination of discord bit settings, flags, permissions, etc.
// For example all the user badges or member permissions.
type BitSet uint64

// Combines multiple bits together, producing a new bit set.
func (f BitSet) Add(bits ...BitSet) BitSet {
	for _, bit := range bits {
		f |= bit
	}
	return f
}

// Remove allows you to subtract multiple bits from the first bit set, producing a new bit set.
func (f BitSet) Remove(bits ...BitSet) BitSet {
	for _, bit := range bits {
		f &^= bit
	}
	return f
}

// Ensures that the bit set includes all the specified bits.
func (f BitSet) Has(bits ...BitSet) bool {
	for _, bit := range bits {
		if (f & bit) != bit {
			return false
		}
	}
	return true
}

// Checks whether the bit set is missing any one of the specified bits.
func (f BitSet) Missing(bits ...BitSet) bool {
	for _, bit := range bits {
		if (f & bit) != bit {
			return true
		}
	}
	return false
}
