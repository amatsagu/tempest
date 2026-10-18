package discord

import "testing"

func TestBitSet(t *testing.T) {
	const (
		BitA BitSet = 1 << iota
		BitB
		BitC
	)

	b := BitA.Add(BitB)
	if !b.Has(BitA, BitB) {
		t.Errorf("expected b to have BitA and BitB")
	}

	if b.Has(BitC) {
		t.Errorf("expected b not to have BitC")
	}

	if !b.Missing(BitC) {
		t.Errorf("expected b to be missing BitC")
	}

	b = b.Remove(BitA)
	if b.Has(BitA) {
		t.Errorf("expected b not to have BitA after removal")
	}

	if !b.Has(BitB) {
		t.Errorf("expected b to still have BitB")
	}
}

func BenchmarkBitSet_Operations(b *testing.B) {
	const (
		BitA BitSet = 1 << 0
		BitB BitSet = 1 << 1
		BitC BitSet = 1 << 2
	)

	b.ReportAllocs()
	for b.Loop() {
		set := BitA.Add(BitB, BitC)
		if !set.Has(BitA, BitC) {
			b.Fatal("unexpected missing bit")
		}
		set = set.Remove(BitB)
		if set.Has(BitB) {
			b.Fatal("unexpected bit present")
		}
	}
}
