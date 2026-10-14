package discord

// https://docs.discord.com/developers/reference#bitwise-permission-flags
type BitSet uint64

func (f BitSet) Add(bits ...BitSet) BitSet {
	for _, bit := range bits {
		f |= bit
	}
	return f
}

func (f BitSet) Remove(bits ...BitSet) BitSet {
	for _, bit := range bits {
		f &^= bit
	}
	return f
}

func (f BitSet) Has(bits ...BitSet) bool {
	for _, bit := range bits {
		if (f & bit) != bit {
			return false
		}
	}
	return true
}

func (f BitSet) Missing(bits ...BitSet) bool {
	return !f.Has(bits...)
}
