package libTime

import (
	"github.com/Bitspark/go-bitnode/bitnode"
	"math"
	"testing"
	"time"
)

func testNode(t *testing.T, dir string) (*bitnode.NativeNode, *bitnode.Domain) {
	h := bitnode.NewNode()
	if err := h.AddFactory(&TimeFactory{}); err != nil {
		t.Fatal(err)
	}
	testDom, _ := bitnode.NewDomain().AddDomain("time")
	if err := testDom.LoadFromDir(dir, false); err != nil {
		t.Fatal(err)
	}
	if err := testDom.Compile(); err != nil {
		t.Fatal(err)
	}
	return h, testDom
}

func TestTime1(t *testing.T) {
	h, dom := testNode(t, "./")

	creds := bitnode.Credentials{}

	clockImpl, err := dom.GetSparkable("Clock")
	if err != nil {
		t.Fatal(err)
	}
	if clockImpl == nil {
		t.Fatal()
	}

	clockSys, err := h.NewSystem(creds, *clockImpl)
	if err != nil {
		t.Fatal(err)
	}

	hub := clockSys.GetHub("getTimestamp")
	ts, err := hub.Invoke(nil)
	if err != nil {
		t.Fatal(err)
	}
	diff := math.Abs(ts[0].(float64) - float64(time.Now().UnixNano())/float64(time.Second/time.Nanosecond))
	if diff > 0.001 {
		t.Fatal(diff)
	}
}

func TestTime2(t *testing.T) {
	h, dom := testNode(t, "./")

	creds := bitnode.Credentials{}

	triggerImpl, err := dom.GetSparkable("Trigger")
	if err != nil {
		t.Fatal(err)
	}
	if triggerImpl == nil {
		t.Fatal()
	}

	clockSys, err := h.NewSystem(creds, *triggerImpl, 0.1)
	if err != nil {
		t.Fatal(err)
	}

	ticks := int64(0)
	elapsed := 0.0
	clockSys.GetHub("tick").Subscribe(bitnode.NewNativeSubscription(func(id string, creds bitnode.Credentials, val bitnode.HubItem) {
		valm := val.(map[string]bitnode.HubItem)
		if ticks != valm["ticks"].(int64) {
			t.Fatal()
		}
		ticks++
		elapsed = valm["elapsed"].(float64)
	}))

	time.Sleep(1 * time.Second)

	if ticks < 9 || ticks > 11 {
		t.Fatal(ticks)
	}
	if elapsed < 0.8 || elapsed > 1.2 {
		t.Fatal(elapsed)
	}
}
