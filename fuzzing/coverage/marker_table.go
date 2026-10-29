package coverage

import "iter"

// markerTable owns its storage for its full lifetime. Lookup and increment share
// one probe sequence; no storage is pooled or recycled after publication.
// Zero is represented separately so all uint64 keys and counts are supported.
type markerTable struct {
	// entries has a power-of-two length; key zero marks an unused slot.
	entries []markerEntry
	// used counts occupied nonzero-key slots, including wrapped zero counts.
	used int
	// zero records the presence of marker zero independently of its hit count.
	zero      bool
	zeroCount uint64
}

// markerEntry stores a complete marker and its exact uint64 hit count.
type markerEntry struct{ key, count uint64 }

// markerHash mixes both halves of a branch marker before masking to a slot.
func markerHash(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	return x ^ (x >> 31)
}

// init lazily allocates the smallest table. Storage is never pooled.
func (m *markerTable) init() {
	if m.entries == nil {
		m.entries = make([]markerEntry, 16)
	}
}

// count includes every recorded key, even if its counter has wrapped to zero.
func (m *markerTable) count() int {
	n := m.used
	if m.zero {
		n++
	}
	return n
}

// get returns the exact count, or zero for an absent marker.
func (m *markerTable) get(key uint64) uint64 {
	if key == 0 {
		return m.zeroCount
	}
	if len(m.entries) == 0 {
		return 0
	}
	mask := uint64(len(m.entries) - 1)
	for i := markerHash(key) & mask; ; i = (i + 1) & mask {
		e := &m.entries[i]
		if e.key == key {
			return e.count
		}
		if e.key == 0 {
			return 0
		}
	}
}

// add increments a marker with uint64 wraparound and reports whether its
// previous count was zero, matching the coverage-increase check of a Go map.
func (m *markerTable) add(key, delta uint64) bool {
	m.init()
	// Keep marker zero separately so an empty slot cannot be confused with it.
	if key == 0 {
		prev := m.zeroCount
		m.zero = true
		m.zeroCount += delta
		return prev == 0
	}
	mask := uint64(len(m.entries) - 1)
	for i := markerHash(key) & mask; ; i = (i + 1) & mask {
		e := &m.entries[i]
		if e.key == key {
			prev := e.count
			e.count += delta
			return prev == 0
		}
		if e.key == 0 {
			// Keep at least one empty slot so unsuccessful probes terminate.
			if m.used >= len(m.entries)*3/4 {
				m.grow()
				return m.add(key, delta)
			}
			*e = markerEntry{key, delta}
			m.used++
			return true
		}
	}
}

// grow doubles the table and rehashes every nonzero key without changing counts.
func (m *markerTable) grow() {
	old := m.entries
	m.entries = make([]markerEntry, len(old)*2)
	mask := uint64(len(m.entries) - 1)
	for _, e := range old {
		if e.key == 0 {
			continue
		}
		i := markerHash(e.key) & mask
		for m.entries[i].key != 0 {
			i = (i + 1) & mask
		}
		m.entries[i] = e
	}
}

// all visits every stored marker, including marker zero and wrapped counts.
func (m *markerTable) all() iter.Seq2[uint64, uint64] {
	return func(yield func(uint64, uint64) bool) {
		if m.zero && !yield(0, m.zeroCount) {
			return
		}
		for _, e := range m.entries {
			if e.key != 0 && !yield(e.key, e.count) {
				return
			}
		}
	}
}
