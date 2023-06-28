package api

import (
	"context"
	"github.com/Bitspark/go-bitnode/api/wsApi"
	"github.com/Bitspark/go-bitnode/bitnode"
	"github.com/Bitspark/go-bitnode/factories"
	"github.com/Bitspark/go-bitnode/store"
	"gopkg.in/yaml.v3"
	"strings"
	"testing"
	"time"
)

func testClient(t *testing.T, rootSystem string, addr string) (*bitnode.NativeNode, *wsApi.NodeConns, *bitnode.Domain) {
	node := bitnode.NewNode()
	dom := bitnode.NewDomain()
	conns := wsApi.NewNodeConns(node, addr)

	node.AddMiddlewares(factories.GetMiddlewares())

	_ = node.AddFactory(factories.NewTimeFactory())
	_ = node.AddFactory(factories.NewJSFactory(dom))
	_ = node.AddFactory(factories.NewNodeFactory())
	_ = node.AddFactory(factories.NewOSFactory())
	_ = node.AddFactory(wsApi.NewWSFactory(conns))

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

	if rootSystem != "" {
		bp, err := dom.GetSparkable(rootSystem)
		if err != nil {
			t.Fatal(err)
		}

		sys, err := node.NewSystem(bitnode.Credentials{}, *bp)
		if err != nil {
			t.Fatal(err)
		}
		if sys == nil {
			t.Fatal()
		}

		node.SetSystem(sys.Native())
	}

	return node, conns, dom
}

func TestNodeClient1(t *testing.T) {
	snode, sconns, _ := testClient(t, "", "ws://127.0.0.1:12345")
	_, cconns, _ := testClient(t, "", "")

	//creds := bitnode.Credentials{}

	srv := wsApi.NewServer(sconns, "0.0.0.0:12345")
	defer srv.Shutdown(context.Background())
	go srv.Listen()

	time.Sleep(500 * time.Millisecond)

	creds := bitnode.Credentials{}

	sys1, err := snode.NewSystem(creds, bitnode.Sparkable{})
	if err != nil {
		t.Fatal(err)
	}
	if sys1 == nil {
		t.Fatal()
	}

	if _, err := cconns.ConnectNode("ws://127.0.0.1:12345"); err != nil {
		t.Fatal(err)
	}

	clt1, err := cconns.GetNodeByName(snode.Name()).AddClient()
	if err != nil {
		t.Fatal(err)
	}
	if err := clt1.Connect(sys1.ID(), bitnode.Credentials{}); err != nil {
		t.Fatal(err)
	}
}

func TestNodeClient2(t *testing.T) {
	snode, sconns, dom := testClient(t, "", "ws://127.0.0.1:12345")
	_, cconns, _ := testClient(t, "", "")

	srv := wsApi.NewServer(sconns, "0.0.0.0:12345")
	defer srv.Shutdown(context.Background())
	go srv.Listen()

	time.Sleep(500 * time.Millisecond)

	bp, err := dom.GetSparkable("hub.meta.Node")
	if err != nil {
		t.Fatal(err)
	}

	creds := bitnode.Credentials{}

	sys, err := snode.NewSystem(creds, *bp)
	if err != nil {
		t.Fatal(err)
	}
	if sys == nil {
		t.Fatal()
	}

	if _, err := cconns.ConnectNode("ws://127.0.0.1:12345"); err != nil {
		t.Fatal(err)
	}

	clt, err := cconns.GetNodeByName(snode.Name()).AddClient()
	if err != nil {
		t.Fatal(err)
	}
	if err := clt.Connect(sys.ID(), bitnode.Credentials{}); err != nil {
		t.Fatal(err)
	}

	hub := clt.GetHub("getSystemCount")
	if hub == nil {
		t.Fatal()
	}

	ret, err := hub.Invoke(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ret) != 1 || ret[0] != int64(1) {
		t.Fatal(ret)
	}
}

func TestNodeClient3(t *testing.T) {
	snode, sconns, dom := testClient(t, "", "ws://127.0.0.1:12346")
	cnode, cconns, _ := testClient(t, "", "")

	srv := wsApi.NewServer(sconns, "0.0.0.0:12346")
	defer srv.Shutdown(context.Background())
	go srv.Listen()

	time.Sleep(500 * time.Millisecond)

	bp, err := dom.GetSparkable("hub.meta.Node")
	if err != nil {
		t.Fatal(err)
	}

	creds := bitnode.Credentials{}

	sys, err := snode.NewSystem(creds, *bp)
	if err != nil {
		t.Fatal(err)
	}
	if sys == nil {
		t.Fatal()
	}

	if _, err := cconns.ConnectNode("ws://127.0.0.1:12346"); err != nil {
		t.Fatal(err)
	}

	clt, err := cconns.GetNodeByName(snode.Name()).AddClient()
	if err != nil {
		t.Fatal(err)
	}
	if err := clt.Connect(sys.ID(), bitnode.Credentials{}); err != nil {
		t.Fatal(err)
	}

	getSystemsHub := clt.GetHub("getSystems")
	if getSystemsHub == nil {
		t.Fatal()
	}

	ret, err := getSystemsHub.Invoke(nil)
	if err != nil {
		t.Fatal(err)
	}
	retv := ret[0].([]bitnode.HubItem)
	if len(ret) != 1 || len(retv) != 1 {
		t.Fatal(retv)
	}
	rets, ok := retv[0].(*wsApi.Client)
	if !ok {
		t.Fatal(retv[0])
	}

	syss := snode.Systems(creds)
	if len(syss) != 1 {
		t.Fatal(len(syss), syss)
	}

	if err := rets.Connect(rets.RemoteID(), rets.Credentials()); err != nil {
		t.Fatal(err)
	}

	syss = cnode.Systems(creds)
	if len(syss) != 2 {
		t.Fatal(snode.Systems(creds))
	}

	getSystemsHub = rets.GetHub("getSystemCount")
	if getSystemsHub == nil {
		t.Fatal()
	}

	ret, err = getSystemsHub.Invoke(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ret) != 1 || ret[0] != int64(1) {
		t.Fatal(ret)
	}
}

func TestNodeClient4(t *testing.T) {
	snode, sconns, dom := testClient(t, "hub.meta.Node", "ws://127.0.0.1:12347")
	cnode1, cconns1, _ := testClient(t, "", "")
	_, cconns2, _ := testClient(t, "", "")
	_, cconns3, _ := testClient(t, "", "")

	creds := bitnode.Credentials{}

	srv := wsApi.NewServer(sconns, "0.0.0.0:12347")
	defer srv.Shutdown(context.Background())
	go srv.Listen()

	time.Sleep(500 * time.Millisecond)

	if _, err := cconns1.ConnectNode("ws://127.0.0.1:12347"); err != nil {
		t.Fatal(err)
	}

	if _, err := cconns2.ConnectNode("ws://127.0.0.1:12347"); err != nil {
		t.Fatal(err)
	}

	if _, err := cconns3.ConnectNode("ws://127.0.0.1:12347"); err != nil {
		t.Fatal(err)
	}

	clt1, err := cconns1.GetNodeByName(snode.Name()).AddClient()
	if err != nil {
		t.Fatal(err)
	}
	if err := clt1.Connect(bitnode.SystemID{}, bitnode.Credentials{}); err != nil {
		t.Fatal(err)
	}

	clt2, err := cconns1.GetNodeByName(snode.Name()).AddClient()
	if err != nil {
		t.Fatal(err)
	}
	if err := clt2.Connect(bitnode.SystemID{}, bitnode.Credentials{}); err != nil {
		t.Fatal(err)
	}

	interf2 := bitnode.Interface{}
	_ = yaml.Unmarshal([]byte(`
hubs:
  - name: pipe
    input:
      - name: val
        value:
          leaf: string
    output:
      - name: val1
        value:
          leaf: string
      - name: val2
        value:
          leaf: string
    type: pipe
    direction: in
`), &interf2)
	if err := interf2.Compile(dom, "", false); err != nil {
		t.Fatal(err)
	}

	sys1, _ := cnode1.NewSystem(creds, interf2.Blank())
	sys1.GetHub("pipe").Handle(bitnode.NewNativeFunction(func(creds bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
		return []bitnode.HubItem{vals[0], vals[0]}, nil
	}))

	hsyss, _ := clt2.GetHub("getSystems").Invoke(nil)
	if len(hsyss[0].([]bitnode.HubItem)) != 1 {
		t.Fatal()
	}

	hsyss, _ = clt1.GetHub("addSystem").Invoke(nil, sys1)
	addedSys := hsyss[0].(*wsApi.Client)

	hsyss, _ = clt2.GetHub("getSystems").Invoke(nil)
	if len(hsyss[0].([]bitnode.HubItem)) != 2 {
		t.Fatal()
	}

	var hsys bitnode.System
	for _, s := range hsyss[0].([]bitnode.HubItem) {
		sc, ok := s.(*wsApi.Client)
		if !ok {
			t.Fatal()
		}
		if sc.RemoteID() == addedSys.RemoteID() {
			hsys = s.(bitnode.System)
		}
	}

	if hsys == nil {
		t.Fatal()
	}

	pipeHub := hsys.GetHub("pipe")
	if pipeHub == nil {
		t.Fatal()
	}
	ret, err := pipeHub.Invoke(nil, "a_string")
	if err != nil {
		t.Fatal(err)
	}
	if len(ret) != 2 {
		t.Fatal()
	}
	if ret[0].(string) != "a_string" || ret[1].(string) != "a_string" {
		t.Fatal(ret)
	}
}

func TestNodeClient5(t *testing.T) {
	snode, sconns, dom := testClient(t, "", "ws://127.0.0.1:12350")
	_, cconns1, _ := testClient(t, "", "")

	creds := bitnode.Credentials{}

	srv := wsApi.NewServer(sconns, "0.0.0.0:12350")
	defer srv.Shutdown(context.Background())
	go srv.Listen()

	time.Sleep(500 * time.Millisecond)

	tickerBP, _ := dom.GetSparkable("hub.time.Trigger")
	tickerSys, _ := snode.NewSystem(creds, *tickerBP, 0.1)

	snode.SetSystem(tickerSys.Native())

	if _, err := cconns1.ConnectNode("ws://127.0.0.1:12350"); err != nil {
		t.Fatal(err)
	}

	clt1, err := cconns1.GetNodeByName(snode.Name()).AddClient()
	if err != nil {
		t.Fatal(err)
	}
	if err := clt1.Connect(bitnode.SystemID{}, bitnode.Credentials{}); err != nil {
		t.Fatal(err)
	}

	clientTicks := 0
	serverTicks := 0

	tickHubClt := clt1.GetHub("tick")
	tickHubClt.Subscribe(bitnode.NewNativeSubscription(func(id string, creds bitnode.Credentials, val bitnode.HubItem) {
		clientTicks++
	}))

	tickHub := tickerSys.GetHub("tick")
	tickHub.Subscribe(bitnode.NewNativeSubscription(func(id string, creds bitnode.Credentials, val bitnode.HubItem) {
		serverTicks++
	}))

	time.Sleep(500 * time.Millisecond)

	if clientTicks == 0 {
		t.Fatal()
	}
}

func TestChangeMeta__Name(t *testing.T) {
	snode, sconns, _ := testClient(t, "", "ws://127.0.0.1:21145")
	_, cconns, _ := testClient(t, "", "")

	creds := bitnode.Credentials{}

	srv := wsApi.NewServer(sconns, "0.0.0.0:21145")
	defer srv.Shutdown(context.Background())
	go srv.Listen()

	time.Sleep(250 * time.Millisecond)

	sys1, _ := snode.NewSystem(creds, bitnode.Sparkable{})

	if _, err := cconns.ConnectNode("ws://127.0.0.1:21145"); err != nil {
		t.Fatal(err)
	}

	clt1, _ := cconns.GetNodeByName(snode.Name()).AddClient()
	_ = clt1.Connect(sys1.ID(), bitnode.Credentials{})

	if clt1.RemoteName() != sys1.Name() {
		t.Fatal(clt1.Name(), sys1.Name())
	}

	time.Sleep(250 * time.Millisecond)

	sys1.SetName("test1")

	time.Sleep(250 * time.Millisecond)

	if clt1.RemoteName() != "test1" {
		t.Fatal()
	}

	clt1.SetName("test2")

	time.Sleep(250 * time.Millisecond)

	if sys1.Name() != "test2" {
		t.Fatal()
	}
}

func TestChangeMeta__Status1(t *testing.T) {
	snode, sconns, _ := testClient(t, "", "ws://127.0.0.1:21145")
	_, cconns, _ := testClient(t, "", "")

	creds := bitnode.Credentials{}

	srv := wsApi.NewServer(sconns, "0.0.0.0:21145")
	defer srv.Shutdown(context.Background())
	go srv.Listen()

	time.Sleep(250 * time.Millisecond)

	sys1, _ := snode.NewSystem(creds, bitnode.Sparkable{})

	if _, err := cconns.ConnectNode("ws://127.0.0.1:21145"); err != nil {
		t.Fatal(err)
	}

	clt1, _ := cconns.GetNodeByName(snode.Name()).AddClient()
	_ = clt1.Connect(sys1.ID(), bitnode.Credentials{})

	if clt1.RemoteName() != sys1.Name() {
		t.Fatal(clt1.Name(), sys1.Name())
	}

	time.Sleep(250 * time.Millisecond)

	sys1.SetStatus(1)

	time.Sleep(250 * time.Millisecond)

	if clt1.Status() != 1 {
		t.Fatal(clt1.Status())
	}

	clt1.SetStatus(2)

	time.Sleep(250 * time.Millisecond)

	// Status and messages cannot go back to the original system.

	if sys1.Status() != 1 {
		t.Fatal(sys1.Status())
	}
}

func TestNodeClientShutdown__NoStore(t *testing.T) {
	snode, sconns, dom := testClient(t, "", "ws://127.0.0.1:12335")
	_, cconns, _ := testClient(t, "", "")

	srv := wsApi.NewServer(sconns, "0.0.0.0:12335")
	go srv.Listen()

	time.Sleep(500 * time.Millisecond)

	bp, err := dom.GetSparkable("hub.time.Clock")
	if err != nil {
		t.Fatal(err)
	}

	creds := bitnode.Credentials{}

	sys, err := snode.NewSystem(creds, *bp)
	if err != nil {
		t.Fatal(err)
	}
	if sys == nil {
		t.Fatal()
	}

	if _, err := cconns.ConnectNode("ws://127.0.0.1:12335"); err != nil {
		t.Fatal(err)
	}

	clt, err := cconns.GetNodeByName(snode.Name()).AddClient()
	if err != nil {
		t.Fatal(err)
	}
	if err := clt.Connect(sys.ID(), bitnode.Credentials{}); err != nil {
		t.Fatal(err)
	}

	hub := clt.GetHub("getTimestamp")
	if hub == nil {
		t.Fatal()
	}

	ret, err := hub.Invoke(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ret) != 1 || ret[0].(float64) == 0 {
		t.Fatal(ret)
	}

	// Now, shutdown the server.

	srv.Shutdown(context.Background())

	time.Sleep(250 * time.Millisecond)

	ret, err = hub.Invoke(nil)
	if err == nil || !strings.Contains(err.Error(), "client inactive") {
		t.Fatal("expected inactive client, got", err)
	}

	// Now, restart the server.

	srv = wsApi.NewServer(sconns, "0.0.0.0:12335")
	go srv.Listen()

	time.Sleep(1000 * time.Millisecond)

	// ...and try again

	ret, err = hub.Invoke(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ret) != 1 || ret[0].(float64) == 0 {
		t.Fatal(ret)
	}
}

func TestNodeClientShutdown__RestartWithStore__Server(t *testing.T) {
	snode, sconns, dom := testClient(t, "", "ws://127.0.0.1:12295")
	_, cconns, _ := testClient(t, "", "")

	srv := wsApi.NewServer(sconns, "0.0.0.0:12295")
	go srv.Listen()

	time.Sleep(500 * time.Millisecond)

	bp, err := dom.GetSparkable("hub.time.Clock")
	if err != nil {
		t.Fatal(err)
	}

	creds := bitnode.Credentials{}

	sys, err := snode.NewSystem(creds, *bp)
	if err != nil {
		t.Fatal(err)
	}
	if sys == nil {
		t.Fatal()
	}

	if _, err := cconns.ConnectNode("ws://127.0.0.1:12295"); err != nil {
		t.Fatal(err)
	}

	clt, err := cconns.GetNodeByName(snode.Name()).AddClient()
	if err != nil {
		t.Fatal(err)
	}
	if err := clt.Connect(sys.ID(), bitnode.Credentials{}); err != nil {
		t.Fatal(err)
	}

	hub := clt.GetHub("getTimestamp")
	if hub == nil {
		t.Fatal()
	}

	ret, err := hub.Invoke(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ret) != 1 || ret[0].(float64) == 0 {
		t.Fatal(ret)
	}

	// Now, shutdown the server.

	srv.Shutdown(context.Background())

	// Store the node.
	st := store.NewStore("test")
	if err := snode.Store(st); err != nil {
		t.Fatal(err)
	}

	snode2, sconns2, dom2 := testClient(t, "", "ws://127.0.0.1:12295")

	// Load the node.
	if err := snode2.Load(st, dom2); err != nil {
		t.Fatal(err)
	}

	srv2 := wsApi.NewServer(sconns2, "0.0.0.0:12295")
	go srv2.Listen()

	time.Sleep(3 * time.Second)

	// ...and try again

	ret, err = hub.Invoke(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ret) != 1 || ret[0].(float64) == 0 {
		t.Fatal(ret)
	}
}

func TestNodeClientShutdown__RestartWithStore__Client(t *testing.T) {
	snode, sconns, dom := testClient(t, "", "ws://127.0.0.1:12285")
	cnode, cconns, _ := testClient(t, "", "")

	srv := wsApi.NewServer(sconns, "0.0.0.0:12285")
	go srv.Listen()

	time.Sleep(500 * time.Millisecond)

	bp, err := dom.GetSparkable("hub.time.Clock")
	if err != nil {
		t.Fatal(err)
	}

	creds := bitnode.Credentials{}

	sys, err := snode.NewSystem(creds, *bp)
	if err != nil {
		t.Fatal(err)
	}
	if sys == nil {
		t.Fatal()
	}

	if _, err := cconns.ConnectNode("ws://127.0.0.1:12285"); err != nil {
		t.Fatal(err)
	}

	clt, err := cconns.GetNodeByName(snode.Name()).AddClient()
	if err != nil {
		t.Fatal(err)
	}
	if err := clt.Connect(sys.ID(), bitnode.Credentials{}); err != nil {
		t.Fatal(err)
	}

	hub := clt.GetHub("getTimestamp")
	if hub == nil {
		t.Fatal()
	}

	ret, err := hub.Invoke(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ret) != 1 || ret[0].(float64) == 0 {
		t.Fatal(ret)
	}

	// Store the node.
	st := store.NewStore("test")
	if err := cnode.Store(st); err != nil {
		t.Fatal(err)
	}

	if err := cconns.Shutdown(); err != nil {
		t.Fatal(err)
	}

	cnode2, cconns2, dom2 := testClient(t, "", "")

	if _, err := cconns2.ConnectNode("ws://127.0.0.1:12285"); err != nil {
		t.Fatal(err)
	}

	// Load the node.
	if err := cnode2.Load(st, dom2); err != nil {
		t.Fatal(err)
	}

	time.Sleep(1000 * time.Millisecond)

	// ...and try again

	clt2, err := cnode2.GetSystemByID(bitnode.Credentials{}, clt.ID())
	if err != nil {
		t.Fatal(err)
	}

	hub2 := clt2.GetHub("getTimestamp")

	ret, err = hub2.Invoke(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ret) != 1 || ret[0].(float64) == 0 {
		t.Fatal(ret)
	}
}
