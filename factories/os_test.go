package factories

import (
	"github.com/Bitspark/go-bitnode/bitnode"
	"github.com/Bitspark/go-bitnode/store"
	"testing"
	"time"
)

func testNodeOS(t *testing.T, dir string) (*bitnode.NativeNode, *bitnode.Domain) {
	h := bitnode.NewNode()
	if err := h.AddFactory(NewOSFactory()); err != nil {
		t.Fatal(err)
	}
	hubDom, _ := bitnode.NewDomain().AddDomain("hub")
	if err := hubDom.LoadFromDir("../library/", false); err != nil {
		t.Fatal(err)
	}
	osDom, _ := hubDom.AddDomain("os")
	if err := osDom.LoadFromDir(dir, false); err != nil {
		t.Fatal(err)
	}
	if err := osDom.Compile(); err != nil {
		t.Fatal(err)
	}
	return h, osDom
}

func TestOS1(t *testing.T) {
	h, dom := testNodeOS(t, "../library/os/")

	creds := bitnode.Credentials{}

	compImpl, err := dom.GetSparkable("OperatingSystem")
	if err != nil {
		t.Fatal(err)
	}
	if compImpl == nil {
		t.Fatal()
	}

	compSys, err := h.NewSystem(creds, *compImpl)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	hub := compSys.GetHub("hostname")
	ts, err := hub.Get()
	if err != nil {
		t.Fatal(err)
	}
	if ts.(string) == "" {
		t.Fatal()
	}
	t.Log(ts.(string))
}

func TestOS2(t *testing.T) {
	h, dom := testNodeOS(t, "../library/os/")

	creds := bitnode.Credentials{}

	compImpl, err := dom.GetSparkable("OperatingSystem")
	if err != nil {
		t.Fatal(err)
	}
	if compImpl == nil {
		t.Fatal()
	}

	compSys, err := h.NewSystem(creds, *compImpl)
	if err != nil {
		t.Fatal(err)
	}

	st := store.NewStore("test")

	if err := h.Store(st); err != nil {
		t.Fatal(err)
	}

	h2, dom2 := testNodeOS(t, "../library/os/")

	if err := h2.Load(st, dom2); err != nil {
		t.Fatal(err)
	}

	compSys2, err := h2.GetSystemByID(creds, compSys.ID())
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	hub2 := compSys2.GetHub("hostname")
	ts, err := hub2.Get()
	if err != nil {
		t.Fatal(err)
	}
	if ts.(string) == "" {
		t.Fatal()
	}
	t.Log(ts.(string))
}

func TestOS3(t *testing.T) {
	h, dom := testNodeOS(t, "../library/os/")

	creds := bitnode.Credentials{}

	compImpl, err := dom.GetSparkable("OperatingSystem")
	if err != nil {
		t.Fatal(err)
	}
	if compImpl == nil {
		t.Fatal()
	}

	compSys, err := h.NewSystem(creds, *compImpl)
	if err != nil {
		t.Fatal(err)
	}

	cpus := compSys.GetHub("cpus")
	memory := compSys.GetHub("memory")

	time.Sleep(2 * time.Second)

	ts, err := cpus.Get()
	if err != nil {
		t.Fatal(err)
	}
	tss := ts.([]bitnode.HubItem)

	for _, t1 := range tss {
		t.Log(t1)
	}

	ts, err = memory.Get()
	if err != nil {
		t.Fatal(err)
	}
	tss2 := ts.(map[string]bitnode.HubItem)
	t.Log(tss2)

	used := 1 - (float64(tss2["free"].(int64)) / float64(tss2["total"].(int64)))

	t.Log(used)
}
