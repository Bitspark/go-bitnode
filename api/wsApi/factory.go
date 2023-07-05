package wsApi

import (
	"encoding/json"
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
	"github.com/Bitspark/go-bitnode/store"
	"github.com/gorilla/websocket"
	"sync"
	"time"
)

const heartbeatInterval = 50 * time.Second

const ApiVersion = "1.0"

const Bitnode = "go:1.0"

type WSFactory struct {
	node          bitnode.Node
	conns         map[string]*Conn
	connsMux      sync.Mutex
	queuedClients map[string]map[string]*Client
	queueMux      sync.Mutex
	address       string
	shutdown      bool
}

var _ bitnode.Factory = &WSFactory{}

func NewWSFactory(node bitnode.Node, address string) *WSFactory {
	return &WSFactory{
		node:          node,
		conns:         map[string]*Conn{},
		queuedClients: map[string]map[string]*Client{},
		address:       address,
	}
}

func (f *WSFactory) Name() string {
	return "ws"
}

func (f *WSFactory) Implementation(impl bitnode.Implementation) (bitnode.Implementation, error) {
	if impl == nil {
		return &clientImpl{
			factory: f,
		}, nil
	}
	nImpl, ok := impl.(*clientImpl)
	if !ok {
		return nil, fmt.Errorf("not a ws implementation")
	} else {
		nImpl.factory = f
	}
	return nImpl, nil
}

type clientImpl struct {
	CID      string              `json:"cid" yaml:"cid"`
	Node     string              `json:"node" yaml:"node"`
	RemoteID bitnode.SystemID    `json:"remoteId" yaml:"remoteId"`
	Creds    bitnode.Credentials `json:"credentials" yaml:"credentials"`
	Server   bool                `json:"server" yaml:"server"`
	factory  *WSFactory
}

func (c *clientImpl) FromInterface(a any) error {
	ciBts, _ := json.Marshal(a)
	return json.Unmarshal(ciBts, c)
}

func (c *clientImpl) ToInterface() (any, error) {
	return c, nil
}

func (c *clientImpl) Implement(node *bitnode.NativeNode, sys bitnode.System) error {
	_, err := c.factory.ReconnectClient(c.Node, c.CID, c.RemoteID, c.Creds, sys.Native(), c.Server)
	return err
}

func (c *clientImpl) Extend(node *bitnode.NativeNode, ext bitnode.Implementation) (bitnode.Implementation, error) {
	return c, nil
}

func (c *clientImpl) Validate() error {
	return nil
}

type ClientExt struct {
	Client    *Client
	Connected bool
}

func (c ClientExt) Implementation() bitnode.Implementation {
	return &clientImpl{
		CID:      c.Client.cid,
		Node:     c.Client.remoteNode,
		RemoteID: c.Client.remoteID,
		Creds:    c.Client.creds,
		Server:   c.Client.server,
	}
}

var _ bitnode.SystemExtension = &ClientExt{}

func (f *WSFactory) Load(st store.Store, dom *bitnode.Domain) error {
	nodeDS, err := st.Ensure("node", store.DSStores)
	if err != nil {
		return err
	}
	nodeStore := nodeDS.Stores()

	nodeSt, err := nodeStore.Get("node")
	if err != nil {
		return err
	}

	if err := f.node.Load(nodeSt, dom); err != nil {
		return err
	}

	propertiesDS, err := st.Ensure("properties", store.DSKeyValue)
	if err != nil {
		return err
	}
	propertiesStore := propertiesDS.KeyValue()

	f.address, _ = propertiesStore.Get("address")

	connsDS, err := st.Ensure("conns", store.DSStores)
	if err != nil {
		return err
	}
	conssStore := connsDS.Stores()

	for connStore := range conssStore.Enumerate() {
		nconn := &Conn{
			factory: f,
			clients: map[string]*Client{},
			refs:    map[string]*NodeRefChan{},
		}
		if err := nconn.Load(connStore); err != nil {
			return err
		}
		if nconn.remoteAddress != "" {
			go nconn.reconnectNode()
		} else {
			nconn.active = false
		}
	}

	return nil
}

func (f *WSFactory) Store(st store.Store) error {
	propertiesDS, err := st.Ensure("properties", store.DSKeyValue)
	if err != nil {
		return err
	}
	propertiesStore := propertiesDS.KeyValue()

	if err := propertiesStore.Set("address", f.address); err != nil {
		return err
	}

	connsDS, err := st.Ensure("conns", store.DSStores)
	if err != nil {
		return err
	}
	conssStore := connsDS.Stores()

	f.connsMux.Lock()
	for _, conn := range f.conns {
		connSt := store.NewStore(conn.node)
		if err := conn.Store(connSt); err != nil {
			return err
		}
		if err := conssStore.Add(connSt); err != nil {
			return err
		}
	}
	f.connsMux.Unlock()

	nodeDS, err := st.Ensure("node", store.DSStores)
	if err != nil {
		return err
	}
	nodeStore := nodeDS.Stores()

	nodeSt := store.NewStore("node")

	if err := f.node.Store(nodeSt); err != nil {
		return err
	}

	if err := nodeStore.Add(nodeSt); err != nil {
		return err
	}

	return nil
}

func (f *WSFactory) ConnectNode(addr string) (*Conn, error) {
	nconn := &Conn{
		factory:       f,
		clients:       map[string]*Client{},
		refs:          map[string]*NodeRefChan{},
		remoteAddress: addr,
	}
	if err := nconn.connectNode(); err != nil {
		return nil, err
	}
	return nconn, nil
}

func (f *WSFactory) AcceptNode(conn *websocket.Conn) (*Conn, error) {
	nconn := &Conn{
		factory: f,
		clients: map[string]*Client{},
		refs:    map[string]*NodeRefChan{},
		ws:      conn,
	}
	return nconn, nil
}

func (f *WSFactory) Node() bitnode.Node {
	return f.node
}

func (f *WSFactory) GetNodeByName(name string) *Conn {
	f.connsMux.Lock()
	defer f.connsMux.Unlock()
	for _, conn := range f.conns {
		if conn.node == name {
			return conn
		}
	}
	return nil
}

func (f *WSFactory) GetNodeByAddress(addr string) *Conn {
	f.connsMux.Lock()
	defer f.connsMux.Unlock()
	for _, conn := range f.conns {
		if conn.remoteAddress == addr {
			return conn
		}
	}
	return nil
}

func (f *WSFactory) ReconnectClient(node string, cid string, remoteID bitnode.SystemID, creds bitnode.Credentials, native *bitnode.NativeSystem, server bool) (*Client, error) {
	f.connsMux.Lock()
	if conn, ok := f.conns[node]; !ok {
		f.connsMux.Unlock()
		cl := &Client{
			NativeSystem: native,
			cid:          cid,
			remoteNode:   node,
			remoteID:     remoteID,
			created:      time.Now(),
			server:       server,
			incomingIDs:  map[string]bool{},
			creds:        creds,
			middlewares:  f.node.Middlewares(),
			attached:     false,
			defined:      true,
		}

		native.SetExtension("ws", &ClientExt{Client: cl})

		f.queueMux.Lock()
		queuedClients, _ := f.queuedClients[node]
		f.queueMux.Unlock()

		if queuedClients == nil {
			queuedClients = map[string]*Client{}
		}
		queuedClients[cl.cid] = cl

		f.queueMux.Lock()
		f.queuedClients[node] = queuedClients
		f.queueMux.Unlock()

		return cl, nil
	} else {
		f.connsMux.Unlock()
		var cl *Client

		conn.clientsMux.Lock()
		if cl, ok = conn.clients[cid]; ok {
			conn.clientsMux.Unlock()
			cl.conn = conn
			cl.remoteNode = node
			cl.creds = creds
			cl.NativeSystem = native
		} else {
			conn.clientsMux.Unlock()
			cl = &Client{
				NativeSystem: native,
				cid:          cid,
				conn:         conn,
				remoteNode:   node,
				remoteID:     remoteID,
				created:      time.Now(),
				server:       server,
				incomingIDs:  map[string]bool{},
				creds:        creds,
				middlewares:  f.node.Middlewares(),
				attached:     false,
				defined:      true,
			}
			conn.clientsMux.Lock()
			conn.clients[cl.cid] = cl
			conn.clientsMux.Unlock()
		}

		native.SetExtension("ws", &ClientExt{Client: cl})

		if !cl.server {
			if err := conn.connectClient(cl); err != nil {
				return nil, err
			}
			if err := cl.connect(); err != nil {
				return nil, err
			}
		}

		return cl, nil
	}
}

func (f *WSFactory) AcceptClient(node string, cid string) (*Client, error) {
	f.connsMux.Lock()
	if h, ok := f.conns[node]; !ok {
		f.connsMux.Unlock()
		return nil, fmt.Errorf("node not found: %s", node)
	} else {
		f.connsMux.Unlock()

		//sys, err := f.node.BlankSystem("")
		//if err != nil {
		//	return nil, err
		//}
		cl := &Client{
			//NativeSystem: sys,
			cid:         cid,
			created:     time.Now(),
			remoteNode:  node,
			conn:        h,
			server:      true,
			incomingIDs: map[string]bool{},
			middlewares: f.node.Middlewares(),
		}
		//cl.SetExtension("ws", &ClientExt{Client: cl})
		//orig, _ := cl.conn.factory.node.BlankSystem("")
		//cl.AddOrigin("ws", orig)

		h.clientsMux.Lock()
		h.clients[cl.cid] = cl
		h.clientsMux.Unlock()
		return cl, nil
	}
}

func (f *WSFactory) Shutdown() error {
	f.shutdown = true
	f.connsMux.Lock()
	defer f.connsMux.Unlock()
	for _, c := range f.conns {
		c.wsMux.Lock()
		if c.ws != nil {
			c.ws.Close()
		}
		c.wsMux.Unlock()
	}
	return nil
}
