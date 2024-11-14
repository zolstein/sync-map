//go:build go1.23

package sync_map_test

import "testing"

func BenchmarkClear(b *testing.B) {
	benchMap(b, bench{
		perG: func(b *testing.B, pb *testing.PB, i int, m casMapInterface) {
			for ; pb.Next(); i++ {
				k, v := i%256, i%256
				m.Clear()
				m.Store(k, v)
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
