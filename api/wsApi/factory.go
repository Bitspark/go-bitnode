package wsApi

import (
	"encoding/json"
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
)

type WSFactory struct {
	conns *NodeConns
}

var _ bitnode.Factory = &WSFactory{}

func NewWSFactory(conns *NodeConns) *WSFactory {
	return &WSFactory{
		conns: conns,
	}
}

func (f *WSFactory) Name() string {
	return "ws"
}

func (f *WSFactory) Implementation(impl bitnode.Implementation) (bitnode.Implementation, error) {
	if impl == nil {
		return &clientImpl{
			conns: f.conns,
		}, nil
	}
	nImpl, ok := impl.(*clientImpl)
	if !ok {
		return nil, fmt.Errorf("not a ws implementation")
	} else {
		nImpl.conns = f.conns
	}
	return nImpl, nil
}

type clientImpl struct {
	CID      string              `json:"cid" yaml:"cid"`
	Node     string              `json:"node" yaml:"node"`
	RemoteID bitnode.SystemID    `json:"remoteId" yaml:"remoteId"`
	Creds    bitnode.Credentials `json:"credentials" yaml:"credentials"`
	Server   bool                `json:"server" yaml:"server"`
	conns    *NodeConns
}

func (c *clientImpl) FromInterface(a any) error {
	ciBts, _ := json.Marshal(a)
	return json.Unmarshal(ciBts, c)
}

func (c *clientImpl) ToInterface() (any, error) {
	return c, nil
}

func (c *clientImpl) Implement(node *bitnode.NativeNode, sys bitnode.System) error {
	_, err := c.conns.ReconnectClient(c.Node, c.CID, c.RemoteID, c.Creds, sys.Native(), c.Server)
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
