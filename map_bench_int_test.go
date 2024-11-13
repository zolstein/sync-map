// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sync_map_test

import (
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	sync_map "github.com/zolstein/sync-map"
)

type benchInt struct {
	setup func(*testing.B, mapInterfaceInt)
	perG  func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt)
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

type MapIntWrapper struct {
	m sync.Map
}

func (m *MapIntWrapper) Load(key int) (value int, ok bool) {
	v, ok := m.m.Load(key)
	if v == nil {
		return 0, ok
	}
	return v.(int), ok
}

func (m *MapIntWrapper) Store(key, value int) {
	m.m.Store(key, value)
}

func (m *MapIntWrapper) LoadOrStore(key, value int) (actual int, loaded bool) {
	v, loaded := m.m.LoadOrStore(key, value)
	if v == nil {
		return 0, loaded
	}
	return v.(int), loaded
}

func (m *MapIntWrapper) LoadAndDelete(key int) (value int, loaded bool) {
	v, loaded := m.m.LoadAndDelete(key)
	if v == nil {
		return 0, loaded
	}
	return v.(int), loaded
}

func (m *MapIntWrapper) Delete(key int) {
	m.m.Delete(key)
}

func (m *MapIntWrapper) Swap(key, value int) (previous int, loaded bool) {
	v, loaded := m.m.Swap(key, value)
	if v == nil {
		return 0, loaded
	}
	return v.(int), loaded
}

func (m *MapIntWrapper) Range(fn func(key, value int) (shouldContinue bool)) {
	m.m.Range(func(key, value any) bool {
		return fn(key.(int), value.(int))
	})
}

func (m *MapIntWrapper) Clear() {
	m.m.Clear()
}

func (m *MapIntWrapper) CompareAndSwap(key, old, new int) (swapped bool) {
	return m.m.CompareAndSwap(key, old, new)
}

func (m *MapIntWrapper) CompareAndDelete(key, old int) (deleted bool) {
	return m.m.CompareAndDelete(key, old)
}

func benchMapInt(b *testing.B, bench benchInt) {
	maps := [...]casMapInterfaceInt{&MapIntWrapper{}, &sync_map.CasMap[int, int]{}}
	names := [...]string{"sync.MapWrapper", "Map[int,int]"}
	for i, m := range maps {
		b.Run(names[i], func(b *testing.B) {
			m = reflect.New(reflect.TypeOf(m).Elem()).Interface().(casMapInterfaceInt)
			if bench.setup != nil {
				bench.setup(b, m)
			}

			b.ResetTimer()

			var i int64
			b.RunParallel(func(pb *testing.PB) {
				id := int(atomic.AddInt64(&i, 1) - 1)
				bench.perG(b, pb, id*b.N, m)
			})
		})
	}
}

func BenchmarkLoadMostlyHitsInt(b *testing.B) {
	const hits, misses = 1023, 1

	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			for i := 0; i < hits; i++ {
				m.LoadOrStore(i, i)
			}
			// Prime the map to get it into a steady state.
			for i := 0; i < hits*2; i++ {
				m.Load(i % hits)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				m.Load(i % (hits + misses))
			}
		},
	})
}

func BenchmarkLoadMostlyMissesInt(b *testing.B) {
	const hits, misses = 1, 1023

	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			for i := 0; i < hits; i++ {
				m.LoadOrStore(i, i)
			}
			// Prime the map to get it into a steady state.
			for i := 0; i < hits*2; i++ {
				m.Load(i % hits)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				m.Load(i % (hits + misses))
			}
		},
	})
}

func BenchmarkLoadOrStoreBalancedInt(b *testing.B) {
	const hits, misses = 128, 128

	benchMapInt(b, benchInt{
		setup: func(b *testing.B, m mapInterfaceInt) {
			for i := 0; i < hits; i++ {
				m.LoadOrStore(i, i)
			}
			// Prime the map to get it into a steady state.
			for i := 0; i < hits*2; i++ {
				m.Load(i % hits)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				j := i % (hits + misses)
				if j < hits {
					if _, ok := m.LoadOrStore(j, i); !ok {
						b.Fatalf("unexpected miss for %v", j)
					}
				} else {
					if v, loaded := m.LoadOrStore(i, i); loaded {
						b.Fatalf("failed to store %v: existing value %v", i, v)
					}
				}
			}
		},
	})
}

func BenchmarkLoadOrStoreUniqueInt(b *testing.B) {
	benchMapInt(b, benchInt{
		setup: func(b *testing.B, m mapInterfaceInt) {
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				m.LoadOrStore(i, i)
			}
		},
	})
}

func BenchmarkLoadOrStoreCollisionInt(b *testing.B) {
	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			m.LoadOrStore(0, 0)
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				m.LoadOrStore(0, 0)
			}
		},
	})
}

func BenchmarkLoadAndDeleteBalancedInt(b *testing.B) {
	const hits, misses = 128, 128

	benchMapInt(b, benchInt{
		setup: func(b *testing.B, m mapInterfaceInt) {
			for i := 0; i < hits; i++ {
				m.LoadOrStore(i, i)
			}
			// Prime the map to get it into a steady state.
			for i := 0; i < hits*2; i++ {
				m.Load(i % hits)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				j := i % (hits + misses)
				if j < hits {
					m.LoadAndDelete(j)
				} else {
					m.LoadAndDelete(i)
				}
			}
		},
	})
}

func BenchmarkLoadAndDeleteUniqueInt(b *testing.B) {
	benchMapInt(b, benchInt{
		setup: func(b *testing.B, m mapInterfaceInt) {
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				m.LoadAndDelete(i)
			}
		},
	})
}

func BenchmarkLoadAndDeleteCollisionInt(b *testing.B) {
	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			m.LoadOrStore(0, 0)
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				if _, loaded := m.LoadAndDelete(0); loaded {
					m.Store(0, 0)
				}
			}
		},
	})
}

func BenchmarkRangeInt(b *testing.B) {
	const mapSize = 1 << 10

	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			for i := 0; i < mapSize; i++ {
				m.Store(i, i)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				m.Range(func(_, _ int) bool { return true })
			}
		},
	})
}

// BenchmarkAdversarialAlloc tests performance when we store a new value
// immediately whenever the map is promoted to clean and otherwise load a
// unique, missing key.
//
// This forces the Load calls to always acquire the map's mutex.
func BenchmarkAdversarialAllocInt(b *testing.B) {
	benchMapInt(b, benchInt{
		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			var stores, loadsSinceStore int
			for ; pb.Next(); i++ {
				m.Load(i)
				if loadsSinceStore++; loadsSinceStore > stores {
					m.LoadOrStore(i, stores)
					loadsSinceStore = 0
					stores++
				}
			}
		},
	})
}

// BenchmarkAdversarialDelete tests performance when we periodically delete
// one key and add a different one in a large map.
//
// This forces the Load calls to always acquire the map's mutex and periodically
// makes a full copy of the map despite changing only one entry.
func BenchmarkAdversarialDeleteInt(b *testing.B) {
	const mapSize = 1 << 10

	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			for i := 0; i < mapSize; i++ {
				m.Store(i, i)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				m.Load(i)

				if i%mapSize == 0 {
					m.Range(func(k, _ int) bool {
						m.Delete(k)
						return false
					})
					m.Store(i, i)
				}
			}
		},
	})
}

func BenchmarkDeleteCollisionInt(b *testing.B) {
	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			m.LoadOrStore(0, 0)
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				m.Delete(0)
			}
		},
	})
}

func BenchmarkSwapCollisionInt(b *testing.B) {
	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			m.LoadOrStore(0, 0)
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				m.Swap(0, 0)
			}
		},
	})
}

func BenchmarkSwapMostlyHitsInt(b *testing.B) {
	const hits, misses = 1023, 1

	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			for i := 0; i < hits; i++ {
				m.LoadOrStore(i, i)
			}
			// Prime the map to get it into a steady state.
			for i := 0; i < hits*2; i++ {
				m.Load(i % hits)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				if i%(hits+misses) < hits {
					v := i % (hits + misses)
					m.Swap(v, v)
				} else {
					m.Swap(i, i)
					m.Delete(i)
				}
			}
		},
	})
}

func BenchmarkSwapMostlyMissesInt(b *testing.B) {
	const hits, misses = 1, 1023

	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			for i := 0; i < hits; i++ {
				m.LoadOrStore(i, i)
			}
			// Prime the map to get it into a steady state.
			for i := 0; i < hits*2; i++ {
				m.Load(i % hits)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				if i%(hits+misses) < hits {
					v := i % (hits + misses)
					m.Swap(v, v)
				} else {
					m.Swap(i, i)
					m.Delete(i)
				}
			}
		},
	})
}

func BenchmarkCompareAndSwapCollisionInt(b *testing.B) {
	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			m.LoadOrStore(0, 0)
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for pb.Next() {
				if m.CompareAndSwap(0, 0, 42) {
					m.CompareAndSwap(0, 42, 0)
				}
			}
		},
	})
}

func BenchmarkCompareAndSwapNoExistingKeyInt(b *testing.B) {
	benchMapInt(b, benchInt{
		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				if m.CompareAndSwap(i, 0, 0) {
					m.Delete(i)
				}
			}
		},
	})
}

func BenchmarkCompareAndSwapValueNotEqualInt(b *testing.B) {
	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			m.Store(0, 0)
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				m.CompareAndSwap(0, 1, 2)
			}
		},
	})
}

func BenchmarkCompareAndSwapMostlyHitsInt(b *testing.B) {
	const hits, misses = 1023, 1

	benchMapInt(b, benchInt{
		setup: func(b *testing.B, m mapInterfaceInt) {
			for i := 0; i < hits; i++ {
				m.LoadOrStore(i, i)
			}
			// Prime the map to get it into a steady state.
			for i := 0; i < hits*2; i++ {
				m.Load(i % hits)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				v := i
				if i%(hits+misses) < hits {
					v = i % (hits + misses)
				}
				m.CompareAndSwap(v, v, v)
			}
		},
	})
}

func BenchmarkCompareAndSwapMostlyMissesInt(b *testing.B) {
	const hits, misses = 1, 1023

	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			for i := 0; i < hits; i++ {
				m.LoadOrStore(i, i)
			}
			// Prime the map to get it into a steady state.
			for i := 0; i < hits*2; i++ {
				m.Load(i % hits)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				v := i
				if i%(hits+misses) < hits {
					v = i % (hits + misses)
				}
				m.CompareAndSwap(v, v, v)
			}
		},
	})
}

func BenchmarkCompareAndDeleteCollisionInt(b *testing.B) {
	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			m.LoadOrStore(0, 0)
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				if m.CompareAndDelete(0, 0) {
					m.Store(0, 0)
				}
			}
		},
	})
}

func BenchmarkCompareAndDeleteMostlyHitsInt(b *testing.B) {
	const hits, misses = 1023, 1

	benchMapInt(b, benchInt{
		setup: func(b *testing.B, m mapInterfaceInt) {
			for i := 0; i < hits; i++ {
				m.LoadOrStore(i, i)
			}
			// Prime the map to get it into a steady state.
			for i := 0; i < hits*2; i++ {
				m.Load(i % hits)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				v := i
				if i%(hits+misses) < hits {
					v = i % (hits + misses)
				}
				if m.CompareAndDelete(v, v) {
					m.Store(v, v)
				}
			}
		},
	})
}

func BenchmarkCompareAndDeleteMostlyMissesInt(b *testing.B) {
	const hits, misses = 1, 1023

	benchMapInt(b, benchInt{
		setup: func(_ *testing.B, m mapInterfaceInt) {
			for i := 0; i < hits; i++ {
				m.LoadOrStore(i, i)
			}
			// Prime the map to get it into a steady state.
			for i := 0; i < hits*2; i++ {
				m.Load(i % hits)
			}
		},

		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				v := i
				if i%(hits+misses) < hits {
					v = i % (hits + misses)
				}
				if m.CompareAndDelete(v, v) {
					m.Store(v, v)
				}
			}
		},
	})
}

func BenchmarkClearInt(b *testing.B) {
	benchMapInt(b, benchInt{
		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterfaceInt) {
			for ; pb.Next(); i++ {
				k, v := i%256, i%256
				m.Clear()
				m.Store(k, v)
			}
		},
	})
}
