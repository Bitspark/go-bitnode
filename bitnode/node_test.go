package bitnode

import "testing"

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
