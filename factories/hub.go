package factories

import (
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
)

func GetMiddlewares() bitnode.Middlewares {
	mws := bitnode.Middlewares{}
	mws.PushBack(NewSystemMiddleware())
	mws.PushBack(NewSparkableMiddleware())
	mws.PushBack(NewInterfaceMiddleware())
	mws.PushBack(NewTypeMiddleware())
	mws.PushBack(NewIDMiddleware())
	mws.PushBack(NewCredentialsMiddleware())
	return mws
}

// The System middleware.

type SystemMiddleware struct {
}

var _ bitnode.Middleware = &SystemMiddleware{}

func NewSystemMiddleware() *SystemMiddleware {
	return &SystemMiddleware{}
}

func (f *SystemMiddleware) Name() string {
	return "system"
}

func (f *SystemMiddleware) Middleware(ext any, val bitnode.HubItem, out bool) (bitnode.HubItem, error) {
	if val == nil {
		return nil, nil
	}
	//i := ext.(*bitnode.Interface)
	s := val.(bitnode.System)
	//if err := s.Interface().Contains(i); err != nil {
	//	return nil, err
	//}
	return s, nil
}

// The Sparkable middleware.

type SparkableMiddleware struct {
}

var _ bitnode.Middleware = &SparkableMiddleware{}

func NewSparkableMiddleware() *SparkableMiddleware {
	return &SparkableMiddleware{}
}

func (f *SparkableMiddleware) Name() string {
	return "blueprint"
}

func (f *SparkableMiddleware) Middleware(ext any, val bitnode.HubItem, out bool) (bitnode.HubItem, error) {
	bp := val.(*bitnode.Sparkable)
	// TODO: Check
	return bp, nil
}

// The Interface middleware.

type InterfaceMiddleware struct {
}

var _ bitnode.Middleware = &InterfaceMiddleware{}

func NewInterfaceMiddleware() *InterfaceMiddleware {
	return &InterfaceMiddleware{}
}

func (f *InterfaceMiddleware) Name() string {
	return "interface"
}

func (f *InterfaceMiddleware) Middleware(ext any, val bitnode.HubItem, out bool) (bitnode.HubItem, error) {
	ie := val.(*bitnode.Interface)
	// TODO: Check
	return ie, nil
}

// The Type middleware.

type TypeMiddleware struct {
}

var _ bitnode.Middleware = &TypeMiddleware{}

func NewTypeMiddleware() *TypeMiddleware {
	return &TypeMiddleware{}
}

func (f *TypeMiddleware) Name() string {
	return "type"
}

func (f *TypeMiddleware) Middleware(ext any, val bitnode.HubItem, out bool) (bitnode.HubItem, error) {
	tp := val.(*bitnode.Type)
	// TODO: Check
	return tp, nil
}

// The ID middleware.

type IDMiddleware struct {
	dom *bitnode.Domain
}

var _ bitnode.Middleware = &IDMiddleware{}

func NewIDMiddleware() *IDMiddleware {
	return &IDMiddleware{}
}

func (f *IDMiddleware) Name() string {
	return "id"
}

func (f *IDMiddleware) Middleware(ext any, val bitnode.HubItem, out bool) (bitnode.HubItem, error) {
	if val == nil {
		return nil, nil
	}
	extC := ext.(map[string]any)
	tp, _ := extC["type"]
	if tp == nil {
		return val.(bitnode.ID), nil
	} else {
		tps := tp.(string)
		if tps == "object" {
			return val.(bitnode.ObjectID), nil
		} else if tps == "system" {
			return val.(bitnode.SystemID), nil
		}
		return nil, fmt.Errorf("unknown ID type %s", tps)
	}
}

// The Credentials middleware.

type CredentialsMiddleware struct {
}

var _ bitnode.Middleware = &CredentialsMiddleware{}

func NewCredentialsMiddleware() *CredentialsMiddleware {
	return &CredentialsMiddleware{}
}

func (f *CredentialsMiddleware) Name() string {
	return "credentials"
}

func (f *CredentialsMiddleware) Middleware(ext any, val bitnode.HubItem, out bool) (bitnode.HubItem, error) {
	c := val.(bitnode.Credentials)
	return c, nil
}
