package factories

import (
	"github.com/Bitspark/go-bitnode/bitnode"
	"testing"
)

func testNodeJS(t *testing.T, dir string) (*bitnode.NativeNode, *bitnode.Domain) {
	h := bitnode.NewNode()
	dom := bitnode.NewDomain()
	h.AddMiddlewares(GetMiddlewares())
	_ = h.AddFactory(NewJSFactory(dom))
	testDom, _ := dom.AddDomain("test")
	if err := testDom.LoadFromDir(dir, false); err != nil {
		t.Fatal(err)
	}
	if err := testDom.Compile(); err != nil {
		t.Fatal(err)
	}
	return h, testDom
}

func TestJSImpl1(t *testing.T) {
	h, dom := testNodeJS(t, "./javascript_test/impl1")

	doublerImpl, err := dom.GetSparkable("Doubler")
	if err != nil {
		t.Fatal(err)
	}
	if doublerImpl == nil {
		t.Fatal()
	}

	if err := doublerImpl.Compile(dom, dom.FullName, true); err != nil {
		t.Fatal(err)
	}

	doublerSys, err := h.NewSystem(bitnode.Credentials{}, *doublerImpl)
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan bool)
	doublerSys.GetHub("out").Subscribe(bitnode.NewNativeSubscription(func(id string, creds bitnode.Credentials, val bitnode.HubItem) {
		if err != nil {
			t.Fatal(err)
		}
		if val != 8.2 {
			t.Fatal(val)
		}
		done <- true
	}))

	go func() {
		inHub := doublerSys.GetHub("in")
		err = inHub.Push("", 4.1)
		if err != nil {
			t.Fatal(err)
		}
	}()

	<-done
}

func TestJSImpl2(t *testing.T) {
	h, dom := testNodeJS(t, "./javascript_test/impl2")

	use1Impl, err := dom.GetSparkable("UseSquareRoot1")
	if err != nil {
		t.Fatal(err)
	}
	use1Sys, err := h.NewSystem(bitnode.Credentials{}, *use1Impl)
	if err != nil {
		t.Fatal(err)
	}
	if use1Sys == nil {
		t.Fatal(err)
	}

	bitnode.WaitFor(use1Sys, bitnode.SystemStatusRunning)

	vals, err := use1Sys.GetHub("pipe").Invoke(nil, 9)
	if err != nil {
		t.Fatal(err)
	}
	if vals[0] != 4.0 {
		t.Fatal(vals)
	}

	use2Impl, _ := dom.GetSparkable("UseSquareRoot2")
	use2Sys, err := h.NewSystem(bitnode.Credentials{}, *use2Impl)
	if err != nil {
		t.Fatal(err)
	}
	if use1Sys == nil {
		t.Fatal(err)
	}

	bitnode.WaitFor(use2Sys, bitnode.SystemStatusRunning)

	vals, err = use2Sys.GetHub("sqrt").Invoke(nil, 9)
	if err != nil {
		t.Fatal(err)
	}
	if vals[0] != 3.0 {
		t.Fatal(vals)
	}
	if vals[1] != 3.0 {
		t.Fatal(vals)
	}
}

type testConsole struct {
	msgs []string
}

func (t *testConsole) Print(msg string) {
	t.msgs = append(t.msgs, msg)
}

func TestJSHelloWorld(t *testing.T) {
	h, dom := testNodeJS(t, "./javascript_test/helloWorld")

	hwImpl, err := dom.GetSparkable("HelloWorld")
	if err != nil {
		t.Fatal(err)
	}
	if hwImpl == nil {
		t.Fatal()
	}

	tc := &testConsole{}
	hwSys, err := h.NewSystem(bitnode.Credentials{}, *hwImpl, tc)
	if err != nil {
		t.Fatal(err)
	}
	if hwSys == nil {
		t.Fatal()
	}

	bitnode.WaitFor(hwSys, bitnode.SystemStatusRunning)

	if len(tc.msgs) == 0 {
		t.Fatal()
	}
	if tc.msgs[0] != "Hello World!" {
		t.Fatal(tc)
	}
}

func TestJSConstr(t *testing.T) {
	h, dom := testNodeJS(t, "./javascript_test/constr")

	hwImpl, err := dom.GetSparkable("Constr")
	if err != nil {
		t.Fatal(err)
	}
	if hwImpl == nil {
		t.Fatal()
	}

	tc := &testConsole{}
	hwSys, err := h.NewSystem(bitnode.Credentials{}, *hwImpl, tc, 5)
	if err != nil {
		t.Fatal(err)
	}
	if hwSys == nil {
		t.Fatal()
	}

	bitnode.WaitFor(hwSys, bitnode.SystemStatusRunning)

	if len(tc.msgs) == 0 {
		t.Fatal()
	}
	if tc.msgs[0] != "My value is 5" {
		t.Fatal(tc)
	}
}
