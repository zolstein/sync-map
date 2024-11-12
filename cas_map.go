package sync_map

import (
	"sync/atomic"
	"unsafe"
)

// CasMap is a Map which supports the operations CompareAndSwap and CompareAndDelete.
// These methods require V to be comparable, so they must be defined on a type with this
// restriction.
//
// In the terminology of [the Go memory model], Map arranges that a write operation
// “synchronizes before” any read operation that observes the effect of the write, where
// read and write operations are defined as follows.
// [Map.Load], [Map.LoadAndDelete], [Map.LoadOrStore], and [Map.Swap] are read operations;
// [Map.Delete], [Map.LoadAndDelete], [Map.Store], and [Map.Swap] are write operations;
// [Map.LoadOrStore] is a write operation when it returns loaded set to false;
// [Map.CompareAndSwap] is a write operation when it returns swapped set to true;
// and [Map.CompareAndDelete] is a write operation when it returns deleted set to true.
//
// [the Go memory model]: https://go.dev/ref/mem
type CasMap[K comparable, V comparable] struct {
	Map[K, V]
}

// CompareAndSwap swaps the old and new values for key
// if the value stored in the map is equal to old.
// The old value must be of a comparable type.
func (m *CasMap[K, V]) CompareAndSwap(key K, old, new V) (swapped bool) {
	read := m.loadReadOnly()
	if e, ok := read.m[key]; ok {
		return (*casEntry[V])(e).tryCompareAndSwap(old, new)
	} else if !read.amended {
		return false // No existing value for key.
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	read = m.loadReadOnly()
	swapped = false
	if e, ok := read.m[key]; ok {
		swapped = (*casEntry[V])(e).tryCompareAndSwap(old, new)
	} else if e, ok := m.dirty[key]; ok {
		swapped = (*casEntry[V])(e).tryCompareAndSwap(old, new)
		// We needed to lock mu in order to load the entry for key,
		// and the operation didn't change the set of keys in the map
		// (so it would be made more efficient by promoting the dirty
		// map to read-only).
		// Count it as a miss so that we will eventually switch to the
		// more efficient steady state.
		m.missLocked()
	}
	return swapped
}

// CompareAndDelete deletes the entry for key if its value is equal to old.
// The old value must be of a comparable type.
//
// If there is no current value for key in the map, CompareAndDelete
// returns false (even if the old value is the zero value of V).
func (m *CasMap[K, V]) CompareAndDelete(key K, old V) (deleted bool) {
	read := m.loadReadOnly()
	e, ok := read.m[key]
	if !ok && read.amended {
		m.mu.Lock()
		read = m.loadReadOnly()
		e, ok = read.m[key]
		if !ok && read.amended {
			e, ok = m.dirty[key]
			// Don't delete key from m.dirty: we still need to do the “compare” part
			// of the operation. The entry will eventually be expunged when the
			// dirty map is promoted to the read map.
			//
			// Regardless of whether the entry was present, record a miss: this key
			// will take the slow path until the dirty map is promoted to the read
			// map.
			m.missLocked()
		}
		m.mu.Unlock()
	}
	for ok {
		ptr := atomic.LoadPointer(&e.p)
		if ptr == nil || ptr == expunged {
			return false
		}
		p := (*V)(ptr)
		if *p != old {
			return false
		}
		if atomic.CompareAndSwapPointer(&e.p, ptr, nil) {
			return true
		}
	}
	return false
}

type casEntry[V comparable] entry[V]

// tryCompareAndSwap compare the entry with the given old value and swaps
// it with a new value if the entry is equal to the old value, and the entry
// has not been expunged.
//
// If the entry is expunged, tryCompareAndSwap returns false and leaves
// the entry unchanged.
func (e *casEntry[V]) tryCompareAndSwap(old, new V) bool {
	ptr := atomic.LoadPointer(&e.p)
	if ptr == nil || ptr == expunged {
		return false
	}
	p := (*V)(ptr)
	if *p != old {
		return false
	}

	// Copy the interface after the first load to make this method more amenable
	// to escape analysis: if the comparison fails from the start, we shouldn't
	// bother heap-allocating an interface value to store.
	nc := new
	for {
		if atomic.CompareAndSwapPointer(&e.p, ptr, unsafe.Pointer(&nc)) {
			return true
		}
		ptr = atomic.LoadPointer(&e.p)
		if ptr == nil || ptr == expunged {
			return false
		}
		p = (*V)(ptr)
		if *p != old {
			return false
		}
	}
}
