package tempest

// Represents combination of discord bit settings, flags, permissions, etc.
// For example all the user badges or member permissions.
type BitSet uint64

// Add allows you to add multiple bits together, producing a new bit set.
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

// Has will ensure that the bit set includes all the bits entered.
func (f BitSet) Has(bits ...BitSet) bool {
	for _, bit := range bits {
		if (f & bit) != bit {
			return false
		}
	}
	return true
}

// Missing will check whether the bit set is missing any one of the bits.
func (f BitSet) Missing(bits ...BitSet) bool {
	for _, bit := range bits {
		if (f & bit) != bit {
			return true
		}
	}
	return false
}
