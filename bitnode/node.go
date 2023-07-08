package bitnode

import (
	"encoding/json"
	"fmt"
	"github.com/Bitspark/go-bitnode/store"
	"github.com/Bitspark/go-bitnode/util"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type NodeAddress struct {
	Network string `json:"network"`
	Address string `json:"address"`
}

type Node interface {
	Name() string

	Description() string

	Created() time.Time

	Factories() []Factory

	AddSystem(sys *NativeSystem) error

	NewSystem(creds Credentials, m Sparkable, payload ...HubItem) (System, error)

	BlankSystem(name string) (*NativeSystem, error)

	PrepareSystem(creds Credentials, m Sparkable) (System, error)

	GetSystemByID(creds Credentials, id SystemID) (System, error)

	GetSystemByName(creds Credentials, name string) (System, error)

	System(creds Credentials) System

	Systems(creds Credentials) []System

	Addresses(creds Credentials) []NodeAddress

	SetAddress(creds Credentials, network string, address string)

	AddMiddlewares(mws Middlewares)

	Middlewares() Middlewares

	Load(st store.Store, dom *Domain) error

	Store(st store.Store) error
}

type NativePermissions struct {
	Admins ObjectIDs
}

type NativeNode struct {
	name        string
	description string
	created     time.Time
	addresses   map[string]string
	system      *NativeSystem
	systems     map[SystemID]*NativeSystem
	systemsMux  sync.Mutex
	factories   map[string]Factory
	middlewares Middlewares
}

var _ Node = &NativeNode{}

// TODO: Remove domain from node: Nodes should not care about domains, they just get Blueprints etc. directly.

func NewNode() *NativeNode {
	nodeName, _ := os.Hostname()
	nativeNode := &NativeNode{
		name:        util.RandomString(util.CharsAlphaNum, 8),
		description: fmt.Sprintf("Node %s, program %s", nodeName, strings.Join(os.Args, " ")),
		created:     time.Now(),
		addresses:   map[string]string{},
		system:      nil,
		systems:     map[SystemID]*NativeSystem{},
		factories:   map[string]Factory{},
		middlewares: Middlewares{},
	}
	return nativeNode
}

func (h *NativeNode) Name() string {
	return h.name
}

func (h *NativeNode) Description() string {
	return h.description
}

func (h *NativeNode) Created() time.Time {
	return h.created
}

func (h *NativeNode) Addresses(creds Credentials) []NodeAddress {
	addrs := []NodeAddress{}
	for network, address := range h.addresses {
		addrs = append(addrs, NodeAddress{
			Network: network,
			Address: address,
		})
	}
	return addrs
}

func (h *NativeNode) SetAddress(creds Credentials, network string, address string) {
	h.addresses[network] = address
}

func (h *NativeNode) System(creds Credentials) System {
	return h.system.Wrap(creds, h.middlewares)
}

func (h *NativeNode) Systems(creds Credentials) []System {
	syss := []System{}
	for _, sys := range h.systems {
		syss = append(syss, sys.Wrap(creds, h.middlewares))
	}
	return syss
}

func (h *NativeNode) Factories() []Factory {
	facs := []Factory{}
	for _, f := range h.factories {
		facs = append(facs, f)
	}
	return facs
}

// NewSystem creates a new blank system from an interface on this node and attaches it to the node.
func (h *NativeNode) NewSystem(creds Credentials, m Sparkable, payload ...HubItem) (System, error) {
	sys, err := h.PrepareSystem(creds, m)
	if err != nil {
		return nil, err
	}

	go func(sys *CredSystem) {
		// Trigger creation.
		if err := sys.EmitEvent(LifecycleCreate, payload...); err != nil {
			sys.LogError(err)
		}

		// Trigger loading.
		if err := sys.EmitEvent(LifecycleLoad); err != nil {
			sys.LogError(err)
		}
	}(sys.(*CredSystem))

	return sys.(*CredSystem).Wrap(creds, h.middlewares), nil
}

// PrepareSystem creates a new blank system from an interface on this node and attaches it to the node.
func (h *NativeNode) PrepareSystem(creds Credentials, m Sparkable) (System, error) {
	name := ""
	if m.Name != "" {
		name = fmt.Sprintf("%s %s", m.Name, GenerateSystemID().Hex()[:4])
	}

	sys, err := h.BlankSystem(name)
	if err != nil {
		return nil, err
	}

	if err := h.ImplementSystem(sys, m); err != nil {
		return nil, err
	}

	return sys.Wrap(creds, h.middlewares), nil
}

func (h *NativeNode) BlankSystem(name string) (*NativeSystem, error) {
	id := GenerateSystemID()

	// Create the system.
	sys := &NativeSystem{
		node:       h,
		id:         id,
		name:       name,
		systems:    map[SystemID]*NativeSystem{},
		origins:    map[string]*NativeSystem{},
		created:    time.Now(),
		events:     map[string]*LifecycleEvent{},
		logs:       util.NewSorted[int64, LogMessage](),
		extensions: []FactoryExtension{},
	}

	if err := h.initSystem(sys); err != nil {
		return nil, err
	}

	// Add the system to this node.
	h.systemsMux.Lock()
	h.systems[sys.id] = sys
	h.systemsMux.Unlock()

	return sys, nil
}

func (h *NativeNode) DefineSystem(sys *NativeSystem, i *Interface) error {
	if i == nil {
		return nil
	}

	sys.extends = i.CompiledExtends

	// Add hubs if interface is present.
	if i.CompiledHubs != nil {
		for _, p := range *i.CompiledHubs {
			hub := NewHub(sys, p)
			sys.hubs = append(sys.hubs, hub)
		}
	}

	return nil
}

func (h *NativeNode) ImplementSystem(sys *NativeSystem, m Sparkable) error {
	if err := h.DefineSystem(sys, m.Interface); err != nil {
		return err
	}

	sys.sparkable = m
	sys.extends = append(sys.extends, m.Domain+DomSep+m.Name+"$")

	// Implement the system.
	if err := m.Implement(h, sys.Wrap(Credentials{}, h.middlewares)); err != nil {
		return err
	}

	return nil
}

func (h *NativeNode) initSystem(s *NativeSystem) error {
	s.AddCallback(LifecycleName, NewNativeEvent(func(vals ...HubItem) error {
		oldName := s.name
		name := vals[0].(string)
		s.name = name
		log.Printf("%s name: %s", oldName, s.name)
		return nil
	}))

	s.AddCallback(LifecycleStatus, NewNativeEvent(func(vals ...HubItem) error {
		status := vals[0].(int64)
		s.status = int(status)
		log.Printf("%s status: %d", s.name, s.status)
		return nil
	}))

	s.AddCallback(LifecycleLog, NewNativeEvent(func(vals ...HubItem) error {
		logTimestampNano := vals[0].(int64)
		level := vals[1].(int64)
		msg := vals[2].(string)
		logTime := time.Unix(logTimestampNano/1e9, logTimestampNano%1e9)
		s.logs.Add(logTimestampNano, LogMessage{
			Level:   int(level),
			Time:    logTime,
			Message: msg,
		})
		log.Printf("[%d] %s", level, msg)
		return nil
	}))

	s.AddCallback(LifecycleDelete, NewNativeEvent(func(vals ...HubItem) error {
		h.systemsMux.Lock()
		delete(h.systems, s.id)
		h.systemsMux.Unlock()

		return nil
	}))

	return nil
}

// AddSystem attaches a system to this node.
func (h *NativeNode) AddSystem(sys *NativeSystem) error {
	if _, ok := h.systems[sys.ID()]; ok {
		return fmt.Errorf("already have a system with id %s: %s", sys.ID(), sys.Name())
	}
	h.systems[sys.ID()] = sys
	return nil
}

// SetSystem sets the root system of the node.
func (h *NativeNode) SetSystem(sys *NativeSystem) {
	h.system = sys
}

func (h *NativeNode) AddFactory(name string, f Factory) error {
	if _, ok := h.factories[name]; ok {
		return fmt.Errorf("factory already set: %s", name)
	}
	h.factories[name] = f
	return nil
}

func (h *NativeNode) GetFactory(name string) (Factory, error) {
	f, ok := h.factories[name]
	if !ok {
		return nil, fmt.Errorf("factory not found: %s", name)
	}
	return f, nil
}

func (h *NativeNode) GetSystemByID(creds Credentials, id SystemID) (System, error) {
	if id.IsNull() {
		if h.system == nil {
			return nil, fmt.Errorf("have no root system")
		}
		return h.System(creds), nil
	}
	if sys, ok := h.systems[id]; ok {
		return sys.Wrap(creds, h.middlewares), nil
	}
	return nil, fmt.Errorf("system not found: %s", id.Hex())
}

func (h *NativeNode) GetSystemByName(creds Credentials, name string) (System, error) {
	if name == "" {
		if h.system == nil {
			return nil, fmt.Errorf("have no root system")
		}
		return h.System(creds), nil
	}
	for _, sys := range h.systems {
		if sys.Name() == name {
			return sys.Wrap(creds, h.middlewares), nil
		}
	}
	return nil, fmt.Errorf("system not found: %s", name)
}

// Load loads a saved node state.
func (h *NativeNode) Load(st store.Store, dom *Domain) error {
	systemStoreDS, err := st.Ensure("systems", store.DSStores)
	if err != nil {
		return err
	}
	systemStore := systemStoreDS.Stores()

	stSys := map[store.Store]*NativeSystem{}

	for st := range systemStore.Enumerate() {
		sys := &NativeSystem{
			extensions: []FactoryExtension{},
			origins:    map[string]*NativeSystem{},
			node:       h,
		}
		if err := sys.LoadInit(h, st); err != nil {
			return err
		}
		h.systems[sys.ID()] = sys
		stSys[st] = sys
	}

	for st := range systemStore.Enumerate() {
		sys := stSys[st]
		if err := sys.Load(h, dom, st); err != nil {
			return err
		}
	}

	for _, sys := range h.systems {
		for chSysID := range sys.systems {
			sys.systems[chSysID] = h.systems[chSysID]
		}

		go func(sys *NativeSystem) {
			if err := sys.EmitEvent(LifecycleLoad); err != nil {
				log.Printf("Error loading %s: %v", sys.Name(), err)
			}
		}(sys)
	}

	nodeStoreDS, err := st.Ensure("node", store.DSKeyValue)
	if err != nil {
		return err
	}
	nodeStore := nodeStoreDS.KeyValue()

	name, _ := nodeStore.Get("name")
	if name != "" {
		h.name = name
	}

	created, _ := nodeStore.Get("created")
	if created != "" {
		_ = json.Unmarshal([]byte(created), &h.created)
	}

	system, _ := nodeStore.Get("system")
	if system != "" {
		h.system = h.systems[ParseSystemID(system)]
	}

	addressesDS, err := st.Ensure("addresses", store.DSKeyValue)
	if err != nil {
		return err
	}
	addresses := addressesDS.KeyValue()
	for network := range addresses.EnumerateKeys() {
		h.addresses[network], _ = addresses.Get(network)
	}

	return nil
}

// Store stores the node state.
func (h *NativeNode) Store(st store.Store) error {
	systemStoreDS, err := st.Ensure("systems", store.DSStores)
	if err != nil {
		return err
	}
	systemStore := systemStoreDS.Stores()

	for _, sys := range h.systems {
		if sys == nil {
			continue
		}
		st := store.NewStore(sys.ID().Hex())
		if err := sys.Store(st); err != nil {
			return err
		}
		if err := systemStore.Add(st); err != nil {
			return err
		}

		// Trigger storing.
		sys.EmitEvent(LifecycleStore)
	}

	nodeStoreDS, err := st.Ensure("node", store.DSKeyValue)
	if err != nil {
		return err
	}
	nodeStore := nodeStoreDS.KeyValue()

	_ = nodeStore.Set("name", h.name)

	created, _ := json.Marshal(h.created)
	_ = nodeStore.Set("created", string(created))

	if h.system != nil {
		_ = nodeStore.Set("system", h.system.ID().Hex())
	}

	addressesDS, err := st.Ensure("addresses", store.DSKeyValue)
	if err != nil {
		return err
	}
	addresses := addressesDS.KeyValue()
	for network, addr := range h.addresses {
		if err := addresses.Set(network, addr); err != nil {
			return err
		}
	}

	return nil
}

func (h *NativeNode) AddMiddlewares(mws Middlewares) {
	h.middlewares = append(h.middlewares, mws...)
}

func (h *NativeNode) Middlewares() Middlewares {
	return h.middlewares
}
