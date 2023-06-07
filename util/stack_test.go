package util

import "testing"

func TestStack_Pop(t *testing.T) {
	s := NewStack[string]()

	if s.Size() != 0 {
		t.Fatal()
	}

	s.Push("1")
	s.Push("2")

	if s.Size() != 2 {
		t.Fatal()
	}

	if s.Pop() != "2" {
		t.Fatal()
	}

	if s.Size() != 1 {
		t.Fatal()
	}

	if s.Pop() != "1" {
		t.Fatal()
	}

	if s.Size() != 0 {
		t.Fatal()
	}
}
