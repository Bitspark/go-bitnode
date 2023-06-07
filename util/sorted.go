package util

import (
	"golang.org/x/exp/constraints"
)

type Ordered interface {
	constraints.Ordered
}

type Entry[K Ordered, V any] struct {
	Key   K
	Value V
}

type Sorted[K Ordered, V any] struct {
	Entries []Entry[K, V]
}

func NewSorted[K Ordered, V any]() Sorted[K, V] {
	return Sorted[K, V]{
		Entries: []Entry[K, V]{},
	}
}

func (s *Sorted[K, V]) Add(k K, v V) {
	idx := s.FindIndex(k)
	entries := make([]Entry[K, V], len(s.Entries)+1)
	copy(entries, s.Entries[:idx])
	copy(entries[idx+1:], s.Entries[idx:])
	entries[idx] = Entry[K, V]{Key: k, Value: v}
	s.Entries = entries
}

func (s *Sorted[K, V]) FindIndex(k K) int {
	return findIndex(0, s.Entries, k)
}

func findIndex[K Ordered, V any](offset int, entries []Entry[K, V], k K) int {
	if len(entries) == 0 {
		return offset
	}
	if len(entries) == 1 {
		if k < entries[0].Key {
			return offset
		} else {
			return offset + 1
		}
	}
	middle := len(entries) / 2
	if k < entries[middle].Key {
		return findIndex(offset, entries[:middle], k)
	} else {
		return findIndex(offset+middle, entries[middle:], k)
	}
}
