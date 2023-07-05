package bitnode

import (
	"github.com/Bitspark/go-bitnode/store"
	"testing"
)

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

func TestNativeSystem_Store1(t *testing.T) {
	h := NewNode()

	sys, err := h.NewSystem(Credentials{}, Sparkable{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sys == nil {
		t.Fatal()
	}

	sys.SetName("test2")

	st := store.NewStore("test")

	if err := sys.Native().Store(st); err != nil {
		t.Fatal(err)
	}

	h2 := NewNode()

	sys2 := &NativeSystem{}

	if err := sys2.LoadInit(h2, st); err != nil {
		t.Fatal(err)
	}

	if sys2.Name() != "test2" {
		t.Fatal(sys2.Name())
	}
}

func TestNativeSystem_Store2(t *testing.T) {
	h := NewNode()

	sys, err := h.NewSystem(Credentials{}, Sparkable{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if sys == nil {
		t.Fatal()
	}

	sys.Native().SetRemoteNode("abc123")
	sys.Native().SetRemoteID(GenerateSystemID())

	st := store.NewStore("test")

	if err := sys.Native().Store(st); err != nil {
		t.Fatal(err)
	}

	h2 := NewNode()

	sys2 := &NativeSystem{}

	if err := sys2.LoadInit(h2, st); err != nil {
		t.Fatal(err)
	}

	if sys2.RemoteNode() != "abc123" {
		t.Fatal(sys2.RemoteNode())
	}
	if sys2.RemoteID() != sys.Native().RemoteID() {
		t.Fatal(sys2.RemoteID().Hex())
	}
}

func TestNativeSystem_Origin1(t *testing.T) {
	h := NewNode()

	sys, _ := h.NewSystem(Credentials{}, Sparkable{}, nil)
	sys1, _ := h.NewSystem(Credentials{}, Sparkable{}, nil)

	sys.Native().AddOrigin("a", sys1.Native())

	if sys.Origin("").Native() != sys.Native() {
		t.Fatal()
	}
	if sys.Origin("a").Native() != sys1.Native() {
		t.Fatal()
	}
}

func TestNativeSystem_Origin2(t *testing.T) {
	h := NewNode()

	sys, _ := h.NewSystem(Credentials{}, Sparkable{}, nil)
	sysA, _ := h.NewSystem(Credentials{}, Sparkable{}, nil)
	sysB, _ := h.NewSystem(Credentials{}, Sparkable{}, nil)
	sysA1, _ := h.NewSystem(Credentials{}, Sparkable{}, nil)
	sysB1, _ := h.NewSystem(Credentials{}, Sparkable{}, nil)
	sysB2, _ := h.NewSystem(Credentials{}, Sparkable{}, nil)

	sysA.Native().AddOrigin("1", sysA1.Native())
	sysB.Native().AddOrigin("1", sysB1.Native())
	sysB.Native().AddOrigin("2", sysB2.Native())

	sys.Native().AddOrigin("a", sysA.Native())
	sys.Native().AddOrigin("b", sysB.Native())

	if sys.Origin("").Native() != sys.Native() {
		t.Fatal()
	}
	if sys.Origin("a").Native() != sysA.Native() {
		t.Fatal()
	}
	if sys.Origin("b").Native() != sysB.Native() {
		t.Fatal()
	}

	if sys.Origin("a").Origin("1").Native() != sysA1.Native() {
		t.Fatal()
	}
	if sys.Origin("b").Origin("1").Native() != sysB1.Native() {
		t.Fatal()
	}
	if sys.Origin("b").Origin("2").Native() != sysB2.Native() {
		t.Fatal()
	}

	if sys.Origin("a/1").Native() != sysA1.Native() {
		t.Fatal()
	}
	if sys.Origin("b/1").Native() != sysB1.Native() {
		t.Fatal()
	}
	if sys.Origin("b/2").Native() != sysB2.Native() {
		t.Fatal()
	}
}
