package factories

import (
	"github.com/Bitspark/go-bitnode/bitnode"
	"gopkg.in/yaml.v3"
	"testing"
	"time"
)

func testNodeNode(t *testing.T) (*bitnode.NativeNode, *bitnode.Domain) {
	h := bitnode.NewNode()
	dom := bitnode.NewDomain()
	h.AddMiddlewares(GetMiddlewares())
	_ = h.AddFactory(NewNodeFactory())
	hubDom, err := dom.AddDomain("hub")
	if err != nil {
		t.Fatal(err)
	}
	if err := hubDom.LoadFromDir("../library", true); err != nil {
		t.Fatal(err)
	}
	if err := dom.Compile(); err != nil {
		t.Fatal(err)
	}
	return h, dom
}

func TestNodeNodeSystem1(t *testing.T) {
	h, dom := testNodeNode(t)

	creds := bitnode.Credentials{}

	nodeBP, err := dom.GetSparkable("hub.meta.Node")
	if err != nil {
		t.Fatal(err)
	}
	if nodeBP == nil {
		t.Fatal()
	}

	nodeSys, err := h.NewSystem(creds, *nodeBP)
	if err != nil {
		t.Fatal(err)
	}
	if nodeSys == nil {
		t.Fatal()
	}

	nodeIF := nodeSys.Interface()
	if err := nodeIF.Hubs.Contains(nodeBP.Interface.Hubs); err != nil {
		t.Fatal(err)
	}

	createSystem := nodeSys.GetHub("createSystem")
	if createSystem == nil {
		t.Fatal()
	}

	r, err := createSystem.Invoke(nil, &bitnode.Sparkable{})
	if err != nil {
		t.Fatal(err)
	}
	if r == nil {
		t.Fatal()
	}
	tsys := r[0].(bitnode.System)

	getSystems := nodeSys.GetHub("getSystems")
	if getSystems == nil {
		t.Fatal()
	}

	st, err := getSystems.Invoke(nil)
	if err != nil {
		t.Fatal(err)
	}
	s := st[0].([]bitnode.HubItem)
	if len(s) != 2 || (s[0].(bitnode.System).ID() != nodeSys.ID() && s[1].(bitnode.System).ID() != nodeSys.ID()) || (s[0].(bitnode.System).ID() != tsys.ID() && s[1].(bitnode.System).ID() != tsys.ID()) {
		t.Fatal()
	}
}

func TestNodeNodeSystem2(t *testing.T) {
	h, dom := testNodeNode(t)
	h2, _ := testNodeNode(t)

	creds := bitnode.Credentials{}

	nodeBP, err := dom.GetSparkable("hub.meta.Node")
	if err != nil {
		t.Fatal(err)
	}
	if nodeBP == nil {
		t.Fatal()
	}

	nodeSys, err := h.NewSystem(creds, *nodeBP)
	if err != nil {
		t.Fatal(err)
	}
	if nodeSys == nil {
		t.Fatal()
	}

	nodeIF := nodeSys.Interface()
	if err := nodeIF.Hubs.Contains(nodeBP.Interface.Hubs); err != nil {
		t.Fatal(err)
	}

	createSystem := nodeSys.GetHub("addSystem")
	if createSystem == nil {
		t.Fatal()
	}

	interf2 := bitnode.Interface{}
	_ = yaml.Unmarshal([]byte(`
hubs:
  - name: a
    input: []
    output: []
    type: pipe
    direction: in
`), &interf2)
	if err := interf2.Compile(nil, "", false); err != nil {
		t.Fatal(err)
	}
	sys2, err := h2.NewSystem(creds, interf2.Blank())
	if err != nil {
		t.Fatal(err)
	}

	invoked := 0

	hub2 := sys2.GetHub("a")
	hub2.Handle(bitnode.NewNativeFunction(func(creds bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
		invoked++
		return []bitnode.HubItem{}, err
	}))

	r, err := createSystem.Invoke(nil, sys2)
	if err != nil {
		t.Fatal(err)
	}
	if r == nil {
		t.Fatal()
	}
	if len(r) != 1 {
		t.Fatal()
	}

	time.Sleep(500 * time.Millisecond)

	sys3, err := h.GetSystemByName(creds, sys2.Name())
	if err != nil {
		t.Fatal(err)
	}
	if sys3 == nil {
		t.Fatal()
	}

	hub3 := sys3.GetHub("a")
	rets, err := hub3.Invoke(nil)
	if err != nil {
		t.Fatal(err)
	}
	if rets == nil {
		t.Fatal()
	}
	if len(rets) != 0 {
		t.Fatal()
	}

	if invoked != 1 {
		t.Fatal()
	}
}

func TestNodeNodeSystem__Addresses1(t *testing.T) {
	h, dom := testNodeNode(t)

	creds := bitnode.Credentials{}

	nodeBP, err := dom.GetSparkable("hub.meta.Node")
	if err != nil {
		t.Fatal(err)
	}
	if nodeBP == nil {
		t.Fatal()
	}

	nodeSys, err := h.NewSystem(creds, *nodeBP)
	if err != nil {
		t.Fatal(err)
	}
	if nodeSys == nil {
		t.Fatal()
	}

	if len(nodeSys.Extends()) == 0 {
		t.Fatal()
	}

	nodeIF := nodeSys.Interface()
	if err := nodeIF.Hubs.Contains(nodeBP.Interface.Hubs); err != nil {
		t.Fatal(err)
	}

	getAddresses := nodeSys.GetHub("getAddresses")
	if getAddresses == nil {
		t.Fatal()
	}

	rets, err := getAddresses.Invoke(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rets) != 1 {
		t.Fatal(rets)
	}
	retMps, ok := rets[0].([]bitnode.HubItem)
	if !ok {
		t.Fatal(rets[0])
	}
	if len(retMps) != 0 {
		t.Fatal()
	}

	setAddress := nodeSys.GetHub("setAddress")
	if setAddress == nil {
		t.Fatal()
	}
	rets, err = setAddress.Invoke(nil, "http:test", "localhost:123")
	if err != nil {
		t.Fatal(err)
	}
	if len(rets) != 0 {
		t.Fatal(rets)
	}

	rets, err = getAddresses.Invoke(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rets) != 1 {
		t.Fatal(rets)
	}
	retMps, ok = rets[0].([]bitnode.HubItem)
	if !ok {
		t.Fatal(rets[0])
	}
	if len(retMps) != 1 {
		t.Fatal()
	}

	retMp, ok := retMps[0].(map[string]bitnode.HubItem)
	if !ok {
		t.Fatal(retMps[0])
	}
	if retMp["network"].(string) != "http:test" {
		t.Fatal(retMp)
	}
	if retMp["address"].(string) != "localhost:123" {
		t.Fatal(retMp)
	}
}
