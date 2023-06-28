package factories

import (
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
)

// The Node factory.

type NodeFactory struct {
}

var _ bitnode.Factory = &NodeFactory{}

func NewNodeFactory() *NodeFactory {
	return &NodeFactory{}
}

func (f *NodeFactory) Name() string {
	return "node"
}

func (f *NodeFactory) Implementation(impl bitnode.Implementation) (bitnode.Implementation, error) {
	if impl == nil {
		return &NodeImpl{}, nil
	}
	nImpl, ok := impl.(*NodeImpl)
	if !ok {
		return nil, fmt.Errorf("not a node implementation")
	}
	return nImpl, nil
}

// The Node implementation.

type NodeImpl struct {
	node *bitnode.NativeNode
}

var _ bitnode.Implementation = &NodeImpl{}

func (m *NodeImpl) Implement(node *bitnode.NativeNode, sys bitnode.System) error {
	m.node = node

	sys.AddExtension("node", nodeImpl{m})

	createSystem := sys.GetHub("createSystem")
	createSystem.Handle(bitnode.NewNativeFunction(func(creds bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
		bp := vals[0].(*bitnode.Sparkable)
		sys, err := m.node.NewSystem(creds, *bp, vals[1:])
		if err != nil {
			return nil, err
		}
		//err = bp.Implement(m.node, sys, dom)
		//if err != nil {
		//	return nil
		//}
		return []bitnode.HubItem{sys}, nil
	}))

	addSystem := sys.GetHub("addSystem")
	addSystem.Handle(bitnode.NewNativeFunction(func(creds bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
		s := vals[0].(bitnode.System)
		if s2, _ := m.node.GetSystemByName(creds, s.Name()); s2 == nil {
			if err := m.node.AddSystem(s.(*bitnode.CredSystem).NativeSystem); err != nil {
				return nil, err
			}
		}
		return []bitnode.HubItem{s}, nil
	}))

	getSystems := sys.GetHub("getSystems")
	getSystems.Handle(bitnode.NewNativeFunction(func(creds bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
		syss := []bitnode.HubItem{}
		for _, sys := range m.node.Systems(creds) {
			syss = append(syss, sys)
		}
		return []bitnode.HubItem{syss}, nil
	}))

	getSystemCount := sys.GetHub("getSystemCount")
	getSystemCount.Handle(bitnode.NewNativeFunction(func(creds bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
		count := len(m.node.Systems(creds))
		return []bitnode.HubItem{count}, nil
	}))

	getAddresses := sys.GetHub("getAddresses")
	getAddresses.Handle(bitnode.NewNativeFunction(func(creds bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
		addrs := m.node.Addresses(creds)
		addrMps := []bitnode.HubItem{}
		for _, addr := range addrs {
			addrMps = append(addrMps, map[string]any{
				"network": addr.Network,
				"address": addr.Address,
			})
		}
		return []bitnode.HubItem{addrMps}, nil
	}))

	setAddress := sys.GetHub("setAddress")
	setAddress.Handle(bitnode.NewNativeFunction(func(creds bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
		m.node.SetAddress(creds, vals[0].(string), vals[1].(string))
		return []bitnode.HubItem{}, nil
	}))

	return nil
}

func (m *NodeImpl) Extend(node *bitnode.NativeNode, ext bitnode.Implementation) (bitnode.Implementation, error) {
	return nil, fmt.Errorf("cannot extend node implementations")
}

func (m *NodeImpl) ToInterface() (any, error) {
	return nil, nil
}

func (m *NodeImpl) FromInterface(i any) error {
	return nil
}

func (m *NodeImpl) Validate() error {
	panic("implement me")
}

type nodeImpl struct {
	m *NodeImpl
}

func (h nodeImpl) Implementation() bitnode.Implementation {
	return h.m
}

var _ bitnode.SystemExtension = &nodeImpl{}
