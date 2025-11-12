//go:build go1.23

package sync_map_test

import (
	"github.com/zolstein/sync-map"
	"math/rand"
	"sync"
	"testing"
)

// mapInterface is the interface Map implements.
type mapInterface interface {
	Load(key any) (value any, ok bool)
	Store(key, value any)
	LoadOrStore(key, value any) (actual any, loaded bool)
	LoadAndDelete(key any) (value any, loaded bool)
	Delete(any)
	Swap(key, value any) (previous any, loaded bool)
	Range(func(key, value any) (shouldContinue bool))
	Clear()
}

type casMapInterface interface {
	mapInterface
	CompareAndSwap(key, old, new any) (swapped bool)
	CompareAndDelete(key, old any) (deleted bool)
}

// mapInterface is the interface Map implements.
type mapInterfaceInt interface {
	Load(key int) (value int, ok bool)
	Store(key, value int)
	LoadOrStore(key, value int) (actual int, loaded bool)
	LoadAndDelete(key int) (value int, loaded bool)
	Delete(int)
	Swap(key, value int) (previous int, loaded bool)
	Range(func(key, value int) (shouldContinue bool))
	Clear()
}

type casMapInterfaceInt interface {
	mapInterfaceInt
	CompareAndSwap(key, old, new int) (swapped bool)
	CompareAndDelete(key, old int) (deleted bool)
}

func (m *MapIntWrapper) Clear() {
	m.m.Clear()
}

var mapOps = [...]mapOp{
	opLoad,
	opStore,
	opLoadOrStore,
	opLoadAndDelete,
	opDelete,
	opSwap,
	opCompareAndSwap,
	opCompareAndDelete,
	opClear,
}

func (c mapCall) apply(m casMapInterface) (any, bool) {
	switch c.op {
	case opLoad:
		return m.Load(c.k)
	case opStore:
		m.Store(c.k, c.v)
		return nil, false
	case opLoadOrStore:
		return m.LoadOrStore(c.k, c.v)
	case opLoadAndDelete:
		return m.LoadAndDelete(c.k)
	case opDelete:
		m.Delete(c.k)
		return nil, false
	case opSwap:
		return m.Swap(c.k, c.v)
	case opCompareAndSwap:
		if m.CompareAndSwap(c.k, c.v, rand.Int()) {
			m.Delete(c.k)
			return c.v, true
		}
		return nil, false
	case opCompareAndDelete:
		if m.CompareAndDelete(c.k, c.v) {
			if _, ok := m.Load(c.k); !ok {
				return nil, true
			}
		}
		return nil, false
	case opClear:
		m.Clear()
		return nil, false
	default:
		panic("invalid mapOp")
	}
}

// TestConcurrentClear tests concurrent behavior of sync_map.Map properties to ensure no data races.
// Checks for proper synchronization between Clear, Store, Load operations.
func TestConcurrentClear(t *testing.T) {
	var m sync_map.Map[int, int]

	wg := sync.WaitGroup{}
	wg.Add(30) // 10 goroutines for writing, 10 goroutines for reading, 10 goroutines for waiting

	// Writing data to the map concurrently
	for i := 0; i < 10; i++ {
		go func(k, v int) {
			defer wg.Done()
			m.Store(k, v)
		}(i, i*10)
	}

	// Reading data from the map concurrently
	for i := 0; i < 10; i++ {
		go func(k int) {
			defer wg.Done()
			if value, ok := m.Load(k); ok {
				t.Logf("Key: %v, Value: %v\n", k, value)
			} else {
				t.Logf("Key: %v not found\n", k)
			}
		}(i)
	}

	// Clearing data from the map concurrently
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			m.Clear()
		}()
	}

	wg.Wait()

	m.Clear()

	m.Range(func(k, v int) bool {
		t.Errorf("after Clear, Map contains (%v, %v); expected to be empty", k, v)

		return true
	})
}

func TestMapClearNoAllocations(t *testing.T) {
	var m sync_map.Map[any, any]
	allocs := testing.AllocsPerRun(10, func() {
		m.Clear()
	})
	if allocs > 0 {
		t.Errorf("AllocsPerRun of m.Clear = %v; want 0", allocs)
	}
}
