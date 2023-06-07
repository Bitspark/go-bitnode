package wsApi

import (
	"context"
	"github.com/Bitspark/go-bitnode/bitnode"
	"github.com/Bitspark/go-bitnode/store"
	"testing"
	"time"
)

func TestConn1(t *testing.T) {
	node1 := bitnode.NewNode()
	node2 := bitnode.NewNode()

	conns1 := NewNodeConns(node1, "ws://127.0.0.1:12340")
	conns2 := NewNodeConns(node2, "")

	server1 := NewServer(conns1, "0.0.0.0:12340")
	defer server1.Shutdown(context.Background())
	go server1.Listen()

	time.Sleep(200 * time.Millisecond)

	conn2, err := conns2.ConnectNode("ws://127.0.0.1:12340")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	if len(conns1.conns) != 1 {
		t.Fatal()
	}
	if len(conns2.conns) != 1 {
		t.Fatal()
	}

	if conn2.node != node1.Name() {
		t.Fatal()
	}
	if conns1.conns[conns2.node.Name()].node != node2.Name() {
		t.Fatal()
	}
	if conns2.conns[conns1.node.Name()].node != node1.Name() {
		t.Fatal()
	}

	if !conns1.conns[conns2.node.Name()].active {
		t.Fatal()
	}
	if !conns2.conns[conns1.node.Name()].active {
		t.Fatal()
	}
}

func TestReConn1(t *testing.T) {
	node1 := bitnode.NewNode()
	node2 := bitnode.NewNode()

	conns1 := NewNodeConns(node1, "ws://127.0.0.1:12340")
	conns2 := NewNodeConns(node2, "")

	server1 := NewServer(conns1, "0.0.0.0:12340")
	go server1.Listen()

	time.Sleep(200 * time.Millisecond)

	_, err := conns2.ConnectNode("ws://127.0.0.1:12340")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	// Now, everything should be connected
	// We shutdown a server now

	t.Log(server1.Shutdown(context.Background()))

	time.Sleep(100 * time.Millisecond)

	// So, node1 should be inactive now

	if conns2.conns[node1.Name()].active {
		t.Fatal()
	}
	if conns1.conns[node2.Name()].active {
		t.Fatal()
	}
}

func TestReConn2(t *testing.T) {
	node1 := bitnode.NewNode()
	node2 := bitnode.NewNode()

	conns1 := NewNodeConns(node1, "ws://127.0.0.1:22340")
	conns2 := NewNodeConns(node2, "")

	server1 := NewServer(conns1, "0.0.0.0:22340")
	go server1.Listen()

	time.Sleep(200 * time.Millisecond)

	_, err := conns2.ConnectNode("ws://127.0.0.1:22340")
	if err != nil {
		t.Fatal(err)
	}

	st := store.NewStore("node2")

	if err := conns2.Store(st); err != nil {
		t.Fatal(err)
	}

	node3 := bitnode.NewNode()
	conns3 := NewNodeConns(node3, "")

	if err := conns3.Load(st, nil); err != nil {
		t.Fatal(err)
	}

	time.Sleep(1500 * time.Millisecond)

	if len(conns3.conns) != 1 {
		t.Fatal(conns3.conns)
	}
	if conns3.conns[conns1.node.Name()] == nil {
		t.Fatal()
	}
}
