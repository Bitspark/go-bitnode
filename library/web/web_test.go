package libWeb

import (
	"github.com/Bitspark/go-bitnode/bitnode"
	"testing"
)

func testNode(t *testing.T, dir string) (*bitnode.NativeNode, *bitnode.Domain) {
	h := bitnode.NewNode()
	if err := h.AddFactory(&WebFactory{}); err != nil {
		t.Fatal(err)
	}
	testDom, _ := bitnode.NewDomain().AddDomain("web")
	if err := testDom.LoadFromDir(dir, false); err != nil {
		t.Fatal(err)
	}
	if err := testDom.Compile(); err != nil {
		t.Fatal(err)
	}
	return h, testDom
}

func TestHTTPClientGET1(t *testing.T) {
	h, dom := testNode(t, "./")

	creds := bitnode.Credentials{}

	httpClientImpl, err := dom.GetSparkable("HTTPTextClient")
	if err != nil {
		t.Fatal(err)
	}
	if httpClientImpl == nil {
		t.Fatal()
	}

	httpClientSys, err := h.NewSystem(creds, *httpClientImpl)
	if err != nil {
		t.Fatal(err)
	}

	hub := httpClientSys.GetHub("getText")
	resp, err := hub.Invoke(nil, "https://google.com", []map[string]bitnode.HubItem{})
	if err != nil {
		t.Fatal(err)
	}
	if resp[1].(int64) != 200 {
		t.Fatal(resp[1])
	}
}

func TestHTTPClientPOST1(t *testing.T) {
	h, dom := testNode(t, "./")

	creds := bitnode.Credentials{}

	httpClientImpl, err := dom.GetSparkable("HTTPTextClient")
	if err != nil {
		t.Fatal(err)
	}
	if httpClientImpl == nil {
		t.Fatal()
	}

	httpClientSys, err := h.NewSystem(creds, *httpClientImpl)
	if err != nil {
		t.Fatal(err)
	}

	hub := httpClientSys.GetHub("postText")
	resp, err := hub.Invoke(nil, "https://google.com", "{}", []map[string]bitnode.HubItem{})
	if err != nil {
		t.Fatal(err)
	}
	if resp[1].(int64) != 405 {
		t.Fatal(resp[1])
	}
}
