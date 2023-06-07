package library

// Replace "Blank" with "YourSystem"
// Replace "blank" with "yourSystem"

import (
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
)

// BlankSystem factory.

type BlankFactory struct {
}

var _ bitnode.Factory = &BlankFactory{}

func NewBlankFactory() *BlankFactory {
	return &BlankFactory{}
}

func (f *BlankFactory) Name() string {
	return "blank"
}

func (f *BlankFactory) Implementation(impl bitnode.Implementation) (bitnode.Implementation, error) {
	if impl == nil {
		return &BlankImpl{}, nil
	}
	nImpl, ok := impl.(*BlankImpl)
	if !ok {
		return nil, fmt.Errorf("not a blank implementation")
	}
	return nImpl, nil
}

// BlankSystem implementation.

type BlankImpl struct {
}

var _ bitnode.Implementation = &BlankImpl{}

func (m *BlankImpl) Implement(node *bitnode.NativeNode, sys bitnode.System) error {
	return nil
}

func (m *BlankImpl) Extend(node *bitnode.NativeNode, ext bitnode.Implementation) (bitnode.Implementation, error) {
	return nil, nil
}

func (m *BlankImpl) ToInterface() (any, error) {
	return nil, nil
}

func (m *BlankImpl) FromInterface(i any) error {
	return nil
}

func (m *BlankImpl) Validate() error {
	return nil
}
