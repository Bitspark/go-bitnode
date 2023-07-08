package bitnode

import (
	"encoding/json"
	"fmt"
	"github.com/Bitspark/go-bitnode/store"
	"github.com/Bitspark/go-bitnode/util"
	"log"
	"strings"
	"sync"
	"time"
)

const (
	LifecycleCreate = "create"
	LifecycleLoad   = "load"
	LifecycleStop   = "stop"
	LifecycleStart  = "start"
	LifecycleDelete = "delete"

	LifecycleStore = "store"

	LifecycleName   = "name"
	LifecycleStatus = "status"
	LifecycleLog    = "log"
)

const (
	SystemStatusUndefined = 0

	SystemStatusImplementing = 1 << (iota - 1)
	SystemStatusImplemented

	SystemStatusCreating
	SystemStatusCreated

	SystemStatusLoading
	SystemStatusLoaded

	SystemStatusStopping
	SystemStatusStarting

	SystemStatusRunning

	SystemStatusDeleting
	SystemStatusDeleted
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

type NativeLink struct {
	Name   string
	Origin *NativeSystem
}

type Origin struct {
	Name   string
	Origin System
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

	// Stop stops the system.
	Stop(timeout float64)

	// Start starts the system.
	Start()

	// Delete deletes the system and kills it if necessary.
	Delete()

	// SetName changes the name of the system.
	SetName(name string)

	// SetStatus sets the provided status on top of the current status. Negative values are unset.
	SetStatus(status int)

	// GetHub returns a hub of this system.
	GetHub(hubName string) Hub

	// Hubs returns all hubs of this system.
	Hubs() []Hub

	// An Origins returns the holograms serving as origin.
	Origins() []Origin

	// An Origin is a hologram which serves as master for this System.
	Origin(name string) System

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
	AddExtension(name string, impl FactorySystem)

	// SetExtension sets a system extension of the system.
	SetExtension(name string, impl FactorySystem)

	// Extensions returns extensions by name.
	Extensions(name string) []FactorySystem

	// Extension returns an extension by name.
	Extension(name string) FactorySystem

	// AddSystem attaches a system.
	AddSystem(sys *NativeSystem) error

	// Systems returns child systems.
	Systems() []System

	// Sparkable returns the sparkable with can be used to re-create the same system.
	Sparkable() (*Sparkable, error)

	Native() *NativeSystem

	RemoteID() SystemID

	RemoteNode() string

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

type FactoryExtension struct {
	Factory string
	System  FactorySystem
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

	origins map[string]*NativeSystem

	parents []NativeLink

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

	extensions []FactoryExtension

	status int

	remoteID SystemID

	remoteNode string

	eventsMux sync.Mutex

	implMux sync.Mutex
}

// SystemInfo stores information about a system.
type SystemInfo struct {
	// Status of the system.
	Status int `json:"status" yaml:"status"`

	// Logs of the system.
	Logs util.Sorted[int64, LogMessage]
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

func (s *NativeSystem) Stop(creds Credentials, timeout float64) {
	_ = s.EmitEvent(LifecycleStop, timeout)
}

func (s *NativeSystem) Start(creds Credentials) {
	_ = s.EmitEvent(LifecycleStart)
}

func (s *NativeSystem) Delete(creds Credentials) {
	_ = s.EmitEvent(LifecycleDelete)
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

func (s *NativeSystem) Sparkable() (*Sparkable, error) {
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
	for _, ms := range s.extensions {
		impl := ms.System.Implementation()
		if impl == nil {
			continue
		}
		impls, _ := bp.Implementation[ms.Factory]
		f, err := s.node.GetFactory(ms.Factory)
		if err != nil {
			return nil, err
		}
		implData, err := f.Serialize(impl)
		if err != nil {
			return nil, err
		}
		impls = append(impls, implData)
		bp.Implementation[ms.Factory] = impls
	}
	s.implMux.Unlock()

	return bp, nil
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

func (s *NativeSystem) Origins() []NativeLink {
	syss := []NativeLink{}
	for name, sys := range s.origins {
		syss = append(syss, NativeLink{
			Name:   name,
			Origin: sys,
		})
	}
	return syss
}

func (s *NativeSystem) Origin(name string) *NativeSystem {
	if name == "" {
		return s
	}
	return s.origin(strings.Split(name, "/"))
}

func (s *NativeSystem) origin(path []string) *NativeSystem {
	if len(path) == 0 {
		return s
	}
	orig, _ := s.origins[path[0]]
	if orig == nil {
		return nil
	}
	return orig.origin(path[1:])
}

func (s *NativeSystem) AddOrigin(name string, origin *NativeSystem) {
	s.origins[name] = origin
	origin.parents = append(origin.parents, NativeLink{
		Name:   name,
		Origin: s,
	})
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
	preStatus := SystemStatusUndefined
	postStatus := SystemStatusUndefined

	switch name {
	case LifecycleCreate:
		preStatus = SystemStatusCreating
		postStatus = SystemStatusCreated

	case LifecycleLoad:
		preStatus = SystemStatusLoading
		postStatus = SystemStatusLoaded

	case LifecycleStart:
		preStatus = SystemStatusStarting
		postStatus = SystemStatusRunning

	case LifecycleStop:
		preStatus = SystemStatusStopping
		postStatus = -SystemStatusRunning

	case LifecycleDelete:
		preStatus = SystemStatusDeleting
		postStatus = SystemStatusDeleted
	}

	oldStatus := s.Status()

	if preStatus != SystemStatusUndefined {
		s.SetStatus(Credentials{}, oldStatus|preStatus)
	}

	s.eventsMux.Lock()
	events, _ := s.events[name]
	s.eventsMux.Unlock()
	if events != nil {
		for _, cb := range events.Callbacks {
			if err := cb.CB(args...); err != nil {
				if preStatus != SystemStatusUndefined {
					s.SetStatus(Credentials{}, oldStatus & ^preStatus)
				}
				return err
			}
		}
	}

	if preStatus != SystemStatusUndefined || postStatus != SystemStatusUndefined {
		if postStatus >= 0 {
			s.SetStatus(Credentials{}, (s.Status() & ^preStatus)|postStatus)
		} else {
			s.SetStatus(Credentials{}, (s.Status() & ^preStatus) & ^(-postStatus))
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

type origSt struct {
	Name   string   `json:"name"`
	Origin SystemID `json:"systemId"`
}

func (s *NativeSystem) Store(st store.Store) error {
	systemStoreDS, _ := st.Ensure("system", store.DSKeyValue)
	systemStore := systemStoreDS.KeyValue()

	_ = systemStore.Set("id", s.id.Hex())
	_ = systemStore.Set("name", s.name)
	_ = systemStore.Set("extends", strings.Join(s.extends, ","))
	_ = systemStore.Set("remoteNode", s.remoteNode)
	_ = systemStore.Set("remoteID", s.remoteID.Hex())

	bp, _ := s.Sparkable()
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

	origs := []origSt{}
	for name, o := range s.origins {
		origs = append(origs, origSt{
			Name:   name,
			Origin: o.ID(),
		})
	}
	originsBts, _ := json.Marshal(origs)
	_ = systemStore.Set("origins", string(originsBts))

	return nil
}

// LoadInit loads all information which does not require other systems.
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
	s.remoteNode, _ = systemStore.Get("remoteNode")
	remoteIDStr, _ := systemStore.Get("remoteID")
	s.remoteID = ParseSystemID(remoteIDStr)

	extends, _ := systemStore.Get("extends")
	s.extends = strings.Split(extends, ",")

	return nil
}

// Load loads parts of the system state which require other systems (e.g., hub values).
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

	origins, _ := systemStore.Get("origins")
	origs := []origSt{}
	_ = json.Unmarshal([]byte(origins), &origs)
	for _, o := range origs {
		orig, err := node.GetSystemByID(Credentials{}, o.Origin)
		if err != nil {
			return err
		}
		s.AddOrigin(o.Name, orig.Native())
	}

	return nil
}

func (s *NativeSystem) AddExtension(name string, ext FactorySystem) {
	s.implMux.Lock()
	defer s.implMux.Unlock()
	s.extensions = append(s.extensions, FactoryExtension{
		Factory: name,
		System:  ext,
	})
}

func (s *NativeSystem) SetExtension(name string, ext FactorySystem) {
	for i, m := range s.extensions {
		if m.Factory == name {
			s.extensions[i].System = ext
			return
		}
	}
	s.AddExtension(name, ext)
}

func (s *NativeSystem) Extensions(name string) []FactorySystem {
	exts := []FactorySystem{}
	s.implMux.Lock()
	defer s.implMux.Unlock()
	for _, m := range s.extensions {
		if m.Factory == name {
			exts = append(exts, m.System)
		}
	}
	return exts
}

func (s *NativeSystem) Extension(name string) FactorySystem {
	exsts := s.Extensions(name)
	if len(exsts) != 1 {
		return nil
	}
	return exsts[0]
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

func (s *NativeSystem) SetRemoteID(id SystemID) {
	s.remoteID = id
}

func (s *NativeSystem) SetRemoteNode(node string) {
	s.remoteNode = node
}

func (s *NativeSystem) RemoteID() SystemID {
	return s.remoteID
}

func (s *NativeSystem) RemoteNode() string {
	return s.remoteNode
}

func (s *NativeSystem) RedirectFrom(origin *NativeSystem) {
	for _, origHub := range origin.Hubs(Credentials{}) {
		func(origHub *NativeHub) {
			hubInterf := origHub.Interface()
			hub := s.GetHub(Credentials{}, nil, hubInterf.Name)
			if hub == nil {
				nativeHub := &NativeHub{
					parent:       s,
					hubInterface: hubInterf,
				}
				s.hubs = append(s.hubs, nativeHub)
				return
			}
			nativeHub := hub.Native()

			switch hubInterf.Type {
			case HubTypePipe:
				_ = nativeHub.Handle(NewNativeFunction(func(user Credentials, vals ...HubItem) ([]HubItem, error) {
					return origHub.Invoke(user, nil, vals...)
				}))

			case HubTypeValue:
				_, _ = nativeHub.Subscribe(Credentials{}, nil, NewNativeSubscription(func(id string, creds Credentials, val HubItem) {
					_ = origHub.Set(creds, nil, id, val)
				}))
				_, _ = origHub.Subscribe(Credentials{}, nil, NewNativeSubscription(func(id string, creds Credentials, val HubItem) {
					_ = nativeHub.Set(creds, nil, id, val)
				}))

			case HubTypeChannel:
				_, _ = nativeHub.Subscribe(Credentials{}, nil, NewNativeSubscription(func(id string, creds Credentials, val HubItem) {
					_ = origHub.Emit(creds, nil, id, val)
				}))
				_, _ = origHub.Subscribe(Credentials{}, nil, NewNativeSubscription(func(id string, creds Credentials, val HubItem) {
					_ = nativeHub.Emit(creds, nil, id, val)
				}))
			}
		}(origHub.Native())
	}
}

type CredSystem struct {
	*NativeSystem
	creds       Credentials
	middlewares Middlewares
}

var _ System = &CredSystem{}

func (s *CredSystem) Stop(timeout float64) {
	s.NativeSystem.Stop(s.creds, timeout)
}

func (s *CredSystem) Start() {
	s.NativeSystem.Start(s.creds)
}

func (s *CredSystem) Delete() {
	s.NativeSystem.Delete(s.creds)
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

func (s *CredSystem) Origins() []Origin {
	syss := []Origin{}
	for n, sys := range s.origins {
		syss = append(syss, Origin{Name: n, Origin: sys.Wrap(s.creds, s.middlewares)})
	}
	return syss
}

func (s *CredSystem) Origin(name string) System {
	orig := s.NativeSystem.Origin(name)
	if orig == nil {
		return nil
	}
	return orig.Wrap(s.creds, s.middlewares)
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

type FactorySystem interface {
	// Root returns the root system to which this FactorySystem is attached.
	//Root() System

	// The Factory which has created this FactoryExtension.
	//Factory() Factory

	// Implementation returns the FactoryImplementation of this FactorySystem.
	Implementation() FactoryImplementation
}

func WaitFor(sys System, status int) {
	if sys.Status()&status == status {
		return
	}
	ch := make(chan bool)
	sys.AddCallback(LifecycleStatus, NewNativeEvent(func(vals ...HubItem) error {
		newStatus := vals[0].(int64)
		if int(newStatus)&status == status {
			ch <- true
		}
		return nil
	}))
	<-ch
}
