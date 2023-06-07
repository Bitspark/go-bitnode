package libProgram

import (
	"errors"
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
	"github.com/dop251/goja"
	"gopkg.in/yaml.v3"
	"log"
)

// The JavaScript factory.

type JSFactory struct {
	dom *bitnode.Domain
}

var _ bitnode.Factory = &JSFactory{}

func NewJSFactory(dom *bitnode.Domain) *JSFactory {
	return &JSFactory{
		dom: dom,
	}
}

func (f *JSFactory) Name() string {
	return "javascript"
}

func (f *JSFactory) Implementation(impl bitnode.Implementation) (bitnode.Implementation, error) {
	if impl == nil {
		return &JSImpl{
			jsImpl{
				dom: f.dom,
			},
		}, nil
	}
	nImpl, ok := impl.(*JSImpl)
	if !ok {
		return nil, fmt.Errorf("not a javascript implementation")
	} else {
		nImpl.dom = f.dom
	}
	return nImpl, nil
}

// The JavaScript implementation.

type JSImpl struct {
	jsImpl
}

type jsImpl struct {
	// Events implement callbacks for lifecycle events.
	Events []*JSEventImpl `json:"events,omitempty" yaml:"events"`

	// Hubs implement callbacks for hub events.
	Hubs []*JSHubImpl `json:"hubs,omitempty" yaml:"hubs"`

	dom *bitnode.Domain
}

var _ bitnode.Implementation = &JSImpl{}

type JSEventImpl struct {
	Name   string `json:"name"`
	Script string `json:"script"`
}

type JSHubImpl struct {
	Name   string `json:"name"`
	Script string `json:"script"`
}

type jsState struct {
	vm    *goja.Runtime
	impl  jsImpl
	this  *goja.Object
	hubs  map[string]*goja.Object
	creds bitnode.Credentials
}

var _ bitnode.SystemExtension = &jsState{}

func (m *JSImpl) Implement(node *bitnode.NativeNode, sys bitnode.System) error {
	vm := goja.New()
	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))

	thisObj := vm.NewObject()

	st := &jsState{
		impl: m.jsImpl,
		vm:   vm,
		this: thisObj,
		hubs: map[string]*goja.Object{},
	}

	sys.AddExtension("javascript", st)

	//for i := range items {
	//	if err := thisObj.Set(itemTypes[i].Name, vm.ToValue(items[i])); err != nil {
	//		return err
	//	}
	//}

	if err := st.wrapSystem(thisObj, sys, true); err != nil {
		return err
	}

	if err := st.jsConsole(sys); err != nil {
		return err
	}

	if err := st.registerDomain(vm, sys, node, m.dom); err != nil {
		return err
	}

	for _, hubImpl := range m.Hubs {
		hub := sys.GetHub(hubImpl.Name)
		if hub == nil {
			return fmt.Errorf("hub not found: %s", hubImpl.Name)
		}
		hubInterf := hub.Interface()
		if hubInterf.Type == bitnode.HubTypePipe {
			if err := st.attachFunction(sys, hubImpl); err != nil {
				return err
			}
		} else if hubInterf.Type == bitnode.HubTypeChannel || hubInterf.Type == bitnode.HubTypeValue {
			if err := st.attachSubscription(sys, hubImpl); err != nil {
				return err
			}
		}
	}

	for _, lcImpl := range m.Events {
		if err := st.attachEvent(sys, lcImpl); err != nil {
			return err
		}
	}

	// Status and message

	sys.AddCallback(bitnode.LifecycleLoad, bitnode.NewNativeEvent(func(vals ...bitnode.HubItem) error {
		sys.SetMessage("JavaScript engine running")
		sys.SetStatus(bitnode.SystemStatusRunning)

		return nil
	}))

	return nil
}

func (m *JSImpl) Extend(node *bitnode.NativeNode, ext bitnode.Implementation) (bitnode.Implementation, error) {
	add := ext.(*JSImpl)
	for _, cbImpl := range add.Hubs {
		m.Hubs = append(m.Hubs, cbImpl)
	}
	for _, cbImpl := range add.Events {
		m.Events = append(m.Events, cbImpl)
	}
	return m, nil
}

func (m *JSImpl) ToInterface() (any, error) {
	return nil, nil
}

func (m *JSImpl) FromInterface(i any) error {
	dat, err := yaml.Marshal(i)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(dat, m); err != nil {
		return err
	}
	return nil
}

func (m *JSImpl) MarshalYAML() (interface{}, error) {
	return m.jsImpl, nil
}

func (m *JSImpl) UnmarshalYAML(value *yaml.Node) error {
	impl := jsImpl{}
	if err := value.Decode(&impl); err != nil {
		return err
	}
	m.jsImpl = impl
	return nil
}

func (m *JSImpl) Validate() error {
	panic("implement me")
}

func (st *jsState) Implementation() bitnode.Implementation {
	m := &JSImpl{st.impl}
	return m
}

// Private code.

func (st *jsState) jsGlobal() error {
	if err := st.vm.GlobalObject().Set("system", st.vm.ToValue(map[string]any{})); err != nil {
		return err
	}
	return nil
}

func (st *jsState) jsEmit(s bitnode.System) error {
	emitCB := func(hubStr string, val bitnode.HubItem) {
		hub := s.GetHub(hubStr)
		if hub != nil {
			hub.Emit("", val)
		}
	}
	if err := st.vm.GlobalObject().Set("emit", emitCB); err != nil {
		return err
	}
	return nil
}

func (st *jsState) jsConsole(s bitnode.System) error {
	console := map[string]any{
		"log": func(msg ...any) {
			s.Log(bitnode.LogDebug, fmt.Sprintf("%v", msg))
		},
		"info": func(msg ...any) {
			s.Log(bitnode.LogInfo, fmt.Sprintf("%v", msg))
		},
		"warn": func(msg ...any) {
			s.Log(bitnode.LogWarn, fmt.Sprintf("%v", msg))
		},
		"error": func(msg ...any) {
			s.Log(bitnode.LogError, fmt.Sprintf("%v", msg))
		},
	}
	if err := st.vm.GlobalObject().Set("console", console); err != nil {
		return nil
	}
	return nil
}

// wrapSystem creates a JavaScript handle for a system.
func (st *jsState) wrapSystem(vmObj *goja.Object, s bitnode.System, self bool) error {
	// Set the name of the system.
	if err := vmObj.Set("name", s.Name()); err != nil {
		return err
	}

	// Attach getHub function.
	if err := vmObj.Set("getHub", func(hubStr string) *goja.Object {
		hubObj := st.hubs[hubStr]
		if hubObj == nil {
			hub := s.GetHub(hubStr)
			if hub == nil {
				return nil
			}
			hubObj = st.vm.NewObject()
			if err := st.wrapHub(hubObj, hub, self); err != nil {
				return nil
			}
			st.hubs[hubStr] = hubObj
		}
		return hubObj
	}); err != nil {
		return err
	}

	// Attach hu functions.
	//for _, hub := range s.Hubs() {
	//	err := func(hub bitnode.Hub) error {
	//		hubFunction := func(vals ...bitnode.HubItem) any {
	//			retVal, _ := hub.Invoke(vals...)
	//			if len(hub.Interface().Output) == 0 {
	//				return goja.Undefined()
	//			} else if len(hub.Interface().Output) == 1 {
	//				return retVal[0]
	//			} else {
	//				return retVal
	//			}
	//		}
	//		if err := vmObj.Set(hub.Interface().Name, hubFunction); err != nil {
	//			return err
	//		}
	//		return nil
	//	}(hub)
	//	if err != nil {
	//		return err
	//	}
	//}

	return nil
}

// wrapHub creates a JavaScript handle for a system hub.
func (st *jsState) wrapHub(vmObj *goja.Object, s bitnode.Hub, self bool) error {
	inter := s.Interface()

	if inter.Type == bitnode.HubTypeChannel {
		return st.wrapChannelHub(vmObj, s, self)
	}

	if inter.Type == bitnode.HubTypeValue {
		return st.wrapValueHub(vmObj, s, self)
	}

	if inter.Type == bitnode.HubTypePipe {
		return st.wrapPipeHub(vmObj, s, self)
	}

	return fmt.Errorf("unknown hub type: %s", inter.Type)
}

func (st *jsState) wrapChannelHub(vmObj *goja.Object, s bitnode.Hub, self bool) error {
	inter := s.Interface()

	// EMIT
	if self || inter.Direction == bitnode.HubDirectionOut || inter.Direction == bitnode.HubDirectionBoth {
		if err := vmObj.Set("emit", func(vals ...goja.Value) {
			val := toItems(vals...)
			err := s.Emit("", val[0])
			if err != nil {
				return
			}
			return
		}); err != nil {
			return err
		}
	}

	// PUSH
	if self || inter.Direction == bitnode.HubDirectionIn || inter.Direction == bitnode.HubDirectionBoth {
		if err := vmObj.Set("push", func(val ...goja.Value) {
			vals := toItems(val...)
			err := s.Push("", vals[0])
			if err != nil {
				return
			}
			return
		}); err != nil {
			return err
		}
	}

	return nil
}

func (st *jsState) wrapValueHub(vmObj *goja.Object, s bitnode.Hub, self bool) error {
	inter := s.Interface()

	// GET
	if self || inter.Direction == bitnode.HubDirectionOut || inter.Direction == bitnode.HubDirectionBoth {
		if err := vmObj.Set("get", func() goja.Value {
			val, err := s.Get()
			if err != nil {
				return nil
			}
			return st.vm.ToValue(val)
		}); err != nil {
			return err
		}
	}

	// SET
	if self || inter.Direction == bitnode.HubDirectionIn || inter.Direction == bitnode.HubDirectionBoth {
		if err := vmObj.Set("set", func(val ...goja.Value) {
			vals := toItems(val...)
			_ = s.Set("", vals[0])
		}); err != nil {
			return err
		}
	}

	return nil
}

func (st *jsState) wrapPipeHub(vmObj *goja.Object, s bitnode.Hub, self bool) error {
	inter := s.Interface()

	// INVOKE
	if self || inter.Direction == bitnode.HubDirectionIn || inter.Direction == bitnode.HubDirectionBoth {
		if err := vmObj.Set("invoke", func(val ...goja.Value) goja.Value {
			vals := toItems(val...)
			ret, err := s.Invoke(nil, vals...)
			if err != nil {
				return nil
			}
			if len(inter.Output) == 0 {
				return nil
			} else if len(inter.Output) == 1 {
				return st.vm.ToValue(ret[0])
			} else {
				return st.vm.ToValue(ret)
			}
		}); err != nil {
			return err
		}
	}

	return nil
}

func (st *jsState) attachEvent(s bitnode.System, lc *JSEventImpl) error {
	impl, err := st.newJSEvent(lc)
	if err != nil {
		return err
	}
	s.AddCallback(lc.Name, impl)
	return nil
}

func (st *jsState) attachSubscription(s bitnode.System, pi *JSHubImpl) error {
	hub := s.GetHub(pi.Name)
	if hub == nil {
		return fmt.Errorf("hub not found: %s", pi.Name)
	}
	sub, err := st.newJSSubscription(pi)
	if err != nil {
		return err
	}
	hub.Subscribe(sub)
	return nil
}

func (st *jsState) attachFunction(s bitnode.System, pi *JSHubImpl) error {
	hub := s.GetHub(pi.Name)
	if hub == nil {
		return fmt.Errorf("hub not found: %s", pi.Name)
	}
	sub, err := st.newJSFunction(s, hub, pi)
	if err != nil {
		return err
	}
	hub.Handle(sub)
	return nil
}

// registerDomain makes constructors available to the JavaScript vm.
func (st *jsState) registerDomain(vm *goja.Runtime, s bitnode.System, h *bitnode.NativeNode, dom *bitnode.Domain) error {
	libMap := map[string]any{}
	if err := st.registerSubDomain(s, h, dom, libMap); err != nil {
		return err
	}
	bpDom, _ := dom.GetDomain(s.Sparkable().Domain)
	if err := st.registerSubDomain(s, h, bpDom, libMap); err != nil {
		return err
	}
	for k, v := range libMap {
		if err := vm.GlobalObject().Set(k, v); err != nil {
			return err
		}
	}
	return nil
}

func (st *jsState) registerSubDomain(s bitnode.System, h *bitnode.NativeNode, dom *bitnode.Domain, libMap map[string]any) error {
	for _, impl := range dom.Sparkables {
		libMap[impl.Name] = st.registerCtor(s, impl, h)
	}
	for _, sub := range dom.Domains {
		subMap := map[string]any{}
		if err := st.registerSubDomain(s, h, sub, subMap); err != nil {
			return err
		}
		libMap[sub.Name] = subMap
	}
	return nil
}

// registerCtor returns a constructor creating a new system.
// sys is the System to which new instances should be attached.
func (st *jsState) registerCtor(sys bitnode.System, m *bitnode.Sparkable, h *bitnode.NativeNode) func(call goja.ConstructorCall) *goja.Object {
	return func(call goja.ConstructorCall) *goja.Object {
		payload := []bitnode.HubItem{}
		for _, arg := range call.Arguments {
			payload = append(payload, arg)
		}
		inst, err := h.NewSystem(st.creds, *m, payload...)
		if err != nil {
			return nil
		}
		_ = sys.AddSystem(inst.Native())
		_ = st.wrapSystem(call.This, inst, false)
		bitnode.WaitFor(inst, bitnode.SystemStatusRunning)
		return nil
	}
}

func toItems(val ...goja.Value) []bitnode.HubItem {
	vals := []bitnode.HubItem{}
	for _, v := range val {
		vals = append(vals, v.Export())
	}
	return vals
}

// JS Impls

type jsEvent struct {
	impl *JSEventImpl
	cb   goja.Callable
	st   *jsState
}

func (j *jsEvent) Name() string {
	return "javascript"
}

func (j *jsEvent) CB(vals ...bitnode.HubItem) error {
	jsVals := []goja.Value{}
	for _, val := range vals {
		jsVals = append(jsVals, j.st.vm.ToValue(val))
	}
	_, err := j.cb(j.st.this, jsVals...)
	if err != nil {
		log.Printf("error in lifecycle callback: %v", err)
	}
	return nil
}

var _ bitnode.EventImpl = &jsEvent{}

func (st *jsState) newJSEvent(lc *JSEventImpl) (bitnode.EventImpl, error) {
	cbVal, err := st.vm.RunString(lc.Script)
	if err != nil {
		return nil, err
	}
	cb, ok := goja.AssertFunction(cbVal)
	if !ok {
		return nil, errors.New("not a callback")
	}

	return &jsEvent{
		impl: lc,
		cb:   cb,
		st:   st,
	}, nil
}

type jsSubscription struct {
	impl *JSHubImpl
	cb   goja.Callable
	st   *jsState
}

var _ bitnode.SubscribeImpl = &jsSubscription{}

func (j *jsSubscription) Name() string {
	return "javascript"
}

func (j *jsSubscription) CB(id string, creds bitnode.Credentials, val bitnode.HubItem) error {
	vmVals := []goja.Value{j.st.vm.ToValue(val)}
	_, err := j.cb(j.st.this, vmVals...)
	if err != nil {
		log.Printf("error in hub callback: %v", err)
	}
	return nil
}

func (st *jsState) newJSSubscription(lc *JSHubImpl) (bitnode.SubscribeImpl, error) {
	cbVal, err := st.vm.RunString(lc.Script)
	if err != nil {
		return nil, err
	}
	cb, ok := goja.AssertFunction(cbVal)
	if !ok {
		return nil, errors.New("not a callback")
	}

	return &jsSubscription{
		impl: lc,
		cb:   cb,
		st:   st,
	}, nil
}

type jsFunction struct {
	impl *JSHubImpl
	cb   goja.Callable
	s    bitnode.System
	st   *jsState
	hub  bitnode.Hub
}

func (j *jsFunction) Name() string {
	return "javascript"
}

func (j *jsFunction) CB(creds bitnode.Credentials, vals ...bitnode.HubItem) ([]bitnode.HubItem, error) {
	vmVals := []goja.Value{}
	for _, val := range vals {
		vmVals = append(vmVals, j.st.vm.ToValue(val))
	}
	ret, err := j.cb(j.st.this, vmVals...)
	if err != nil {
		log.Printf("error in hub callback: %v", err)
		return nil, err
	}
	if len(j.hub.Interface().Output) == 0 {
		return []bitnode.HubItem{}, nil
	} else if len(j.hub.Interface().Output) == 1 {
		if _, ok := j.hub.Interface().Output[0].Value.Compiled.Extensions["system"]; ok {
			val := ret.Export()
			valMp, ok := val.(map[string]any)
			if !ok {
				return nil, nil
			}
			sys, _ := j.s.Native().GetSystemByName(valMp["name"].(string))
			return []bitnode.HubItem{sys.Wrap(creds, j.s.Middlewares())}, nil
		} else {
			retVal := ret.Export()
			return []bitnode.HubItem{retVal}, nil
		}
	} else {
		retVal := ret.Export()
		if retVals, ok := retVal.([]bitnode.HubItem); ok {
			return retVals, nil
		} else if retVals, ok := retVal.([]any); ok {
			retValsCv := []bitnode.HubItem{}
			for _, retVal := range retVals {
				retValsCv = append(retValsCv, bitnode.HubItem(retVal))
			}
			return retValsCv, nil
		} else {
			return []bitnode.HubItem{retVal}, nil
		}
	}
}

var _ bitnode.FunctionImpl = &jsFunction{}

func (st *jsState) newJSFunction(s bitnode.System, hub bitnode.Hub, lc *JSHubImpl) (bitnode.FunctionImpl, error) {
	cbVal, err := st.vm.RunString(lc.Script)
	if err != nil {
		return nil, err
	}
	cb, ok := goja.AssertFunction(cbVal)
	if !ok {
		return nil, errors.New("not a callback")
	}

	return &jsFunction{
		hub:  hub,
		impl: lc,
		cb:   cb,
		s:    s,
		st:   st,
	}, nil
}
