package wsApi

import (
	"encoding/json"
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
	"github.com/Bitspark/go-bitnode/store"
	"github.com/Bitspark/go-bitnode/util"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"net/url"
	"sync"
	"time"
)

// A Conn represents a connection to another node.
type Conn struct {
	factory         *WSFactory
	node            string
	version         string
	bitnode         string
	remoteAddress   string
	clientsMux      sync.Mutex
	clients         map[string]*Client
	refs            map[string]*NodeRefChan
	refsMux         sync.Mutex
	active          bool
	ws              *websocket.Conn
	wsMux           sync.Mutex
	remoteBeatCount int64
	remoteBeatTime  float64
	beatCount       int64
}

func (c *Conn) Handle() {
	for {
		if hmsg, err := c.Receive(); err != nil {
			c.Log(bitnode.LogError, err.Error())
			break
		} else {
			go func() {
				if err := hmsg.Handle(c); err != nil {
					c.Log(bitnode.LogError, err.Error())
					if hmsg.Request != "" {
						c.SendError(err, hmsg.Request)
					}
				}
			}()
		}
	}
	_ = c.ws.Close()
	c.Log(bitnode.LogError, fmt.Sprintf("Disconnected node: %s", c.node))
	c.active = false

	clients := []*Client{}
	c.clientsMux.Lock()
	for _, client := range c.clients {
		clients = append(clients, client)
	}
	c.clientsMux.Unlock()
	for _, client := range clients {
		if client.NativeSystem == nil || client.server {
			continue
		}
		_ = client.Native().EmitEvent(bitnode.LifecycleStop, 0.0)
	}

	if c.remoteAddress == "" {
		// TODO: Here, we need to stand by still, in order to reconnectNode later!
		// This is also what we should do if the node could not be found
	} else {
		go c.reconnectNode()
	}
}

func (c *Conn) Heartbeat(interval time.Duration) {
	for {
		c.beatCount++
		c.Send("heartbeat", &NodePayloadHeartbeat{
			Count: c.beatCount,
			Time:  float64(time.Now().UnixMicro()) / 1000000,
		}, "", false)
		time.Sleep(interval)
	}
}

func (c *Conn) Node() string {
	return c.node
}

func (c *Conn) AddClient() (*Client, error) {
	sys, err := c.factory.node.BlankSystem("")
	if err != nil {
		return nil, err
	}
	cl := &Client{
		NativeSystem: sys,
		cid:          util.RandomString(util.CharsAlphaNum, 8),
		created:      time.Now(),
		remoteNode:   c.node,
		conn:         c,
		server:       false,
		incomingIDs:  map[string]bool{},
		middlewares:  c.factory.node.Middlewares(),
	}
	cl.SetExtension("ws", &WSExt{Client: cl})

	orig, _ := cl.conn.factory.node.BlankSystem("")
	cl.AddOrigin("ws", orig)

	c.clientsMux.Lock()
	c.clients[cl.cid] = cl
	c.clientsMux.Unlock()
	if err := c.connectClient(cl); err != nil {
		return nil, err
	}
	return cl, nil
}

func (c *Conn) Send(cmd string, hmsg NodePayload, reference string, returns bool) *NodeRefChan {
	var ret string
	if reference == "" && returns {
		ret = util.RandomString(util.CharsAlphaNum, 8)
	}
	ch := &NodeRefChan{cmd: cmd, ch: make(chan NodePayload)}
	c.refsMux.Lock()
	c.refs[ret] = ch
	c.refsMux.Unlock()
	chSent := make(chan bool)
	go func(ch chan bool) {
		defer close(ch)
		msgBts, _ := json.Marshal(NodeMessage{
			Cmd:       cmd,
			Request:   ret,
			Reference: reference,
			Payload:   hmsg,
		})
		sent := false
		c.wsMux.Lock()
		if c.ws != nil {
			_ = c.ws.WriteMessage(websocket.TextMessage, msgBts)
			sent = true
		}
		c.wsMux.Unlock()
		ch <- sent
	}(chSent)
	<-chSent
	return ch
}

func (c *Conn) SendError(err error, reference string) {
	msgBts, _ := json.Marshal(NodeMessage{
		Cmd:       "error",
		Reference: reference,
		Payload: &NodePayloadError{
			Error: err.Error(),
		},
	})
	c.wsMux.Lock()
	_ = c.ws.WriteMessage(websocket.TextMessage, msgBts)
	c.wsMux.Unlock()
}

func (c *Conn) Receive() (*NodeMessage, error) {
	msg := &NodeMessage{}
	msgType, msgBts, err := c.ws.ReadMessage()
	if err != nil {
		return nil, err
	}
	switch msgType {
	case websocket.TextMessage:
		if err := json.Unmarshal(msgBts, msg); err != nil {
			return nil, err
		}
		return msg, nil

	case websocket.BinaryMessage:
		return nil, nil

	case websocket.CloseMessage:
		return nil, fmt.Errorf("closed")
	}
	return nil, nil
}

func (c *Conn) Log(code int, msg string) {
	log.Printf("[%s-%d] %s", c.node, code, msg)
}

func (c *Conn) Load(st store.Store) error {
	connPropsDS, err := st.Ensure("conn", store.DSKeyValue)
	if err != nil {
		return err
	}
	connPropsStore := connPropsDS.KeyValue()
	c.remoteAddress, _ = connPropsStore.Get("remoteAddress")
	c.node, _ = connPropsStore.Get("node")
	c.version, _ = connPropsStore.Get("version")
	c.bitnode, _ = connPropsStore.Get("bitnode")
	return nil
}

func (c *Conn) Store(st store.Store) error {
	connPropsDS, err := st.Ensure("conn", store.DSKeyValue)
	if err != nil {
		return err
	}
	connPropsStore := connPropsDS.KeyValue()
	if err := connPropsStore.Set("remoteAddress", c.remoteAddress); err != nil {
		return err
	}
	if err := connPropsStore.Set("node", c.node); err != nil {
		return err
	}
	if err := connPropsStore.Set("version", c.version); err != nil {
		return err
	}
	if err := connPropsStore.Set("bitnode", c.bitnode); err != nil {
		return err
	}
	return nil
}

func (c *Conn) connectNode() error {
	u, err := url.Parse(c.remoteAddress + wsPath)
	if err != nil {
		return err
	}
	c.wsMux.Lock()
	c.ws, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	c.wsMux.Unlock()
	if err != nil {
		return err
	}

	go c.Handle()
	go c.Heartbeat(heartbeatInterval)

	ref := c.Send("handshake", &NodePayloadHandshake{
		Version: ApiVersion,
		Bitnode: Bitnode,
		Node:    c.factory.node.Name(),
	}, "", true)
	<-ref.ch

	c.active = true

	return nil
}

func (c *Conn) connectClient(cl *Client) error {
	ch := c.Send("new_client", &NodePayloadNewClient{
		Client: cl.cid,
	}, "", true)
	ret := <-ch.ch
	if err, _ := ret.(error); err != nil {
		return err
	}
	return nil
}

// reconnectNode tries to establish a connection to the node again.
func (c *Conn) reconnectNode() {
	wait := 1 * time.Millisecond
	for {
		if c.factory.shutdown {
			return
		}

		time.Sleep(wait)

		if err := c.connectNode(); err == nil {
			if err := c.reconnectClients(); err != nil {
				log.Println(err)
			}
			return
		} else {
			log.Println(err)
		}

		wait = time.Duration(float64(wait)*(rand.Float64()+1)) + 1*time.Millisecond
	}
}

func (c *Conn) reconnectClients() error {
	clients := []*Client{}
	c.clientsMux.Lock()
	for _, cl := range c.clients {
		clients = append(clients, cl)
	}
	c.clientsMux.Unlock()
	for _, cl := range clients {
		if cl.server {
			continue
		}
		if err := c.connectClient(cl); err != nil {
			return err
		}
		if err := cl.connect(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Conn) takeOver(econn *Conn) {
	econn.clientsMux.Lock()
	if c != econn {
		c.clientsMux.Lock()
	}
	c.clients = econn.clients
	for _, cl := range c.clients {
		cl.conn = c
	}
	if c != econn {
		c.clientsMux.Unlock()
	}
	econn.clientsMux.Unlock()
}

// A NodeMessage is a SystemMessage sent from a node to another node.
type NodeMessage struct {
	Cmd       string      `json:"cmd,omitempty"`
	Request   string      `json:"request,omitempty"`
	Reference string      `json:"reference,omitempty"`
	Payload   NodePayload `json:"payload,omitempty"`
}

func (hm *NodeMessage) Handle(nconn *Conn) error {
	var err error
	if hm.Payload != nil {
		err = hm.Payload.Handle(nconn, hm.Request)
	}
	if hm.Reference != "" {
		nconn.refsMux.Lock()
		ch := nconn.refs[hm.Reference]
		delete(nconn.refs, hm.Reference)
		nconn.refsMux.Unlock()
		if ch != nil {
			if ch.ch != nil {
				go func() {
					ch.ch <- hm.Payload
					close(ch.ch)
				}()
			}
		} else {
			log.Printf("reference not found: %v %v", hm, hm.Payload)
		}
	}
	return err
}

func (hm *NodeMessage) MarshalJSON() ([]byte, error) {
	var hms struct {
		Cmd       string          `json:"cmd"`
		Request   string          `json:"request,omitempty"`
		Reference string          `json:"reference,omitempty"`
		Payload   json.RawMessage `json:"payload,omitempty"`
	}
	hms.Cmd = hm.Cmd
	hm.Request = hms.Request
	hm.Reference = hms.Reference
	hms.Payload, _ = json.Marshal(hm.Payload)
	return json.Marshal(hms)
}

func (hm *NodeMessage) UnmarshalJSON(data []byte) error {
	var hms struct {
		Cmd       string          `json:"cmd"`
		Request   string          `json:"request,omitempty"`
		Reference string          `json:"reference,omitempty"`
		Payload   json.RawMessage `json:"payload,omitempty"`
	}
	if err := json.Unmarshal(data, &hms); err != nil {
		return err
	}
	hm.Cmd = hms.Cmd
	hm.Request = hms.Request
	hm.Reference = hms.Reference

	switch hms.Cmd {
	case "error":
		hm.Payload = &NodePayloadError{}
	case "handshake":
		hm.Payload = &NodePayloadHandshake{}
	case "heartbeat":
		hm.Payload = &NodePayloadHeartbeat{}
	case "new_client":
		hm.Payload = &NodePayloadNewClient{}
	case "client":
		hm.Payload = &NodePayloadClient{}
	default:
		if hm.Cmd == "" {
			return nil
		}
		return fmt.Errorf("unknown command: %s", hms.Cmd)
	}

	return json.Unmarshal(hms.Payload, hm.Payload)
}

type NodePayload interface {
	Handle(nconn *Conn, reference string) error
}

type NodePayloadHandshake struct {
	Version string `json:"version"`
	Bitnode string `json:"bitnode"`
	Node    string `json:"node"`
	Address string `json:"address"`
}

func (p *NodePayloadHandshake) Handle(nconn *Conn, reference string) error {
	if p.Version != ApiVersion {
		return fmt.Errorf("unsupported version: %s", p.Version)
	}
	if p.Bitnode == "" {
		return fmt.Errorf("bitnode not specified")
	}

	nconn.active = true
	nconn.node = p.Node
	nconn.version = p.Version
	nconn.bitnode = p.Bitnode
	nconn.remoteAddress = p.Address

	reconnectClients := false

	nconn.factory.connsMux.Lock()
	if econn, ok := nconn.factory.conns[p.Node]; ok {
		nconn.factory.connsMux.Unlock()
		nconn.takeOver(econn)
		if nconn.remoteAddress == "" {
			reconnectClients = true
		}
		econn.refsMux.Lock()
		for _, refChan := range econn.refs {
			log.Printf("remove node msg ref %s", refChan.cmd)
			close(refChan.ch)
		}
		econn.refs = map[string]*NodeRefChan{}
		econn.refsMux.Unlock()
	} else {
		nconn.factory.connsMux.Unlock()
	}
	nconn.factory.connsMux.Lock()
	nconn.factory.conns[p.Node] = nconn
	nconn.factory.connsMux.Unlock()

	if reference != "" {
		nconn.Send("handshake", &NodePayloadHandshake{
			Version: ApiVersion,
			Bitnode: Bitnode,
			Node:    nconn.factory.node.Name(),
			Address: nconn.factory.address,
		}, reference, false)
	}

	if reconnectClients {
		time.Sleep(100 * time.Millisecond) // TODO: Find better way to wait for connection
		if err := nconn.reconnectClients(); err != nil {
			return err
		}
	}

	nconn.factory.queueMux.Lock()
	queuedClients, _ := nconn.factory.queuedClients[p.Node]
	nconn.factory.queueMux.Unlock()

	for _, cl := range queuedClients {
		cl.conn = nconn
		if cl.server {
			continue
		}
		nconn.clientsMux.Lock()
		nconn.clients[cl.cid] = cl
		nconn.clientsMux.Unlock()
		if err := nconn.connectClient(cl); err != nil {
			return err
		}
		if err := cl.connect(); err != nil {
			return err
		}
	}

	nconn.factory.queueMux.Lock()
	nconn.factory.queuedClients[p.Node] = nil
	nconn.factory.queueMux.Unlock()

	return nil
}

type NodePayloadHeartbeat struct {
	Count int64   `json:"beat"`
	Time  float64 `json:"time"`
}

func (p *NodePayloadHeartbeat) Handle(nconn *Conn, reference string) error {
	nconn.remoteBeatCount = p.Count
	nconn.remoteBeatTime = p.Time
	return nil
}

type NodePayloadNewClient struct {
	Client string `json:"client"`
}

func (pc *NodePayloadNewClient) Handle(nconn *Conn, reference string) error {
	_, err := nconn.factory.AcceptClient(nconn.node, pc.Client)
	nconn.Send("", nil, reference, false)
	return err
}

type NodePayloadClient struct {
	Cmd     string        `json:"cmd"`
	Client  string        `json:"client"`
	Payload SystemMessage `json:"payload,omitempty"`
}

func (pc *NodePayloadClient) MarshalJSON() ([]byte, error) {
	var hms struct {
		Cmd     string          `json:"cmd"`
		Client  string          `json:"client"`
		Payload json.RawMessage `json:"payload,omitempty"`
	}
	hms.Cmd = pc.Cmd
	hms.Client = pc.Client
	hms.Payload, _ = json.Marshal(pc.Payload)
	return json.Marshal(hms)
}

func (pc *NodePayloadClient) UnmarshalJSON(data []byte) error {
	var hms struct {
		Cmd       string          `json:"cmd"`
		Client    string          `json:"client"`
		Request   string          `json:"request,omitempty"`
		Reference string          `json:"reference,omitempty"`
		Payload   json.RawMessage `json:"payload,omitempty"`
	}
	if err := json.Unmarshal(data, &hms); err != nil {
		return err
	}
	pc.Cmd = hms.Cmd
	pc.Client = hms.Client

	switch hms.Cmd {
	case "":
		pc.Payload = nil
	case "conn":
		pc.Payload = &SystemMessageConn{}
	case "creds":
		pc.Payload = &SystemMessageCreds{}
	case "init":
		pc.Payload = &SystemMessageInit{}
	case "invoke":
		pc.Payload = &SystemMessageInvoke{}
	case "return":
		pc.Payload = &SystemMessageReturn{}
	case "push":
		pc.Payload = &SystemMessagePush{}
	case "create":
		pc.Payload = &SystemMessageLifecycleCreate{}
	case "load":
		pc.Payload = &SystemMessageLifecycleLoad{}
	case "stop":
		pc.Payload = &SystemMessageLifecycleStop{}
	case "start":
		pc.Payload = &SystemMessageLifecycleStart{}
	case "delete":
		pc.Payload = &SystemMessageLifecycleDelete{}
	case "name":
		pc.Payload = &SystemMessageLifecycleName{}
	case "status":
		pc.Payload = &SystemMessageLifecycleStatus{}
	default:
		return fmt.Errorf("unknown system command: %s", hms.Cmd)
	}

	if pc.Payload == nil {
		return nil
	}

	return json.Unmarshal(hms.Payload, pc.Payload)
}

func (pc *NodePayloadClient) Handle(nconn *Conn, reference string) error {
	nconn.clientsMux.Lock()
	if client, ok := nconn.clients[pc.Client]; !ok {
		nconn.clientsMux.Unlock()
		return fmt.Errorf("client not found in %s: %s", nconn.factory.address, pc.Client)
	} else {
		nconn.clientsMux.Unlock()
		client.handleMux.Lock()
		defer client.handleMux.Unlock()
		if pc.Payload == nil {
			return nil
		}
		return pc.Payload.HandleClient(client, reference)
	}
}

type NodePayloadError struct {
	Error string `json:"error"`
}

func (p *NodePayloadError) Handle(nconn *Conn, reference string) error {
	return fmt.Errorf("%s", p.Error)
}

type NodeRefChan struct {
	cmd string
	ch  chan NodePayload
}

func (ch NodeRefChan) await(msg NodePayload) (NodePayload, error) {
	ret := <-ch.ch
	if err, ok := ret.(error); ok {
		return nil, err
	}
	msg = ret.(NodePayload)
	return ret, nil
}

func (ch NodeRefChan) close() {
	close(ch.ch)
}
