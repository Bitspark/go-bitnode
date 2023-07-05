package wsApi

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
	"sync"
	"time"
)

// Client wraps around a system and provides its interface across a websocket connection.
type Client struct {
	*bitnode.NativeSystem

	// cid is the ID of the client-server connection, chosen by the client.
	cid string

	conn       *Conn
	remoteNode string
	remoteID   bitnode.SystemID

	created time.Time
	server  bool

	incomingIDs map[string]bool
	incomingMux sync.Mutex

	handleMux sync.Mutex

	creds       bitnode.Credentials
	middlewares bitnode.Middlewares

	// defined indicates that hubs and implementation have already been performed.
	defined bool

	// attached indicates that this client has already been attached to the underlying NativeSystem.
	attached bool
}

var _ bitnode.System = &Client{}

// Connect connects the client to the server and ultimately attaches it to the node.
func (cl *Client) Connect(remoteID bitnode.SystemID, creds bitnode.Credentials) error {
	done := make(chan error)
	cl.remoteID = remoteID
	cl.creds = creds
	go func() {
		done <- cl.connect()
		close(done)
	}()
	return <-done
}

func (cl *Client) RemoteName() string {
	return cl.Origin("ws").Name()
}

func (cl *Client) RemoteStatus() int {
	return cl.Origin("ws").Status()
}

func (cl *Client) RemoteID() bitnode.SystemID {
	return cl.remoteID
}

func (cl *Client) Stop(timeout float64) {
	cl.NativeSystem.Stop(cl.creds, timeout)
}

func (cl *Client) Start() {
	cl.NativeSystem.Start(cl.creds)
}

func (cl *Client) Delete() {
	cl.NativeSystem.Delete(cl.creds)
}

func (cl *Client) SetName(name string) {
	cl.NativeSystem.SetName(cl.creds, name)
}

func (cl *Client) SetStatus(status int) {
	cl.NativeSystem.SetStatus(cl.creds, status)
}

func (cl *Client) Hubs() []bitnode.Hub {
	return cl.NativeSystem.Hubs(cl.creds)
}

func (cl *Client) Origins() []bitnode.Origin {
	syss := []bitnode.Origin{}
	for _, sys := range cl.NativeSystem.Origins() {
		syss = append(syss, bitnode.Origin{
			Name:   sys.Name,
			Origin: sys.Origin.Wrap(cl.creds, cl.middlewares),
		})
	}
	return syss
}

func (cl *Client) Origin(name string) bitnode.System {
	orig := cl.NativeSystem.Origin(name)
	if orig == nil {
		return nil
	}
	return cl.NativeSystem.Origin(name).Wrap(cl.creds, cl.middlewares)
}

func (cl *Client) GetHub(name string) bitnode.Hub {
	return cl.NativeSystem.GetHub(cl.creds, cl.middlewares, name)
}

func (cl *Client) Disconnect() error {
	panic("implement me")
}

func (cl *Client) Interface() *bitnode.Interface {
	if cl.NativeSystem == nil {
		return nil
	}
	return cl.NativeSystem.Interface()
}

func (cl *Client) Active() bool {
	return cl.conn.active
}

func (cl *Client) EmitCreate(ctor bitnode.HubItemsInterface, vals ...bitnode.HubItem) error {
	vvals, err := cl.wrapValues(ctor, vals...)
	if err != nil {
		return err
	}
	sendCreate := &SystemMessageLifecycleCreate{
		Params: vvals,
		Types:  ctor,
	}
	ret := cl.send("create", sendCreate, "", true)
	resp := <-ret.ch
	if err, ok := resp.(error); ok {
		return err
	}
	return nil
}

// send sends a command to the remote node it is connected to.
func (cl *Client) send(cmd string, m SystemMessage, reference string, returns bool) *ClientRefChan {
	if cl.conn == nil {
		panic("connection not found")
	}
	chSent := make(chan bool)
	chRef := &ClientRefChan{cmd: cmd, ch: make(chan any)}
	go func(c *Client, nconn *Conn, chSent chan bool, ch *ClientRefChan, reference string, returns bool) {
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
				c.LogError(fmt.Errorf("received error: %s", err.Error))
				ch.ch <- errors.New(err.Error)
				return
			} else if msg == nil {
				c.LogError(fmt.Errorf("received nil response (%s)", ch.cmd))
				ch.ch <- nil
				return
			}
			ch.ch <- msg.(*NodePayloadClient).Payload
		}
	}(cl, cl.conn, chSent, chRef, reference, returns)
	<-chSent
	return chRef
}

/*
attachSystem attaches callbacks to the system and establishes a connection between the websocket connection and the
system.
*/
func (cl *Client) attachSystem() error {
	cl.attached = true
	if cl.NativeSystem == nil {
		return fmt.Errorf("require a system")
	}
	hubs := cl.Hubs()
	errs := make(chan error)

	if cl.server {
		if err := cl.attachSubSystem(cl.NativeSystem, ""); err != nil {
			return err
		}
	}

	cl.NativeSystem.AddCallback(bitnode.LifecycleStop, bitnode.NewNativeEvent(func(vals ...bitnode.HubItem) error {
		// We disconnect the client.

		// TODO: Disconnect.

		return nil
	}))

	cl.NativeSystem.AddCallback(bitnode.LifecycleDelete, bitnode.NewNativeEvent(func(vals ...bitnode.HubItem) error {
		// We disconnect and remove the client.

		// TODO: Disconnect.

		// TODO: Remove client.

		return nil
	}))

	if cl.server {
		for _, hub := range hubs {
			go func(hub bitnode.Hub) { errs <- cl.attachServerHub(hub) }(hub)
		}
	} else {
		for _, hub := range hubs {
			go func(hub bitnode.Hub) { errs <- cl.attachClientHub(hub) }(hub)
		}
	}

	for range hubs {
		if err := <-errs; err != nil {
			return err
		}
	}

	return nil
}

func (cl *Client) attachSubSystem(sys *bitnode.NativeSystem, path string) error {
	sys.AddCallback(bitnode.LifecycleName, bitnode.NewNativeEvent(func(vals ...bitnode.HubItem) error {
		name := vals[0].(string)
		cl.send("name", &SystemMessageLifecycleName{
			Name: name,
			Path: path,
		}, "", false)
		return nil
	}))

	sys.AddCallback(bitnode.LifecycleStatus, bitnode.NewNativeEvent(func(vals ...bitnode.HubItem) error {
		status := vals[0].(int64)
		cl.send("status", &SystemMessageLifecycleStatus{
			Status: int(status),
			Path:   path,
		}, "", false)
		return nil
	}))

	origs := sys.Origins()
	for _, orig := range origs {
		origPath := fmt.Sprintf("%s/%s", path, orig.Name)

		if err := cl.attachSubSystem(orig.Origin, origPath); err != nil {
			return err
		}
	}
	return nil
}

func (cl *Client) attachClientHub(hub bitnode.Hub) error {
	interf := hub.Interface()
	if interf == nil {
		return fmt.Errorf("require interface")
	}
	if interf.Direction != bitnode.HubDirectionIn && interf.Direction != bitnode.HubDirectionBoth {
		return nil
	}
	switch interf.Type {
	case bitnode.HubTypePipe:
		_ = hub.Handle(bitnode.NewNativeFunction(func(creds bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
			if !cl.Active() {
				return nil, fmt.Errorf("client inactive: %s %v", cl.cid, cl.conn)
			}
			wrappedVals, err := cl.wrapValues(interf.Input, vals...)
			if err != nil {
				cl.LogError(err)
				return nil, err
			}
			invoke := cl.send("invoke", &SystemMessageInvoke{
				Hub:   hub.Name(),
				Value: wrappedVals,
			}, "", true)
			if wrappedRets, err := invoke.await(); err != nil {
				cl.LogError(err)
				return nil, err
			} else {
				wrappedVals := wrappedRets.(*SystemMessageReturn)
				rets, err := cl.unwrapValues(interf.Output, wrappedVals.Return...)
				if err != nil {
					cl.LogError(err)
					return nil, err
				}
				return rets, nil
			}
		}))

	case bitnode.HubTypeValue:
		_, _ = hub.Subscribe(bitnode.NewNativeSubscription(func(id string, creds bitnode.Credentials, val bitnode.HubItem) {
			if !cl.Active() {
				return
			}
			cl.incomingMux.Lock()
			if cl.incomingIDs[id] {
				cl.incomingMux.Unlock()
				return
			}
			cl.incomingMux.Unlock()
			wrappedVal, err := cl.wrapValue(*interf.Value, val)
			if err != nil {
				cl.LogError(err)
				return
			}
			cl.send("push", &SystemMessagePush{
				Hub:   hub.Name(),
				ID:    id,
				Value: wrappedVal,
			}, "", false)
		}))

	case bitnode.HubTypeChannel:
		_, _ = hub.Subscribe(bitnode.NewNativeSubscription(func(id string, creds bitnode.Credentials, val bitnode.HubItem) {
			if !cl.Active() {
				return
			}
			wrappedVals, err := cl.wrapValue(*interf.Value, val)
			if err != nil {
				cl.LogError(err)
				return
			}
			cl.send("push", &SystemMessagePush{
				Hub:   hub.Name(),
				ID:    id,
				Value: wrappedVals,
			}, "", false)
		}))
	}
	return nil
}

func (cl *Client) attachServerHub(hub bitnode.Hub) error {
	interf := hub.Interface()
	if interf == nil {
		return fmt.Errorf("require interface")
	}
	if interf.Direction != bitnode.HubDirectionOut && interf.Direction != bitnode.HubDirectionBoth {
		return nil
	}
	switch interf.Type {
	case bitnode.HubTypeChannel:
		_, _ = hub.Subscribe(bitnode.NewNativeSubscription(func(id string, creds bitnode.Credentials, val bitnode.HubItem) {
			if !cl.Active() {
				return
			}
			wrappedVals, err := cl.wrapValue(*interf.Value, val)
			if err != nil {
				cl.LogError(err)
				return
			}
			cl.send("push", &SystemMessagePush{
				Hub:   hub.Name(),
				ID:    id,
				Value: wrappedVals,
			}, "", false)
		}))

	case bitnode.HubTypeValue:
		_, _ = hub.Subscribe(bitnode.NewNativeSubscription(func(id string, creds bitnode.Credentials, val bitnode.HubItem) {
			if !cl.Active() {
				return
			}
			cl.incomingMux.Lock()
			if cl.incomingIDs[id] {
				cl.incomingMux.Unlock()
				return
			}
			cl.incomingMux.Unlock()
			wrappedVal, err := cl.wrapValue(*interf.Value, val)
			if err != nil {
				cl.LogError(err)
				return
			}
			cl.send("push", &SystemMessagePush{
				Hub:   hub.Name(),
				ID:    id,
				Value: wrappedVal,
			}, "", false)
		}))
	}
	return nil
}

// wrapValues transforms native values into values that can be transferred via websocket.
func (cl *Client) wrapValues(interf bitnode.HubItemsInterface, unwrappedVals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
	wrappers := &bitnode.Middlewares{}
	wrappers.PushBack(&systemWrapper{c: cl})
	wrappers.PushBack(&sparkableWrapper{c: cl})
	wrappers.PushBack(&interfaceWrapper{c: cl})
	wrappers.PushBack(&typeWrapper{c: cl})
	wrappers.PushBack(&credsWrapper{c: cl})
	wrappers.PushBack(&idWrapper{c: cl})
	return interf.ApplyMiddlewares(*wrappers, true, unwrappedVals...)
}

// wrapValue transforms a native value into values that can be transferred via websocket.
func (cl *Client) wrapValue(interf bitnode.HubItemInterface, unwrappedVal bitnode.HubItem) (bitnode.HubItem, error) {
	wrappers := &bitnode.Middlewares{}
	wrappers.PushBack(&systemWrapper{c: cl})
	wrappers.PushBack(&sparkableWrapper{c: cl})
	wrappers.PushBack(&interfaceWrapper{c: cl})
	wrappers.PushBack(&typeWrapper{c: cl})
	wrappers.PushBack(&credsWrapper{c: cl})
	wrappers.PushBack(&idWrapper{c: cl})
	return interf.ApplyMiddlewares(*wrappers, unwrappedVal, true)
}

// unwrapValues transforms websocket values into native values.
func (cl *Client) unwrapValues(interf bitnode.HubItemsInterface, wrappedVals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
	unwrappers := &bitnode.Middlewares{}
	unwrappers.PushBack(&systemWrapper{c: cl})
	unwrappers.PushBack(&sparkableWrapper{c: cl})
	unwrappers.PushBack(&interfaceWrapper{c: cl})
	unwrappers.PushBack(&typeWrapper{c: cl})
	unwrappers.PushBack(&credsWrapper{c: cl})
	unwrappers.PushBack(&idWrapper{c: cl})
	return interf.ApplyMiddlewares(*unwrappers, false, wrappedVals...)
}

// unwrapValue transforms a websocket value into native values.
func (cl *Client) unwrapValue(interf bitnode.HubItemInterface, wrappedVal bitnode.HubItem) (bitnode.HubItem, error) {
	unwrappers := &bitnode.Middlewares{}
	unwrappers.PushBack(&systemWrapper{c: cl})
	unwrappers.PushBack(&sparkableWrapper{c: cl})
	unwrappers.PushBack(&interfaceWrapper{c: cl})
	unwrappers.PushBack(&typeWrapper{c: cl})
	unwrappers.PushBack(&credsWrapper{c: cl})
	unwrappers.PushBack(&idWrapper{c: cl})
	return interf.ApplyMiddlewares(*unwrappers, wrappedVal, false)
}

func (cl *Client) Systems() []bitnode.System {
	syss := []bitnode.System{}
	for _, sys := range cl.NativeSystem.Systems() {
		syss = append(syss, sys.Wrap(cl.creds, cl.middlewares))
	}
	return syss
}

func (cl *Client) Native() *bitnode.NativeSystem {
	return cl.NativeSystem
}

func (cl *Client) Credentials() bitnode.Credentials {
	return cl.creds
}

func (cl *Client) SetCredentials(creds bitnode.Credentials) {
	cl.creds = creds
	ret := cl.send("creds", &SystemMessageCreds{
		Credentials: creds,
	}, "", true)
	<-ret.ch
}

func (cl *Client) Middlewares() bitnode.Middlewares {
	return cl.middlewares
}

func (cl *Client) connect() error {
	if cl.server {
		panic("server clients cannot connect to a server")
	}
	ret := cl.send("conn", &SystemMessageConn{
		ID:          cl.remoteID.Hex(),
		Credentials: cl.creds,
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
		if err := tp.Compile(nil, tp.Domain, false); err != nil {
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
