package factories

import (
	"github.com/Bitspark/go-bitnode/bitnode"
	"testing"
)

func testNodeHub(t *testing.T, dir string) (*bitnode.NativeNode, *bitnode.Domain) {
	h := bitnode.NewNode()
	dom := bitnode.NewDomain()
	h.AddMiddlewares(GetMiddlewares())
	_ = h.AddFactory(NewTimeFactory())
	_ = h.AddFactory(NewJSFactory(dom))
	_ = h.AddFactory(NewNodeFactory())

	hubDom, err := dom.AddDomain("hub")
	if err != nil {
		t.Fatal(err)
	}
	if err := hubDom.LoadFromDir(dir, true); err != nil {
		t.Fatal(err)
	}
	if err := hubDom.Compile(); err != nil {
		t.Fatal(err)
	}
	return h, dom
}

func TestRoot1(t *testing.T) {
	_, dom := testNodeHub(t, "../library/")

	ensureDomains := []string{"hub", "hub.meta", "hub.time", "hub.program"}
	ensureTypes := []string{"hub.type", "hub.interface", "hub.blueprint", "hub.system"}
	ensureInterfaces := []string{"hub.meta.Node", "hub.time.Clock", "hub.time.Trigger", "hub.program.Program"}
	ensureBlueprints := []string{"hub.meta.Node", "hub.time.Clock", "hub.time.Trigger"}

	for _, ee := range ensureDomains {
		if tp, err := dom.GetDomain(ee); err != nil || tp == nil {
			t.Fatalf("missing domain: %s", ee)
		}
	}

	for _, ee := range ensureTypes {
		if tp, err := dom.GetType(ee); err != nil || tp == nil {
			t.Fatalf("missing type: %s", ee)
		}
	}

	for _, ee := range ensureInterfaces {
		if interf, err := dom.GetInterface(ee); err != nil || interf == nil {
			t.Fatalf("missing interface: %s", ee)
		}
	}

	for _, ee := range ensureBlueprints {
		if bp, err := dom.GetSparkable(ee); err != nil || bp == nil {
			t.Fatalf("missing blueprint: %s", ee)
		}
	}
}
