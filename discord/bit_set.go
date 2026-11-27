package discord

// BitSet represents a combination of Discord bit settings, flags, permissions, etc.
// For example all the user badges or member permissions.
//
// https://docs.discord.com/developers/topics/permissions#permissions-bitwise-permission-flags
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
	return !f.Has(bits...)
}
