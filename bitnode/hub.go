package bitnode

import (
	"fmt"
	"github.com/Bitspark/go-bitnode/util"
	"sync"
	"time"
)

type ValueLog struct {
	Time  time.Time
	Value any
}

type Hub interface {
	// Name returns the name of the hub.
	Name() string

	Native() *NativeHub

	// Interface returns the interface of the hub.
	Interface() *HubInterface

	// Push pushes a set of values from outside into the system. Must follow HubInterface.Input of Interface.
	Push(id string, val HubItem) error

	// Emit pushes a set of values inside the system. Must follow HubInterface.Output of Interface.
	Emit(id string, val HubItem) error

	// Invoke triggers an invocation of the hub.
	Invoke(user *User, value ...HubItem) ([]HubItem, error)

	// Subscribe adds a callback to this hub which is called when values are pushed into the hub.
	Subscribe(impl SubscribeImpl) (string, error)

	// Unsubscribe removes a callback previously attached via Subscribe.
	Unsubscribe(subID string) error

	// Handle sets the invocation routine.
	Handle(impl FunctionImpl) error

	// Set sets a set of values. Requires this hub to be a value hub.
	Set(id string, val HubItem) error

	// Get returns the current value, provided it is a value hub, returns an error otherwise.
	Get() (HubItem, error)
}

type SubscribeImpl interface {
	Name() string
	CB(id string, creds Credentials, val HubItem) error
}

type nativeSubscription struct {
	cb func(id string, creds Credentials, val HubItem)
}

func (n *nativeSubscription) Name() string {
	//TODO implement me
	panic("native")
}

func (n *nativeSubscription) CB(id string, creds Credentials, val HubItem) error {
	n.cb(id, creds, val)
	return nil
}

var _ SubscribeImpl = &nativeSubscription{}

func NewNativeSubscription(cb func(id string, creds Credentials, val HubItem)) SubscribeImpl {
	return &nativeSubscription{
		cb: cb,
	}
}

type FunctionImpl interface {
	Name() string
	CB(creds Credentials, vals ...HubItem) ([]HubItem, error)
}

type nativeFunction struct {
	cb func(creds Credentials, vals ...HubItem) ([]HubItem, error)
}

var _ FunctionImpl = &nativeFunction{}

func (n *nativeFunction) Name() string {
	return "native"
}

func (n *nativeFunction) CB(creds Credentials, vals ...HubItem) ([]HubItem, error) {
	return n.cb(creds, vals...)
}

func NewNativeFunction(cb func(user Credentials, vals ...HubItem) ([]HubItem, error)) FunctionImpl {
	return &nativeFunction{
		cb: cb,
	}
}

// A NativeHub is a node which accepts and distributes values.
// It also keeps track of previously sent values and offers them to new connections.
// values sent through a hub have a timestamp attached to them, allowing to keep the orders of values.
type NativeHub struct {
	// parent system this factory is part of.
	parent *NativeSystem

	// hubInterface for the values of this hub.
	hubInterface *HubInterface

	// subscriptions contains functions which are called when a new value is pushed.
	subscriptions map[string]SubscribeImpl

	// function holds the function which is called when the hub is invoked.
	function FunctionImpl

	// value is the value of this hub. If this hub is not a value hub, it remains nil.
	value HubItem

	// handled contains IDs that have already been processed.
	handled map[string]bool

	handledMux sync.Mutex
	mux        sync.Mutex
}

func NewHub(parent *NativeSystem, t *HubInterface) *NativeHub {
	p := &NativeHub{
		parent:        parent,
		hubInterface:  t,
		subscriptions: map[string]SubscribeImpl{},
		handled:       map[string]bool{},
	}
	return p
}

func (p *NativeHub) Name() string {
	return p.hubInterface.Name
}

func (p *NativeHub) Interface() *HubInterface {
	return p.hubInterface
}

func (p *NativeHub) Native() *NativeHub {
	return p
}

func (p *NativeHub) Push(creds Credentials, mws Middlewares, id string, val HubItem) error {
	p.handledMux.Lock()
	if p.handled[id] {
		p.handledMux.Unlock()
		return nil
	}
	p.handledMux.Unlock()
	if p.Interface().Type != HubTypeChannel {
		return fmt.Errorf("require a channel hub")
	}
	if val == NilItem {
		return nil
	}
	if p.hubInterface == nil {
		return fmt.Errorf("require interface")
	}
	if vval, err := p.hubInterface.Value.ApplyMiddlewares(mws, val, false); err != nil {
		return err
	} else {
		return p.broadcast(id, creds, vval)
	}
}

func (p *NativeHub) Emit(creds Credentials, mws Middlewares, id string, val HubItem) error {
	if id == "" {
		id = util.RandomString(util.CharsAlphaNum, 8)
	}
	if p.Interface().Type != HubTypeChannel {
		return fmt.Errorf("require a channel hub")
	}
	return p.emit(id, creds, mws, val)
}

func (p *NativeHub) emit(id string, creds Credentials, mws Middlewares, val HubItem) error {
	p.handledMux.Lock()
	if p.handled[id] {
		p.handledMux.Unlock()
		return nil
	}
	p.handled[id] = true
	p.handledMux.Unlock()
	if id == "" {
		id = util.RandomString(util.CharsAlphaNum, 8)
	}
	if p.hubInterface == nil {
		return fmt.Errorf("require interface")
	}
	var interf *HubItemInterface
	interf = p.hubInterface.Value
	if vval, err := interf.ApplyMiddlewares(mws, val, false); err != nil {
		return err
	} else {
		if p.Interface().Type == HubTypeValue {
			p.value = vval
		}
		go p.broadcast(id, creds, vval)
		return nil
	}
}

func (p *NativeHub) broadcast(id string, creds Credentials, val HubItem) error {
	p.mux.Lock()
	for _, cb := range p.subscriptions {
		_ = cb.CB(id, creds, val)
	}
	p.mux.Unlock()
	return nil
}

func (p *NativeHub) Invoke(creds Credentials, mws Middlewares, vals ...HubItem) ([]HubItem, error) {
	if p.Interface().Type != HubTypePipe {
		return nil, fmt.Errorf("require a pipe hub")
	}
	if p.hubInterface == nil {
		return nil, fmt.Errorf("require interface")
	}
	if vvals, err := p.hubInterface.Input.ApplyMiddlewares(mws, false, vals...); err != nil {
		return nil, err
	} else {
		if p.function == nil {
			return nil, fmt.Errorf("[system %s %s] have no invoke callback for %s", p.parent.id.Hex(), p.parent.name, p.Name())
		}
		rets, err := p.function.CB(creds, vvals...)
		if err != nil {
			return nil, err
		}
		if vrets, err := p.hubInterface.Output.ApplyMiddlewares(mws, true, rets...); err != nil {
			return nil, err
		} else {
			return vrets, nil
		}
	}
}

func (p *NativeHub) Handle(proc FunctionImpl) error {
	if p.function != nil {
		panic("already have handle function")
		return nil
	}
	if p.Interface().Type != HubTypePipe {
		return fmt.Errorf("require a pipe hub")
	}
	p.function = proc
	return nil
}

func (p *NativeHub) Subscribe(creds Credentials, mws Middlewares, impl SubscribeImpl) (string, error) {
	// TODO: mws!
	if p.Interface().Type != HubTypeValue && p.Interface().Type != HubTypeChannel {
		return "", fmt.Errorf("require a value or channel hub")
	}
	hubType := p.Interface().Type
	subID := util.RandomString(util.CharsAlphaNum, 8)
	p.mux.Lock()
	p.subscriptions[subID] = impl
	p.mux.Unlock()
	if hubType == HubTypeValue {
		_ = impl.CB(util.RandomString(util.CharsAlphaNum, 8), creds, p.value)
	}
	return subID, nil
}

func (p *NativeHub) Unsubscribe(subID string) error {
	if p.Interface().Type != HubTypeValue && p.Interface().Type != HubTypeChannel {
		return fmt.Errorf("require a value or channel hub")
	}
	p.mux.Lock()
	if _, ok := p.subscriptions[subID]; !ok {
		p.mux.Unlock()
		return fmt.Errorf("subscription %s not found", subID)
	}
	delete(p.subscriptions, subID)
	p.mux.Unlock()
	return nil
}

func (p *NativeHub) Set(creds Credentials, mws Middlewares, id string, val HubItem) error {
	if id == "" {
		id = util.RandomString(util.CharsAlphaNum, 8)
	}
	if p.Interface().Type != HubTypeValue {
		return fmt.Errorf("require a value hub")
	}
	p.value = val
	return p.emit(id, creds, mws, p.value)
}

func (p *NativeHub) Get(creds Credentials, mws Middlewares) (HubItem, error) {
	// TODO: creds and mws!
	if p.Interface().Type != HubTypeValue {
		return nil, fmt.Errorf("require a value hub")
	}
	return p.value, nil
}

type CredHub struct {
	*NativeHub
	creds Credentials
	mws   Middlewares
}

var _ Hub = &CredHub{}

func (c CredHub) Push(id string, val HubItem) error {
	return c.NativeHub.Push(c.creds, c.mws, id, val)
}

func (c CredHub) Emit(id string, val HubItem) error {
	return c.NativeHub.Emit(c.creds, c.mws, id, val)
}

func (c CredHub) Invoke(user *User, value ...HubItem) ([]HubItem, error) {
	cred := c.creds
	if user != nil {
		cred.User = *user
	}
	return c.NativeHub.Invoke(cred, c.mws, value...)
}

func (c CredHub) Subscribe(impl SubscribeImpl) (string, error) {
	return c.NativeHub.Subscribe(c.creds, c.mws, impl)
}

func (c CredHub) Unsubscribe(subID string) error {
	return c.NativeHub.Unsubscribe(subID)
}

func (c CredHub) Handle(impl FunctionImpl) error {
	return c.NativeHub.Handle(impl)
}

func (c CredHub) Set(id string, val HubItem) error {
	return c.NativeHub.Set(c.creds, c.mws, id, val)
}

func (c CredHub) Get() (HubItem, error) {
	return c.NativeHub.Get(c.creds, c.mws)
}

type HubType string
type HubDirection string

const (
	HubTypeChannel = HubType("channel")
	HubTypeValue   = HubType("value")
	HubTypePipe    = HubType("pipe")
)

const (
	HubDirectionNone = HubDirection("none")
	HubDirectionIn   = HubDirection("in")
	HubDirectionOut  = HubDirection("out")
	HubDirectionBoth = HubDirection("both")
)
