package bitnode

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// A Sparkable for systems.
type Sparkable struct {
	compilable
	RawSparkable
}

var _ Implementation = &Sparkable{}
var _ Compilable = &Sparkable{}
var _ Savable = &Sparkable{}

type RawSparkable struct {
	// Name of the impl.
	Name string `json:"name" yaml:"name"`

	// Description of the sparkable.
	Description string `json:"description" yaml:"description"`

	// Permissions of this interface.
	Permissions *Permissions `json:"permissions,omitempty" yaml:"permissions,omitempty"`

	// Constructor of this sparkable specifying values required for creation of this system.
	Constructor HubItemsInterface `json:"constructor" yaml:"constructor"`

	// Reference to another sparkable.
	Reference string `json:"reference,omitempty" yaml:"reference,omitempty"`

	// Interface specifies the interface the sparkable implements.
	Interface *Interface `json:"interface" yaml:"interface"`

	// Implementation contains implementations.
	Implementation map[string][]any `json:"implementation" yaml:"implementation"`

	// Domain this sparkable resides in.
	Domain string `json:"domain,omitempty" yaml:"-"`
}

type HubImplementations struct {
	Name string `json:"name" yaml:"name"`

	Implementation any `json:"implementation" yaml:"implementation"`
}

type ChildImpl struct {
	Name string `json:"name" yaml:"name"`

	Impl *Sparkable `json:"impl" yaml:"impl"`
}

func (m *Sparkable) Save(dom *Domain) error {
	dom, err := dom.GetDomain(m.Domain)
	if err != nil {
		return err
	}
	chDefsBytes, err := os.ReadFile(dom.FilePath)
	if err != nil {
		return fmt.Errorf("reading definitions from %s: %v", dom.FilePath, err)
	}

	defs := Domain{}
	if err := yaml.Unmarshal(chDefsBytes, &defs); err != nil {
		return fmt.Errorf("parsing definitions from %s: %v", dom.FilePath, err)
	}

	for _, bp := range defs.Sparkables {
		if bp.Name == m.Name {
			*bp = *m
		}
	}

	if yamlBts, err := yaml.Marshal(defs); err != nil {
		return fmt.Errorf("parsing definitions from %s: %v", dom.FilePath, err)
	} else {
		if err := os.WriteFile(dom.FilePath, yamlBts, os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

func (m *Sparkable) Reset() {
	m.compiled = false
	for _, c := range m.Constructor {
		c.Value.Reset()
	}
	if m.Interface != nil {
		m.Interface.Reset()
	}
}

func (m *Sparkable) FullDomain() string {
	return m.Domain
}

func (m *Sparkable) Compile(dom *Domain, domName string, resolve bool) error {
	if m == nil {
		return nil
	}
	if m.compiled {
		return nil
	}
	m.compiling = true
	defer func() {
		m.compiling = false
	}()

	if m.Domain == "" && domName != "" {
		m.Domain = domName
	}

	if m.Interface == nil {
		m.Interface = NewInterface()
	}
	if err := m.Interface.Compile(dom, domName, resolve); err != nil {
		return err
	}

	for i := range m.Constructor {
		if err := m.Constructor[i].Compile(dom, domName, resolve); err != nil {
			return err
		}
	}

	m.compiled = true

	return nil
}

func (m *Sparkable) Compiled() bool {
	return m.compiled
}

// HubItem is data that can be passed through hubs, possibly across systems and physical machines.
type HubItem any

var NilItem = HubItem(nil)

// Implement adds this implementation to the system sys.
func (m *Sparkable) Implement(node *NativeNode, sys System) error {
	// Add implementations.
	for fName, implDatas := range m.Implementation {
		f, err := node.GetFactory(fName)
		if err != nil {
			return err
		}
		for _, implData := range implDatas {
			implAny, _ := f.Implementation(nil)
			if err := implAny.FromInterface(implData); err != nil {
				return err
			}
			impl, err := f.Implementation(implAny)
			if err != nil {
				return err
			}
			if err := impl.Implement(node, sys); err != nil {
				return err
			}
		}
	}

	// Successful.
	return nil
}

func (m *Sparkable) Validate() error {
	panic("implement me")
}

func (m *Sparkable) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.RawSparkable)
}

func (m *Sparkable) UnmarshalJSON(data []byte) error {
	if m.Implementation == nil {
		m.Implementation = map[string][]any{}
	}
	return json.Unmarshal(data, &m.RawSparkable)
}

func (m *Sparkable) MarshalYAML() (interface{}, error) {
	return m.RawSparkable, nil
}

func (m *Sparkable) UnmarshalYAML(value *yaml.Node) error {
	if m.Implementation == nil {
		m.Implementation = map[string][]any{}
	}
	return value.Decode(&m.RawSparkable)
}

func (m *Sparkable) ToInterface() (any, error) {
	dat, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	datMp := map[string]any{}
	if err := json.Unmarshal(dat, &datMp); err != nil {
		return nil, err
	}
	return datMp, nil
}

func (m *Sparkable) FromInterface(i any) error {
	iMp := i.(map[string]any)
	if iMp["domain"] != nil {
		m.Domain = iMp["domain"].(string)
	}
	bpMpImpl := iMp["implementation"]
	m.Implementation = map[string][]any{}
	if bpMpImpl != nil {
		for f, impls := range bpMpImpl.(map[string]any) {
			for _, impl := range impls.([]any) {
				m.Implementation[f] = append(m.Implementation[f], impl)
			}
		}
	}
	delete(iMp, "implementation")
	delete(iMp, "compiledImplementation")
	interf := iMp["interface"]
	delete(iMp, "interface")
	dat, err := json.Marshal(iMp)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(dat, m); err != nil {
		return err
	}
	m.Interface = &Interface{}
	if interf != nil {
		if err := m.Interface.FromInterface(interf); err != nil {
			return err
		}
	}
	return nil
}
