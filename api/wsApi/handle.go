package wsApi

import (
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
)

const wsPath = "/ws"

// HANDLERS

type SystemMessage interface {
	HandleClient(client *Client, reference string) error
}

// Conn

// SystemMessageConn is what the client sends to the server to initiate a connection.
type SystemMessageConn struct {
	ID          string              `json:"id"`
	Credentials bitnode.Credentials `json:"credentials"`
}

func (msg *SystemMessageConn) HandleClient(client *Client, reference string) error {
	if client.NativeSystem == nil {
		sys, err := client.conn.conns.node.GetSystemByID(client.creds, bitnode.ParseSystemID(msg.ID))
		if err != nil {
			return err
		}
		client.NativeSystem = sys.Native()
		client.SetExtension("ws", &ClientExt{Client: client})
	}
	client.creds = msg.Credentials
	client.send("init", &SystemMessageInit{
		ID:        client.ID(),
		Name:      client.Name(),
		Status:    client.Status(),
		Interface: client.Interface(),
		Extends:   client.Extends(),
	}, reference, false)
	if err := client.attachSystem(); err != nil {
		return err
	}
	client.Extension("ws").(*ClientExt).Connected = true
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
	ID        bitnode.SystemID   `json:"id"`
	Name      string             `json:"name"`
	Status    int                `json:"status"`
	Message   string             `json:"message"`
	Interface *bitnode.Interface `json:"interface"`
	Extends   []string           `json:"extends"`
}

func (msg *SystemMessageInit) HandleClient(client *Client, reference string) error {
	bp := msg.Interface.Blank()
	if client.NativeSystem == nil {
		sys, err := client.conn.conns.node.PrepareSystem(client.creds, bp)
		if err != nil {
			return err
		}
		client.NativeSystem = sys.Native()
		client.SetExtension("ws", &ClientExt{Client: client})
	}
	if !client.attached {
		if err := client.attachSystem(); err != nil {
			return err
		}
	}

	client.SetName(msg.Name)
	client.remoteName = msg.Name

	client.SetStatus(msg.Status)
	client.remoteStatus = msg.Status

	client.SetExtends(msg.Extends)

	client.Extension("ws").(*ClientExt).Connected = true

	return nil
}

// Invoke

type SystemMessageInvoke struct {
	Hub   string            `json:"hub"`
	Value []bitnode.HubItem `json:"value"`
	User  *bitnode.User     `json:"user"`
}

func (msg *SystemMessageInvoke) HandleClient(client *Client, reference string) error {
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

// Lifecycle Load

type SystemMessageLifecycleKill struct {
}

func (msg *SystemMessageLifecycleKill) HandleClient(client *Client, reference string) error {
	if err := client.EmitEvent(bitnode.LifecycleKill); err != nil {
		return err
	}
	client.send("", nil, reference, false)
	return nil
}

// Lifecycle Name

type SystemMessageLifecycleName struct {
	Name string `json:"name"`
}

func (msg *SystemMessageLifecycleName) HandleClient(client *Client, reference string) error {
	if client.NativeSystem == nil {
		return nil
	}
	client.SetName(msg.Name)
	client.remoteName = msg.Name
	return nil
}

// Lifecycle Status

type SystemMessageLifecycleStatus struct {
	Status int `json:"status"`
}

func (msg *SystemMessageLifecycleStatus) HandleClient(client *Client, reference string) error {
	if client.NativeSystem == nil {
		return nil
	}
	client.SetStatus(msg.Status)
	client.remoteStatus = msg.Status
	return nil
}
