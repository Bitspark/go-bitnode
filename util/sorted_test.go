package util

import "testing"

func TestSorted_Add(t *testing.T) {
	s := NewSorted[int, string]()
	s.Add(4, "a")
	s.Add(2, "b")
	s.Add(8, "c")
	s.Add(1, "d")

	if s.Entries[0].Value != "d" {
		t.Fatal(s.Entries)
	}
	if s.Entries[1].Value != "b" {
		t.Fatal(s.Entries)
	}
	if s.Entries[2].Value != "a" {
		t.Fatal(s.Entries)
	}
	if s.Entries[3].Value != "c" {
		t.Fatal(s.Entries)
	}
}
