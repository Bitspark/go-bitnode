package bitnode

import (
	"encoding/json"
	"fmt"
	"github.com/Bitspark/go-bitnode/store"
	"github.com/Bitspark/go-bitnode/util"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	LifecycleCreate = "create"
	LifecycleLoad   = "load"
	LifecycleStore  = "store"
	LifecycleName   = "name"
	LifecycleStatus = "status"
	LifecycleLog    = "log"
	LifecycleStop   = "stop"
	LifecycleKill   = "kill"
	LifecycleStart  = "start"
)

const (
	SystemStatusUndefined    = 0
	SystemStatusStarting     = 1
	SystemStatusRunning      = 2
	SystemStatusStopped      = 3
	SystemStatusKilled       = 4
	SystemStatusDisconnected = 5
	SystemStatusStopping     = 6
	SystemStatusConnecting   = 7
)

// LifecycleEvent contains events event callbacks.
type LifecycleEvent struct {
	Event     string
	Callbacks []EventImpl
}

// LogMessage is a log message of a system.
type LogMessage struct {
	Level   int       `json:"level" yaml:"level"`
	Time    time.Time `json:"time" yaml:"time"`
	Message string    `json:"message" yaml:"message"`
}

type System interface {
	// Node this system is running on.
	Node() Node

	// ID which uniquely identifies this system.
	ID() SystemID

	// Name of this system.
	Name() string

	// Interface returns the interface this system implements.
	Interface() *Interface

	// Status of this system.
	Status() int

	// Kill kills the system instantly.
	Kill()

	// SetName changes the name of the system.
	SetName(name string)

	// SetStatus changes the status of the system.
	SetStatus(status int)

	// GetHub returns a hub of this system.
	GetHub(hubName string) Hub

	// Hubs returns all hubs of this system.
	Hubs() []Hub

	// LogDebug logs a debug message.
	LogDebug(msg string)

	// LogInfo logs an info message.
	LogInfo(msg string)

	// LogWarning logs a warning message.
	LogWarning(msg string)

	// LogError logs an error message.
	LogError(err error)

	// LogFatal logs a fatal error message.
	LogFatal(err error)

	// Connected reveals if this system is connected.
	Connected() bool

	// AddCallback adds a callback to an event.
	AddCallback(event string, impl EventImpl)

	// AddExtension attaches a system extension to the system.
	AddExtension(name string, impl SystemExtension)

	// SetExtension sets a system extension of the system.
	SetExtension(name string, impl SystemExtension)

	// Extensions returns extensions by name.
	Extensions(name string) []SystemExtension

	// Extension returns an extension by name.
	Extension(name string) SystemExtension

	// AddSystem attaches a system.
	AddSystem(sys *NativeSystem) error

	// Systems returns child systems.
	Systems() []System

	// Sparkable returns the sparkable.
	Sparkable() *Sparkable

	Native() *NativeSystem

	Credentials() Credentials

	SetCredentials(creds Credentials)

	Middlewares() Middlewares

	Extends() []string
}

type EventImpl interface {
	Name() string
	CB(vals ...HubItem) error
}

type nativeEvent struct {
	cb func(vals ...HubItem) error
}

var _ EventImpl = &nativeEvent{}

func (n *nativeEvent) Name() string {
	return "native"
}

func (n *nativeEvent) CB(vals ...HubItem) error {
	return n.cb(vals...)
}

func NewNativeEvent(cb func(vals ...HubItem) error) EventImpl {
	return &nativeEvent{
		cb: cb,
	}
}

type NativeSystem struct {
	node *NativeNode

	// id of this system.
	id SystemID

	// name of this system.
	name string

	// sparkable this system has been created from.
	sparkable Sparkable

	// The parent system of this system.
	parent System

	// The systems which are children of this system and should be destroyed together with it.
	systems map[SystemID]*NativeSystem

	// created is the time when the system has been created.
	created time.Time

	// The hubs of this system.
	hubs []*NativeHub

	// events contains callbacks for lifecycle events.
	events map[string]*LifecycleEvent

	// logs of this system.
	logs util.Sorted[int64, LogMessage]

	// extends these interfaces.
	extends []string

	impls map[string][]SystemExtension

	status int

	message string

	eventsMux sync.Mutex

	implMux sync.Mutex
}

func (s *NativeSystem) Node() Node {
	return s.node
}

func (s *NativeSystem) ID() SystemID {
	return s.id
}

func (s *NativeSystem) Name() string {
	return s.name
}

func (s *NativeSystem) Status() int {
	return s.status
}

func (s *NativeSystem) Kill(creds Credentials) {
	_ = s.EmitEvent(LifecycleKill)
}

func (s *NativeSystem) SetName(creds Credentials, name string) {
	_ = s.EmitEvent(LifecycleName, name)
}

func (s *NativeSystem) SetStatus(creds Credentials, status int) {
	_ = s.EmitEvent(LifecycleStatus, int64(status))
}

func (s *NativeSystem) Constructor() HubItemsInterface {
	return s.sparkable.Constructor
}

func (s *NativeSystem) Sparkable() *Sparkable {
	bp := &Sparkable{}
	bp.compiled = true
	bp.Implementation = map[string][]any{}

	bp.Interface = NewInterface()
	bp.Domain = s.sparkable.Domain
	bp.Interface.Domain = s.sparkable.Domain
	for _, hub := range s.hubs {
		hubInterf := hub.Interface()
		_ = bp.Interface.Hubs.AddHub(hubInterf)
		_ = bp.Interface.CompiledHubs.AddHub(hubInterf)
	}

	s.implMux.Lock()
	for f, ms := range s.impls {
		for _, m := range ms {
			impl := m.Implementation()
			if impl == nil {
				continue
			}
			impls, _ := bp.Implementation[f]
			impls = append(impls, impl)
			bp.Implementation[f] = impls
		}
	}
	s.implMux.Unlock()

	return bp
}

func (s *NativeSystem) Interface() *Interface {
	if s == nil {
		return nil
	}
	interf := NewInterface()
	interf.Domain = s.sparkable.Domain
	for _, hub := range s.hubs {
		hubInterf := hub.Interface()
		_ = interf.Hubs.AddHub(hubInterf)
		_ = interf.CompiledHubs.AddHub(hubInterf)
	}
	return interf
}

func (s *NativeSystem) GetNativeHub(hubName string) *NativeHub {
	return s.getHub(hubName)
}

func (s *NativeSystem) GetHub(creds Credentials, mws Middlewares, hubName string) Hub {
	hub := s.getHub(hubName)
	if hub == nil {
		return nil
	}
	return CredHub{NativeHub: hub, creds: creds, mws: mws}
}

func (s *NativeSystem) getHub(hubName string) *NativeHub {
	if s == nil {
		return nil
	}
	for _, hub := range s.hubs {
		if hub.hubInterface.Name == hubName {
			return hub
		}
	}
	return nil
}

func (s *NativeSystem) Hubs(creds Credentials) []Hub {
	hubs := []Hub{}
	for _, h := range s.hubs {
		hubs = append(hubs, CredHub{NativeHub: h, creds: creds})
	}
	return hubs
}

func (s *NativeSystem) Systems() []*NativeSystem {
	syss := []*NativeSystem{}
	for _, sys := range s.systems {
		syss = append(syss, sys)
	}
	return syss
}

// AddCallback registers a callback to the factory which is called when a new system is produced.
func (s *NativeSystem) AddCallback(event string, impl EventImpl) {
	s.eventsMux.Lock()
	defer s.eventsMux.Unlock()
	if evts, ok := s.events[event]; ok {
		evts.Callbacks = append(evts.Callbacks, impl)
		s.events[event] = evts
	} else {
		s.events[event] = &LifecycleEvent{
			Event:     event,
			Callbacks: []EventImpl{impl},
		}
	}
}

// EmitEvent emits a new events event.
func (s *NativeSystem) EmitEvent(name string, args ...HubItem) error {
	s.eventsMux.Lock()
	events, _ := s.events[name]
	s.eventsMux.Unlock()
	if events != nil {
		for _, cb := range events.Callbacks {
			if err := cb.CB(args...); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *NativeSystem) GetSystemByName(name string) (*NativeSystem, error) {
	for _, sys := range s.systems {
		if sys.Name() == name {
			return sys, nil
		}
	}
	return nil, fmt.Errorf("child system not found: %s", name)
}

func (s *NativeSystem) AddSystem(sys *NativeSystem) error {
	if _, ok := s.systems[sys.ID()]; ok {
		return fmt.Errorf("already have child with name %s", sys.Name())
	}
	s.systems[sys.ID()] = sys
	return nil
}

func (s *NativeSystem) Connected() bool {
	return true
}

func (s *NativeSystem) Connect() error {
	return nil
}

func (s *NativeSystem) Store(st store.Store) error {
	systemStoreDS, _ := st.Ensure("system", store.DSKeyValue)
	systemStore := systemStoreDS.KeyValue()

	_ = systemStore.Set("id", s.id.Hex())
	_ = systemStore.Set("name", s.name)
	_ = systemStore.Set("status", fmt.Sprintf("%d", s.status))
	_ = systemStore.Set("message", s.message)
	_ = systemStore.Set("extends", strings.Join(s.extends, ","))

	bp, _ := s.Sparkable().ToInterface()
	bpJSON, _ := json.Marshal(bp)
	_ = systemStore.Set("sparkable", string(bpJSON))

	hubStoreDS, _ := st.Ensure("hubs", store.DSKeyValue)
	hubStore := hubStoreDS.KeyValue()

	creds := Credentials{}

	for _, hub := range s.hubs {
		hubInterf := hub.Interface()
		switch hubInterf.Type {
		case HubTypeValue:
			val, err := hub.Get(creds, s.node.middlewares)
			if err != nil {
				log.Printf("error getting value %s: %v", hub.Name(), err)
				continue
			}
			if hub.Interface().Value == nil {
				panic(hub.Name())
			}
			vval, _ := hub.Interface().Value.ApplyMiddlewares(Middlewares{systemWrapper{h: s.node}, idWrapper{h: s.node}}, val, true)
			valBts, _ := json.Marshal(vval)
			_ = hubStore.Set(hub.Name(), string(valBts))
		}
	}

	childStoreDS, _ := st.Ensure("children", store.DSKeyValue)
	childStore := childStoreDS.KeyValue()

	for _, sys := range s.systems {
		_ = childStore.Set(sys.ID().Hex(), sys.Name())
	}

	return nil
}

func (s *NativeSystem) LoadInit(node *NativeNode, st store.Store) error {
	s.node = node
	s.systems = map[SystemID]*NativeSystem{}
	s.events = map[string]*LifecycleEvent{}

	if err := node.initSystem(s); err != nil {
		return err
	}

	systemStoreDS, _ := st.Ensure("system", store.DSKeyValue)
	systemStore := systemStoreDS.KeyValue()

	idStr, _ := systemStore.Get("id")
	s.id = ParseSystemID(idStr)

	s.name, _ = systemStore.Get("name")
	status, _ := systemStore.Get("status")
	s.status, _ = strconv.Atoi(status)
	s.message, _ = systemStore.Get("message")

	extends, _ := systemStore.Get("extends")
	s.extends = strings.Split(extends, ",")

	return nil
}

func (s *NativeSystem) Load(node *NativeNode, dom *Domain, st store.Store) error {
	systemStoreDS, _ := st.Ensure("system", store.DSKeyValue)
	systemStore := systemStoreDS.KeyValue()

	bpJSON, _ := systemStore.Get("sparkable")
	var bpIF any
	if err := json.Unmarshal([]byte(bpJSON), &bpIF); err != nil {
		return err
	}

	if err := s.sparkable.FromInterface(bpIF); err != nil {
		return err
	}

	if err := s.sparkable.Compile(dom, s.sparkable.Domain, true); err != nil {
		return err
	}

	// Add hubs if interface is present.
	mi := s.sparkable.Interface
	if mi != nil && mi.CompiledHubs != nil {
		for _, p := range *s.sparkable.Interface.CompiledHubs {
			hub := NewHub(s, p)
			s.hubs = append(s.hubs, hub)
		}
	}

	if err := s.sparkable.Implement(node, s.Wrap(Credentials{}, node.middlewares)); err != nil {
		return err
	}

	hubStoreDS, _ := st.Ensure("hubs", store.DSKeyValue)
	hubStore := hubStoreDS.KeyValue()

	creds := Credentials{}

	for hubName := range hubStore.EnumerateKeys() {
		hub := s.getHub(hubName)
		hubInterf := hub.Interface()
		switch hubInterf.Type {
		case HubTypeValue:
			hubValBts, _ := hubStore.Get(hubName)
			var val HubItem
			_ = json.Unmarshal([]byte(hubValBts), &val)
			vval, _ := hub.Interface().Value.ApplyMiddlewares(Middlewares{systemWrapper{h: s.node}, idWrapper{h: s.node}}, val, false)
			_ = hub.Set(creds, node.middlewares, "", vval)
		}
	}

	childStoreDS, _ := st.Ensure("children", store.DSKeyValue)
	childStore := childStoreDS.KeyValue()

	for chIDStr := range childStore.EnumerateKeys() {
		chID := ParseSystemID(chIDStr)
		s.systems[chID] = nil
	}

	return nil
}

func (s *NativeSystem) AddExtension(name string, impl SystemExtension) {
	s.implMux.Lock()
	exts, _ := s.impls[name]
	exts = append(exts, impl)
	s.impls[name] = exts
	s.implMux.Unlock()
}

func (s *NativeSystem) SetExtension(name string, impl SystemExtension) {
	s.implMux.Lock()
	s.impls[name] = []SystemExtension{impl}
	s.implMux.Unlock()
}

func (s *NativeSystem) Extensions(name string) []SystemExtension {
	s.implMux.Lock()
	defer s.implMux.Unlock()
	return s.impls[name]
}

func (s *NativeSystem) Extension(name string) SystemExtension {
	s.implMux.Lock()
	defer s.implMux.Unlock()
	exts := s.impls[name]
	if len(exts) != 1 {
		return nil
	}
	return exts[0]
}

func (s *NativeSystem) Wrap(creds Credentials, mws Middlewares) *CredSystem {
	return &CredSystem{
		NativeSystem: s,
		creds:        creds,
		middlewares:  mws,
	}
}

func (s *NativeSystem) SetExtends(extends []string) {
	s.extends = extends
}

func (s *NativeSystem) Extends() []string {
	return s.extends
}

type CredSystem struct {
	*NativeSystem
	creds       Credentials
	middlewares Middlewares
}

var _ System = &CredSystem{}

func (s *CredSystem) Kill() {
	s.NativeSystem.Kill(s.creds)
}

func (s *CredSystem) SetName(name string) {
	s.NativeSystem.SetName(s.creds, name)
}

func (s *CredSystem) SetStatus(status int) {
	s.NativeSystem.SetStatus(s.creds, status)
}

func (s *CredSystem) GetHub(hubName string) Hub {
	return s.NativeSystem.GetHub(s.creds, s.middlewares, hubName)
}

func (s *CredSystem) Hubs() []Hub {
	return s.NativeSystem.Hubs(s.creds)
}

func (s *CredSystem) Systems() []System {
	syss := []System{}
	for _, sys := range s.systems {
		syss = append(syss, sys.Wrap(s.creds, s.middlewares))
	}
	return syss
}

func (s *CredSystem) Native() *NativeSystem {
	return s.NativeSystem
}

func (s *CredSystem) Credentials() Credentials {
	return s.creds
}

func (s *CredSystem) SetCredentials(creds Credentials) {
	s.creds = creds
}

func (s *CredSystem) Middlewares() Middlewares {
	return s.middlewares
}

const (
	// LogDebug indicates the message provides details about the inner workings of the system.
	LogDebug = iota

	// LogInfo indicates the message provides information about the progress of the system.
	LogInfo

	// LogWarning indicates the message provides details about a non-critical problem that occurred in the system.
	LogWarning

	// LogError indicates the message informs about a local non-fatal error in the system.
	LogError

	// LogFatal indicates the message informs about a global fatal error causing the system to stop working.
	LogFatal
)

// log emits a log message.
func (s *NativeSystem) log(level int, msg string) {
	_ = s.EmitEvent(LifecycleLog, time.Now().UnixNano(), int64(level), msg)
}

func (s *NativeSystem) LogDebug(msg string) {
	s.log(LogDebug, msg)
}

func (s *NativeSystem) LogInfo(msg string) {
	s.log(LogInfo, msg)
}

func (s *NativeSystem) LogWarning(msg string) {
	s.log(LogWarning, msg)
}

func (s *NativeSystem) LogError(err error) {
	s.log(LogError, err.Error())
}

func (s *NativeSystem) LogFatal(err error) {
	s.log(LogFatal, err.Error())
}

// SYSTEM

type systemWrapper struct {
	h     *NativeNode
	creds Credentials
}

var _ Middleware = &systemWrapper{}

func (s systemWrapper) Name() string {
	return "system"
}

func (s systemWrapper) Middleware(ext any, val HubItem, out bool) (HubItem, error) {
	if out {
		sys, _ := val.(System)
		if sys == nil {
			return nil, nil
		}
		return sys.ID().Hex(), nil
	} else {
		if val == nil {
			return nil, nil
		}
		sys, err := s.h.GetSystemByID(s.creds, ParseSystemID(val.(string)))
		if err != nil {
			return nil, err
		}
		return sys, nil
	}
}

// SYSTEM IMPLEMENTATION

type SystemExtension interface {
	Implementation() Implementation
}

func WaitFor(sys System, status int) {
	if sys.Status() == status {
		return
	}
	ch := make(chan bool)
	sys.AddCallback(LifecycleStatus, NewNativeEvent(func(vals ...HubItem) error {
		newStatus := vals[0].(int64)
		if int(newStatus) == status {
			ch <- true
		}
		return nil
	}))
	<-ch
}
