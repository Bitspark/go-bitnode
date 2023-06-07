package wsApi

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
	"log"
	"sync"
	"time"
)

// Client wraps around a system and provides its interface across a websocket connection.
type Client struct {
	*bitnode.NativeSystem

	// cid is the ID of the client-server connection, chosen by the client.
	cid string

	conn          *NodeConn
	remoteName    string
	remoteStatus  int
	remoteMessage string
	remoteID      bitnode.SystemID

	created time.Time
	server  bool

	incomingIDs map[string]bool
	incomingMux sync.Mutex

	handleMux sync.Mutex

	creds       bitnode.Credentials
	middlewares bitnode.Middlewares

	attached bool
}

var _ bitnode.System = &Client{}

// Connect connects the client to the server and ultimately attaches it to the node.
func (c *Client) Connect(remoteID bitnode.SystemID, creds bitnode.Credentials) error {
	if c.NativeSystem != nil {
		return nil
	}
	done := make(chan error)
	c.remoteID = remoteID
	c.creds = creds
	go func() {
		done <- c.connect()
		close(done)
	}()
	return <-done
}

func (c *Client) RemoteName() string {
	return c.remoteName
}

func (c *Client) RemoteID() bitnode.SystemID {
	return c.remoteID
}

func (c *Client) SetName(name string) {
	c.NativeSystem.SetName(c.creds, name)
}

func (c *Client) SetStatus(status int) {
	c.NativeSystem.SetStatus(c.creds, status)
}

func (c *Client) SetMessage(message string) {
	c.NativeSystem.SetMessage(c.creds, message)
}

func (c *Client) Hubs() []bitnode.Hub {
	return c.NativeSystem.Hubs(c.creds)
}

func (c *Client) GetHub(name string) bitnode.Hub {
	return c.NativeSystem.GetHub(c.creds, c.middlewares, name)
}

// Connected tells whether the client has been connected already.
func (c *Client) Connected() bool {
	return c.NativeSystem != nil
}

func (c *Client) Log(code int, msg string) {
	if c.server {
		log.Printf("[S-%s-%d] %s", c.cid, code, msg)
	} else {
		log.Printf("[C-%s-%d] %s", c.cid, code, msg)
	}
}

func (c *Client) Disconnect() error {
	panic("implement me")
}

func (c *Client) Interface() *bitnode.Interface {
	if c.NativeSystem == nil {
		return nil
	}
	return c.NativeSystem.Interface()
}

func (c *Client) Active() bool {
	return c.conn.active
}

func (c *Client) EmitCreate(ctor bitnode.HubItemsInterface, vals ...bitnode.HubItem) error {
	vvals, err := c.wrapValues(ctor, vals...)
	if err != nil {
		return err
	}
	sendCreate := &SystemMessageLifecycleCreate{
		Params: vvals,
		Types:  ctor,
	}
	ret := c.send("create", sendCreate, "", true)
	resp := <-ret.ch
	if err, ok := resp.(error); ok {
		return err
	}
	return nil
}

func (c *Client) EmitLoad() error {
	sendLoad := &SystemMessageLifecycleLoad{}
	ret := c.send("load", sendLoad, "", true)
	resp := <-ret.ch
	if err, ok := resp.(error); ok {
		return err
	}
	return nil
}

// send sends a command to the remote node it is connected to.
func (c *Client) send(cmd string, m SystemMessage, reference string, returns bool) *ClientRefChan {
	if c.conn == nil {
		panic("connection not found")
	}
	chSent := make(chan bool)
	chRef := &ClientRefChan{cmd: cmd, ch: make(chan any)}
	go func(c *Client, nconn *NodeConn, chSent chan bool, ch *ClientRefChan, reference string, returns bool) {
		defer close(ch.ch)
		ref := nconn.Send("client", &NodePayloadClient{
			Cmd:     cmd,
			Client:  c.cid,
			Payload: m,
		}, reference, returns)
		chSent <- true
		close(chSent)
		if returns {
			msg := <-ref.ch
			err, isErr := msg.(*NodePayloadError)
			if isErr {
				c.Log(bitnode.LogError, fmt.Sprintf("received error: %s", err.Error))
				ch.ch <- errors.New(err.Error)
				return
			} else if msg == nil {
				c.Log(bitnode.LogError, fmt.Sprintf("received nil response (%s)", ch.cmd))
				ch.ch <- nil
				return
			}
			ch.ch <- msg.(*NodePayloadClient).Payload
		}
	}(c, c.conn, chSent, chRef, reference, returns)
	<-chSent
	return chRef
}

/*
attachSystem attaches callbacks to the system and establishes a connection between the websocket connection and the
system.
*/
func (c *Client) attachSystem() error {
	c.attached = true
	if c.NativeSystem == nil {
		return fmt.Errorf("require a system")
	}
	hubs := c.Hubs()
	errs := make(chan error)

	c.NativeSystem.AddCallback(bitnode.LifecycleMeta, bitnode.NewNativeEvent(func(vals ...bitnode.HubItem) error {
		sendMeta := &SystemMessageLifecycleMeta{}
		if vals[0] != nil {
			name := vals[0].(string)
			if name != c.remoteName {
				sendMeta.Name = &name
			}
		}
		if c.server {
			if vals[1] != nil {
				status := vals[1].(int)
				if status != c.remoteStatus {
					sendMeta.Status = &status
				}
			}
			if vals[2] != nil {
				message := vals[2].(string)
				if message != c.remoteMessage {
					sendMeta.Message = &message
				}
			}
		}
		if sendMeta.Name == nil && sendMeta.Status == nil && sendMeta.Message == nil {
			return nil
		}
		c.send("meta", sendMeta, "", false)

		return nil
	}))

	if c.server {
		for _, hub := range hubs {
			go func(hub bitnode.Hub) { errs <- c.attachServerHub(hub) }(hub)
		}
	} else {
		for _, hub := range hubs {
			go func(hub bitnode.Hub) { errs <- c.attachClientHub(hub) }(hub)
		}
	}

	for range hubs {
		if err := <-errs; err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) attachClientHub(hub bitnode.Hub) error {
	interf := hub.Interface()
	if interf == nil {
		return fmt.Errorf("require interface")
	}
	if interf.Direction != bitnode.HubDirectionIn && interf.Direction != bitnode.HubDirectionBoth {
		return nil
	}
	switch interf.Type {
	case bitnode.HubTypePipe:
		hub.Handle(bitnode.NewNativeFunction(func(creds bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
			if !c.Active() {
				return nil, fmt.Errorf("client inactive: %s %v", c.cid, c.conn)
			}
			wrappedVals, err := c.wrapValues(interf.Input, vals...)
			if err != nil {
				c.Log(bitnode.LogError, err.Error())
				return nil, err
			}
			invoke := c.send("invoke", &SystemMessageInvoke{
				Hub:   hub.Name(),
				Value: wrappedVals,
			}, "", true)
			if wrappedRets, err := invoke.await(); err != nil {
				c.Log(bitnode.LogError, err.Error())
				return nil, err
			} else {
				wrappedVals := wrappedRets.(*SystemMessageReturn)
				rets, err := c.unwrapValues(interf.Output, wrappedVals.Return...)
				if err != nil {
					c.Log(bitnode.LogError, err.Error())
					return nil, err
				}
				return rets, nil
			}
		}))

	case bitnode.HubTypeChannel:
		hub.Subscribe(bitnode.NewNativeSubscription(func(id string, creds bitnode.Credentials, val bitnode.HubItem) {
			if !c.Active() {
				return
			}
			wrappedVals, err := c.wrapValue(*interf.Value, val)
			if err != nil {
				c.Log(bitnode.LogError, err.Error())
				return
			}
			c.send("push", &SystemMessagePush{
				Hub:   hub.Name(),
				ID:    id,
				Value: wrappedVals,
			}, "", false)
		}))
	}
	return nil
}

func (c *Client) attachServerHub(hub bitnode.Hub) error {
	interf := hub.Interface()
	if interf == nil {
		return fmt.Errorf("require interface")
	}
	if interf.Direction != bitnode.HubDirectionOut && interf.Direction != bitnode.HubDirectionBoth {
		return nil
	}
	switch interf.Type {
	case bitnode.HubTypeChannel:
		hub.Subscribe(bitnode.NewNativeSubscription(func(id string, creds bitnode.Credentials, val bitnode.HubItem) {
			if !c.Active() {
				return
			}
			wrappedVals, err := c.wrapValue(*interf.Value, val)
			if err != nil {
				c.Log(bitnode.LogError, err.Error())
				return
			}
			c.send("push", &SystemMessagePush{
				Hub:   hub.Name(),
				ID:    id,
				Value: wrappedVals,
			}, "", false)
		}))

	case bitnode.HubTypeValue:
		hub.Subscribe(bitnode.NewNativeSubscription(func(id string, creds bitnode.Credentials, val bitnode.HubItem) {
			if !c.Active() {
				return
			}
			c.incomingMux.Lock()
			if c.incomingIDs[id] {
				c.incomingMux.Unlock()
				return
			}
			c.incomingMux.Unlock()
			wrappedVal, err := c.wrapValue(*interf.Value, val)
			if err != nil {
				c.Log(bitnode.LogError, err.Error())
				return
			}
			c.send("push", &SystemMessagePush{
				Hub:   hub.Name(),
				ID:    id,
				Value: wrappedVal,
			}, "", false)
		}))
		//val, _ := hub.Get()
		//c.send("push", &SystemMessagePush{
		//	Hub:   hub.Name(),
		//	ID:    "",
		//	Value: val,
		//}, "", false)
	}
	return nil
}

// wrapValues transforms native values into values that can be transferred via websocket.
func (c *Client) wrapValues(interf bitnode.HubItemsInterface, unwrappedVals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
	wrappers := &bitnode.Middlewares{}
	wrappers.PushBack(&systemWrapper{c: c})
	wrappers.PushBack(&modelWrapper{c: c})
	wrappers.PushBack(&sparkableWrapper{c: c})
	wrappers.PushBack(&interfaceWrapper{c: c})
	wrappers.PushBack(&typeWrapper{c: c})
	wrappers.PushBack(&credsWrapper{c: c})
	wrappers.PushBack(&idWrapper{c: c})
	return interf.ApplyMiddlewares(*wrappers, true, unwrappedVals...)
}

// wrapValue transforms a native value into values that can be transferred via websocket.
func (c *Client) wrapValue(interf bitnode.HubItemInterface, unwrappedVal bitnode.HubItem) (bitnode.HubItem, error) {
	wrappers := &bitnode.Middlewares{}
	wrappers.PushBack(&systemWrapper{c: c})
	wrappers.PushBack(&modelWrapper{c: c})
	wrappers.PushBack(&sparkableWrapper{c: c})
	wrappers.PushBack(&interfaceWrapper{c: c})
	wrappers.PushBack(&typeWrapper{c: c})
	wrappers.PushBack(&credsWrapper{c: c})
	wrappers.PushBack(&idWrapper{c: c})
	return interf.ApplyMiddlewares(*wrappers, unwrappedVal, true)
}

// unwrapValues transforms websocket values into native values.
func (c *Client) unwrapValues(interf bitnode.HubItemsInterface, wrappedVals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
	unwrappers := &bitnode.Middlewares{}
	unwrappers.PushBack(&systemWrapper{c: c})
	unwrappers.PushBack(&modelWrapper{c: c})
	unwrappers.PushBack(&sparkableWrapper{c: c})
	unwrappers.PushBack(&interfaceWrapper{c: c})
	unwrappers.PushBack(&typeWrapper{c: c})
	unwrappers.PushBack(&credsWrapper{c: c})
	unwrappers.PushBack(&idWrapper{c: c})
	return interf.ApplyMiddlewares(*unwrappers, false, wrappedVals...)
}

// unwrapValue transforms a websocket value into native values.
func (c *Client) unwrapValue(interf bitnode.HubItemInterface, wrappedVal bitnode.HubItem) (bitnode.HubItem, error) {
	unwrappers := &bitnode.Middlewares{}
	unwrappers.PushBack(&systemWrapper{c: c})
	unwrappers.PushBack(&modelWrapper{c: c})
	unwrappers.PushBack(&sparkableWrapper{c: c})
	unwrappers.PushBack(&interfaceWrapper{c: c})
	unwrappers.PushBack(&typeWrapper{c: c})
	unwrappers.PushBack(&credsWrapper{c: c})
	unwrappers.PushBack(&idWrapper{c: c})
	return interf.ApplyMiddlewares(*unwrappers, wrappedVal, false)
}

func (c *Client) Systems() []bitnode.System {
	syss := []bitnode.System{}
	for _, sys := range c.NativeSystem.Systems() {
		syss = append(syss, sys.Wrap(c.creds, c.middlewares))
	}
	return syss
}

func (c *Client) Native() *bitnode.NativeSystem {
	return c.NativeSystem
}

func (c *Client) Credentials() bitnode.Credentials {
	return c.creds
}

func (c *Client) SetCredentials(creds bitnode.Credentials) {
	c.creds = creds
	ret := c.send("creds", &SystemMessageCreds{
		Credentials: creds,
	}, "", true)
	<-ret.ch
}

func (c *Client) Middlewares() bitnode.Middlewares {
	return c.middlewares
}

func (c *Client) connect() error {
	if c.server {
		panic("server clients cannot connect to a server")
	}
	ret := c.send("conn", &SystemMessageConn{
		ID:          c.remoteID.Hex(),
		Credentials: c.creds,
	}, "", true)
	if _, err := ret.await(); err != nil {
		return err
	}
	return nil
}

type ClientRefChan struct {
	cmd string
	ch  chan any
}

func (ch ClientRefChan) await() (any, error) {
	ret := <-ch.ch
	if err, ok := ret.(error); ok {
		return nil, err
	}
	return ret, nil
}

func (ch ClientRefChan) close() {
	close(ch.ch)
}

// SYSTEM

type systemWrapper struct {
	c *Client
}

type wsSystem struct {
	Node        string              `json:"node"`
	System      string              `json:"system"`
	Credentials bitnode.Credentials `json:"credentials"`
}

var _ bitnode.Middleware = &systemWrapper{}

func (s systemWrapper) Name() string {
	return "system"
}

func (s systemWrapper) Middleware(ext any, val bitnode.HubItem, out bool) (bitnode.HubItem, error) {
	if out {
		sys, _ := val.(bitnode.System)
		if sys == nil {
			return nil, nil
		}
		return wsSystem{
			Node:        sys.Node().Name(),
			System:      sys.ID().Hex(),
			Credentials: sys.Credentials(),
		}, nil
	} else {
		valJSON, _ := json.Marshal(val)
		var wsSys wsSystem
		_ = json.Unmarshal(valJSON, &wsSys)
		sys, err := s.c.conn.AddClient()
		if err != nil {
			return nil, err
		}
		if err := sys.Connect(bitnode.ParseSystemID(wsSys.System), wsSys.Credentials); err != nil {
			return nil, err
		}
		return sys, nil
	}
}

// MODEL

type modelWrapper struct {
	c *Client
}

var _ bitnode.Middleware = &modelWrapper{}

func (s modelWrapper) Name() string {
	return "model"
}

func (s modelWrapper) Middleware(ext any, val bitnode.HubItem, out bool) (bitnode.HubItem, error) {
	if out {
		model, _ := val.(*bitnode.Model)
		if model == nil {
			return nil, nil
		}
		modelMp, _ := model.ToInterface()
		modelMp.(map[string]any)["myPermissions"] = map[string]bitnode.HubItem{
			"owner":  model.Permissions.HavePermissions("owner", s.c.creds),
			"admin":  model.Permissions.HavePermissions("admin", s.c.creds),
			"extend": model.Permissions.HavePermissions("extend", s.c.creds),
			"view":   model.Permissions.HavePermissions("view", s.c.creds),
		}
		return modelMp, nil
	} else {
		model := &bitnode.Model{}
		if err := model.FromInterface(val); err != nil {
			return nil, err
		}
		if err := model.Compile(nil, model.Domain, true); err != nil {
			return nil, err
		}
		return model, nil
	}
}

// SPARKABLE

type sparkableWrapper struct {
	c *Client
}

var _ bitnode.Middleware = &sparkableWrapper{}

func (s sparkableWrapper) Name() string {
	return "blueprint"
}

func (s sparkableWrapper) Middleware(ext any, val bitnode.HubItem, out bool) (bitnode.HubItem, error) {
	if out {
		bp, _ := val.(*bitnode.Sparkable)
		if bp == nil {
			return nil, nil
		}
		bpMp, _ := bp.ToInterface()
		bpMp.(map[string]any)["myPermissions"] = map[string]bitnode.HubItem{
			"owner":  bp.Permissions.HavePermissions("owner", s.c.creds),
			"admin":  bp.Permissions.HavePermissions("admin", s.c.creds),
			"extend": bp.Permissions.HavePermissions("extend", s.c.creds),
			"view":   bp.Permissions.HavePermissions("view", s.c.creds),
		}
		return bpMp, nil
	} else {
		bp := &bitnode.Sparkable{}
		if err := bp.FromInterface(val); err != nil {
			return nil, err
		}
		return bp, nil
	}
}

// INTERFACE

type interfaceWrapper struct {
	c *Client
}

var _ bitnode.Middleware = &interfaceWrapper{}

func (s interfaceWrapper) Name() string {
	return "interface"
}

func (s interfaceWrapper) Middleware(ext any, val bitnode.HubItem, out bool) (bitnode.HubItem, error) {
	if out {
		inf, _ := val.(*bitnode.Interface)
		if inf == nil {
			return nil, nil
		}
		interf, err := inf.ToInterface()
		interf.(map[string]any)["myPermissions"] = map[string]bitnode.HubItem{
			"owner":  inf.Permissions.HavePermissions("owner", s.c.creds),
			"admin":  inf.Permissions.HavePermissions("admin", s.c.creds),
			"extend": inf.Permissions.HavePermissions("extend", s.c.creds),
			"view":   inf.Permissions.HavePermissions("view", s.c.creds),
		}
		if err != nil {
			return nil, err
		}
		return interf, nil
	} else {
		inf := &bitnode.Interface{}
		if err := inf.FromInterface(val); err != nil {
			return nil, err
		}
		if err := inf.Compile(nil, inf.Domain, false); err != nil {
			return nil, err
		}
		return inf, nil
	}
}

// TYPE

type typeWrapper struct {
	c *Client
}

var _ bitnode.Middleware = &typeWrapper{}

func (s typeWrapper) Name() string {
	return "type"
}

func (s typeWrapper) Middleware(ext any, val bitnode.HubItem, out bool) (bitnode.HubItem, error) {
	if out {
		tp, _ := val.(*bitnode.Type)
		if tp == nil {
			return nil, nil
		}
		tpMp, _ := tp.ToInterface()
		tpMp.(map[string]any)["myPermissions"] = map[string]bitnode.HubItem{
			"owner":  tp.Permissions.HavePermissions("owner", s.c.creds),
			"admin":  tp.Permissions.HavePermissions("admin", s.c.creds),
			"extend": tp.Permissions.HavePermissions("extend", s.c.creds),
			"view":   tp.Permissions.HavePermissions("view", s.c.creds),
		}
		return tpMp, nil
	} else {
		tp := &bitnode.Type{}
		if err := tp.FromInterface(val); err != nil {
			return nil, err
		}
		if err := tp.Compile(nil, tp.Domain, true); err != nil {
			return nil, err
		}
		return tp, nil
	}
}

// CREDENTIALS

type credsWrapper struct {
	c *Client
}

var _ bitnode.Middleware = &credsWrapper{}

func (s credsWrapper) Name() string {
	return "credentials"
}

func (s credsWrapper) Middleware(ext any, val bitnode.HubItem, out bool) (bitnode.HubItem, error) {
	if out {
		creds, _ := val.(bitnode.Credentials)
		return creds, nil
	} else {
		creds := bitnode.Credentials{}
		credsJSON, _ := json.Marshal(val)
		_ = json.Unmarshal(credsJSON, &creds)
		return creds, nil
	}
}

// ID WRAPPER

type idWrapper struct {
	c *Client
}

var _ bitnode.Middleware = &idWrapper{}

func (s idWrapper) Name() string {
	return "id"
}

func (s idWrapper) Middleware(ext any, val bitnode.HubItem, out bool) (bitnode.HubItem, error) {
	extC := ext.(map[string]any)
	if out {
		tp, _ := extC["type"]
		if tp == nil {
			if i, ok := val.(bitnode.ID); ok {
				return i.Hex(), nil
			}
		} else {
			tps := tp.(string)
			if tps == "object" {
				if i, ok := val.(bitnode.ObjectID); ok {
					return i.Hex(), nil
				}
			} else if tps == "system" {
				if i, ok := val.(bitnode.SystemID); ok {
					return i.Hex(), nil
				}
			}
		}
		return nil, fmt.Errorf("not an ID: %v", val)
	} else {
		if i, ok := val.(string); ok {
			tp, _ := extC["type"]
			if tp == nil {
				return bitnode.ParseID(i), nil
			} else {
				tps := tp.(string)
				if tps == "object" {
					return bitnode.ParseObjectID(i), nil
				} else if tps == "system" {
					return bitnode.ParseSystemID(i), nil
				}
			}
		}
		return nil, fmt.Errorf("not an ID: %v", val)
	}
}
