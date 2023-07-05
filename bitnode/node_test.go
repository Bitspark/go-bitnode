package bitnode

import (
	"github.com/Bitspark/go-bitnode/store"
	"testing"
)

func testNode(t *testing.T, dir string) (*NativeNode, *Domain) {
	dom := NewDomain()
	if err := dom.LoadFromDir(dir, true); err != nil {
		t.Fatal(err)
	}
	if err := dom.Compile(); err != nil {
		t.Fatal(err)
	}
	return NewNode(), dom
}

func TestDomainFromDir1(t *testing.T) {
	_, dom := testNode(t, "./test/types1")

	assistantDom, err := dom.GetDomain("assistant")
	if err != nil {
		t.Fatal(err)
	}
	if assistantDom == nil {
		t.Fatal()
	}

	// Types

	assistantType, err := dom.GetType("assistant.at")
	if err != nil {
		t.Fatal(err)
	}
	if assistantType == nil {
		t.Fatal()
	}
	if assistantType.MapOf["name"].Leaf != LeafString {
		t.Fatal()
	}

	assistantType, err = assistantDom.GetType("assistant.at")
	if err != nil {
		t.Fatal(err)
	}
	if assistantType == nil {
		t.Fatal()
	}
	if assistantType.MapOf["name"].Leaf != LeafString {
		t.Fatal()
	}

	assistantType, err = assistantDom.GetType("at")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDomainFromDir2(t *testing.T) {
	_, dom := testNode(t, "./test/inter1")

	io, err := dom.GetInterface("IO")
	if err != nil {
		t.Fatal(err)
	}
	if io == nil {
		t.Fatal()
	}
}

func TestDomainFromDir3(t *testing.T) {
	_, dom := testNode(t, "./test/impl1")

	hwInter, err := dom.GetInterface("HelloWorld")
	if err != nil {
		t.Fatal(err)
	}
	if hwInter == nil {
		t.Fatal()
	}

	hwImpl, err := dom.GetSparkable("HelloWorld")
	if err != nil {
		t.Fatal(err)
	}
	if hwImpl == nil {
		t.Fatal()
	}

	if hwImpl.Interface == nil {
		t.Fatal()
	}
}

func TestNativeNode_Store1(t *testing.T) {
	n := NewNode()
	name := n.Name()
	t.Log(name)

	sys, err := n.PrepareSystem(Credentials{}, Sparkable{
		RawSparkable: RawSparkable{
			Name:      "Blank",
			Interface: NewInterface(),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	id := sys.ID()
	sysName := sys.Name()
	t.Log(sysName)

	st := store.NewStore("test")

	if err := n.Store(st); err != nil {
		t.Fatal(err)
	}

	n2 := NewNode()

	if err := n2.Load(st, nil); err != nil {
		t.Fatal()
	}

	if n2.Name() != name {
		t.Fatal(n2.Name())
	}

	sys2, err := n2.GetSystemByID(Credentials{}, id)
	if err != nil {
		t.Fatal(err)
	}

	if sys2.ID() != id {
		t.Fatal(sys2.ID())
	}
	if sys2.Name() != sysName {
		t.Fatal(sys2.Name())
	}
}

func TestNativeNode_Store2(t *testing.T) {
	n := NewNode()

	sys1, _ := n.PrepareSystem(Credentials{}, Sparkable{
		RawSparkable: RawSparkable{
			Name:      "Blank",
			Interface: NewInterface(),
		},
	})
	sys2, _ := n.PrepareSystem(Credentials{}, Sparkable{
		RawSparkable: RawSparkable{
			Name:      "Blank",
			Interface: NewInterface(),
		},
	})
	sys3, _ := n.PrepareSystem(Credentials{}, Sparkable{
		RawSparkable: RawSparkable{
			Name:      "Blank",
			Interface: NewInterface(),
		},
	})

	sys2.Native().AddOrigin("b", sys3.Native())
	sys1.Native().AddOrigin("a", sys2.Native())

	st := store.NewStore("test")

	if err := n.Store(st); err != nil {
		t.Fatal(err)
	}

	n2 := NewNode()

	if err := n2.Load(st, nil); err != nil {
		t.Fatal()
	}

	sys1b, _ := n2.GetSystemByID(Credentials{}, sys1.ID())
	sys2b, _ := n2.GetSystemByID(Credentials{}, sys2.ID())
	sys3b, _ := n2.GetSystemByID(Credentials{}, sys3.ID())

	orig := sys1b.Origin("a")
	if orig.Native() != sys2b.Native() {
		t.Fatal()
	}

	parents := sys2b.Native().parents
	if len(parents) != 1 {
		t.Fatal()
	}
	if parents[0].Origin != sys1b.Native() {
		t.Fatal()
	}

	orig2 := sys2b.Origin("b")
	if orig2.Native() != sys3b.Native() {
		t.Fatal()
	}

	parents2 := sys3b.Native().parents
	if len(parents2) != 1 {
		t.Fatal()
	}
	if parents2[0].Origin != sys2b.Native() {
		t.Fatal()
	}
}
