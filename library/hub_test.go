package library

import (
	"github.com/Bitspark/go-bitnode/bitnode"
	"github.com/Bitspark/go-bitnode/library/meta"
	"github.com/Bitspark/go-bitnode/library/program"
	"github.com/Bitspark/go-bitnode/library/time"
	"testing"
)

func testNode(t *testing.T, dir string) (*bitnode.NativeNode, *bitnode.Domain) {
	h := bitnode.NewNode()
	dom := bitnode.NewDomain()
	h.AddMiddlewares(GetMiddlewares())
	_ = h.AddFactory(libTime.NewTimeFactory())
	_ = h.AddFactory(libProgram.NewJSFactory(dom))
	_ = h.AddFactory(libMeta.NewNodeFactory())

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
	_, dom := testNode(t, "./")

	ensureDomains := []string{"hub", "hub.meta", "hub.time", "hub.program"}
	ensureTypes := []string{"hub.type", "hub.interface", "hub.blueprint", "hub.system"}
	ensureInterfaces := []string{"hub.meta.Node", "hub.time.Clock", "hub.time.Trigger", "hub.program.Program"}
	ensureBlueprints := []string{"hub.meta.Node", "hub.time.Clock", "hub.time.Trigger"}
	ensureModels := []string{"hub.Object"}

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

	for _, ee := range ensureModels {
		if bp, err := dom.GetModel(ee); err != nil || bp == nil {
			t.Fatalf("missing model: %s", ee)
		}
	}
}
