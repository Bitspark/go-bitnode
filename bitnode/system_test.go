package bitnode

import "testing"

func TestSystem1(t *testing.T) {
	h := NewNode()

	sys, err := h.NewSystem(Credentials{}, Sparkable{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sys == nil {
		t.Fatal()
	}

	sys.SetName("test2")

	if sys.Name() != "test2" {
		t.Fatal()
	}
}
