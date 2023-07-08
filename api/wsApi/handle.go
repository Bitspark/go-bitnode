package wsApi

import (
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
	"log"
)

const wsPath = "/ws"

// HANDLERS

type SystemMessage interface {
	HandleClient(client *Client, reference string) error
}

// SystemOrigin provides data about the origins of a hologram.
type SystemOrigin struct {
	ID     bitnode.SystemID        `json:"id"`
	Node   string                  `json:"node"`
	Name   string                  `json:"name"`
	Status int                     `json:"status"`
	Origin map[string]SystemOrigin `json:"origin"`
}

func getOrigin(sys *bitnode.NativeSystem) SystemOrigin {
	origs := SystemOrigin{
		ID:     sys.RemoteID(),
		Node:   sys.RemoteNode(),
		Name:   sys.Name(),
		Status: sys.Status(),
	}
	if origs.ID.IsNull() {
		origs.ID = sys.ID()
	}
	if origs.Node == "" {
		origs.Node = sys.Node().Name()
	}
	os := sys.Origins()
	if len(os) > 0 {
		origs.Origin = map[string]SystemOrigin{}
		for _, o := range os {
			origs.Origin[o.Name] = getOrigin(o.Origin)
		}
	}
	return origs
}

func (cl *Client) attachOrigin(os SystemOrigin, node *bitnode.NativeNode, sys *bitnode.NativeSystem) error {
	sys.SetRemoteNode(os.Node)
	sys.SetRemoteID(os.ID)
	sys.SetName(bitnode.Credentials{}, os.Name)
	sys.SetStatus(bitnode.Credentials{}, os.Status)
	for n, o := range os.Origin {
		orig, err := node.BlankSystem(o.Name)
		if err != nil {
			return err
		}
		sys.AddOrigin(n, orig)
		if err := cl.attachOrigin(o, node, orig); err != nil {
			return err
		}
	}
	return nil
}

// Conn

// SystemMessageConn is what the client sends to the server to initiate a connection.
type SystemMessageConn struct {
	ID          string              `json:"id"`
	Credentials bitnode.Credentials `json:"credentials"`
}

func (msg *SystemMessageConn) HandleClient(client *Client, reference string) error {
	if client.NativeSystem == nil {
		sys, err := client.conn.factory.node.GetSystemByID(client.creds, bitnode.ParseSystemID(msg.ID))
		if err != nil {
			return err
		}
		client.NativeSystem = sys.Native()
		client.SetExtension("ws", &WSExt{Client: client})
	}
	client.creds = msg.Credentials
	client.send("init", &SystemMessageInit{
		Interface: client.Interface(),
		Extends:   client.Extends(),
		Origins:   getOrigin(client.Native()),
	}, reference, false)
	if err := client.attachSystem(); err != nil {
		return err
	}
	client.Extension("ws").(*WSExt).Connected = true
	return nil
}

// Creds

type SystemMessageCreds struct {
	Credentials bitnode.Credentials `json:"credentials"`
}

func (msg *SystemMessageCreds) HandleClient(client *Client, reference string) error {
	client.creds = msg.Credentials
	client.send("", nil, reference, false)
	return nil
}

// Init

// SystemMessageInit is what the client receives from the server after initiating a connection and specifying the
// remote system.
type SystemMessageInit struct {
	Interface *bitnode.Interface `json:"interface"`
	Extends   []string           `json:"extends"`
	Origins   SystemOrigin       `json:"origin"`
}

func (msg *SystemMessageInit) HandleClient(client *Client, reference string) error {
	if client.server {
		return fmt.Errorf("init: %s not a client", client.cid)
	}

	if !client.defined {
		if err := client.conn.factory.node.(*bitnode.NativeNode).ImplementSystem(client.Native(), msg.Interface.Blank()); err != nil {
			return err
		}
		client.defined = true
	}

	orig := client.Origin("ws")
	if orig == nil {
		return fmt.Errorf("init: have no ws origin")
	}
	if err := client.attachOrigin(msg.Origins, client.conn.factory.node.(*bitnode.NativeNode), orig.Native()); err != nil {
		return err
	}

	if !client.attached {
		if err := client.attachSystem(); err != nil {
			return err
		}
	}

	client.SetExtends(msg.Extends)
	client.Extension("ws").(*WSExt).Connected = true

	_ = client.Native().EmitEvent(bitnode.LifecycleStart)

	return nil
}

// Invoke

type SystemMessageInvoke struct {
	Hub   string            `json:"hub"`
	Value []bitnode.HubItem `json:"value"`
	User  *bitnode.User     `json:"user"`
}

func (msg *SystemMessageInvoke) HandleClient(client *Client, reference string) error {
	if !client.server {
		return fmt.Errorf("invoke: %s not a server", client.cid)
	}
	hub := client.GetHub(msg.Hub)
	if hub == nil {
		return fmt.Errorf("could not find hub: %s", msg.Hub)
	}
	vals, err := client.unwrapValues(hub.Interface().Input, msg.Value...)
	if err != nil {
		return err
	}
	wrappedRets, err := hub.Invoke(msg.User, vals...)
	if err != nil {
		return err
	}
	rets, err := client.wrapValues(hub.Interface().Output, wrappedRets...)
	if err != nil {
		return err
	}
	msgRet := &SystemMessageReturn{Return: rets}
	client.send("return", msgRet, reference, false)
	return nil
}

// Return

type SystemMessageReturn struct {
	Return []bitnode.HubItem `json:"return"`
}

func (msg *SystemMessageReturn) HandleClient(client *Client, reference string) error {
	if client.server {
		return fmt.Errorf("return: %s not a client", client.cid)
	}
	return nil
}

// Push

type SystemMessagePush struct {
	Hub   string          `json:"hub"`
	ID    string          `json:"id"`
	Value bitnode.HubItem `json:"value"`
}

func (msg *SystemMessagePush) HandleClient(client *Client, reference string) error {
	if client.NativeSystem == nil {
		return nil
	}
	client.incomingMux.Lock()
	client.incomingIDs[msg.ID] = true
	client.incomingMux.Unlock()
	hub := client.GetHub(msg.Hub)
	if hub == nil {
		return fmt.Errorf("could not find hub: %s", msg.Hub)
	}
	interf := hub.Interface()
	if client.server {
		if interf.Direction == bitnode.HubDirectionOut || interf.Direction == bitnode.HubDirectionNone {
			return fmt.Errorf("wrong direction")
		}
	} else {
		if interf.Direction == bitnode.HubDirectionIn || interf.Direction == bitnode.HubDirectionNone {
			return fmt.Errorf("wrong direction")
		}
	}
	if interf.Type == bitnode.HubTypeValue {
		val, err := client.unwrapValue(*interf.Value, msg.Value)
		if err != nil {
			return err
		}
		err = hub.Set(msg.ID, val)
		if err != nil {
			return err
		}
	} else if interf.Type == bitnode.HubTypeChannel {
		val, err := client.unwrapValue(*interf.Value, msg.Value)
		if err != nil {
			return err
		}
		err = hub.Emit(msg.ID, val)
		if err != nil {
			return err
		}
	}
	return nil
}

// Lifecycle Create

type SystemMessageLifecycleCreate struct {
	Params []bitnode.HubItem         `json:"values"`
	Types  bitnode.HubItemsInterface `json:"types"`
}

func (msg *SystemMessageLifecycleCreate) HandleClient(client *Client, reference string) error {
	if client.NativeSystem == nil {
		return nil
	}
	params, err := client.unwrapValues(msg.Types, msg.Params...)
	if err != nil {
		return err
	}
	if err := client.EmitEvent(bitnode.LifecycleCreate, params...); err != nil {
		return err
	}
	client.send("", nil, reference, false)
	return nil
}

// Lifecycle Load

type SystemMessageLifecycleLoad struct {
}

func (msg *SystemMessageLifecycleLoad) HandleClient(client *Client, reference string) error {
	if err := client.EmitEvent(bitnode.LifecycleLoad); err != nil {
		return err
	}
	client.send("", nil, reference, false)
	return nil
}

// Lifecycle Stop

type SystemMessageLifecycleStop struct {
	Timeout float64 `json:"timeout,omitempty"`
}

func (msg *SystemMessageLifecycleStop) HandleClient(client *Client, reference string) error {
	if err := client.EmitEvent(bitnode.LifecycleStop, msg.Timeout); err != nil {
		return err
	}
	client.send("", nil, reference, false)
	return nil
}

// Lifecycle Start

type SystemMessageLifecycleStart struct {
}

func (msg *SystemMessageLifecycleStart) HandleClient(client *Client, reference string) error {
	if err := client.EmitEvent(bitnode.LifecycleStart); err != nil {
		return err
	}
	client.send("", nil, reference, false)
	return nil
}

// Lifecycle Delete

type SystemMessageLifecycleDelete struct {
}

func (msg *SystemMessageLifecycleDelete) HandleClient(client *Client, reference string) error {
	if err := client.EmitEvent(bitnode.LifecycleDelete); err != nil {
		return err
	}
	client.send("", nil, reference, false)
	return nil
}

// Lifecycle Name

type SystemMessageLifecycleName struct {
	Path string `json:"path,omitempty"`
	Name string `json:"name"`
}

func (msg *SystemMessageLifecycleName) HandleClient(client *Client, reference string) error {
	if client.server {
		return fmt.Errorf("name: %s not a client", client.cid)
	}
	if client.NativeSystem == nil {
		return nil
	}
	path := "ws" + msg.Path
	orig := client.Origin(path)
	if orig == nil {
		return fmt.Errorf("name: path not found: %s", path)
	}
	orig.SetName(msg.Name)
	return nil
}

// Lifecycle Status

type SystemMessageLifecycleStatus struct {
	Path   string `json:"path,omitempty"`
	Status int    `json:"status"`
}

func (msg *SystemMessageLifecycleStatus) HandleClient(client *Client, reference string) error {
	log.Println(*msg)
	if client.server {
		return fmt.Errorf("status: %s not a client", client.cid)
	}
	if client.NativeSystem == nil {
		return nil
	}
	path := "ws" + msg.Path
	orig := client.Origin(path)
	if orig == nil {
		return fmt.Errorf("status: path not found: %s", path)
	}
	orig.SetStatus(msg.Status)
	return nil
}
