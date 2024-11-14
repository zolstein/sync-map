//go:build !go1.23

package sync_map_test

import "math/rand"

// mapInterface is the interface Map implements.
type mapInterface interface {
	Load(key any) (value any, ok bool)
	Store(key, value any)
	LoadOrStore(key, value any) (actual any, loaded bool)
	LoadAndDelete(key any) (value any, loaded bool)
	Delete(any)
	Swap(key, value any) (previous any, loaded bool)
	Range(func(key, value any) (shouldContinue bool))
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
}

type casMapInterfaceInt interface {
	mapInterfaceInt
	CompareAndSwap(key, old, new int) (swapped bool)
	CompareAndDelete(key, old int) (deleted bool)
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
	default:
		panic("invalid mapOp")
	}
}
