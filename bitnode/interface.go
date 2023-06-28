package bitnode

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"strings"
)

// Interface

type Interface struct {
	compilable
	RawInterface

	references map[Compilable]bool
}

type RawInterface struct {
	// Name of the interface.
	Name string `json:"name" yaml:"name,omitempty"`

	// FullName of this interface, can be used to reference it from elsewhere.
	FullName string `json:"fullName,omitempty" yaml:"-"`

	// Description of the interface.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Permissions of this interface.
	Permissions *Permissions `json:"permissions,omitempty" yaml:"permissions,omitempty"`

	// Extends contains interfaces this interface extends.
	Extends []string `json:"extends" yaml:"extends,omitempty"`

	// Domain this interface resides in.
	Domain string `json:"domain,omitempty" yaml:"-"`

	// Hubs are interactions this interface provides.
	Hubs *HubInterfaces `json:"hubs,omitempty" yaml:"hubs,omitempty"`

	// CompiledHubs are all hubs this interface has, including compiled ones.
	CompiledHubs *HubInterfaces `json:"compiledHubs" yaml:"-"`

	// Compiled extends contains compiled interfaces this interface extends.
	CompiledExtends []string `json:"compiledExtends" yaml:"-"`

	// Extensions of this interface.
	Extensions map[string]bool `json:"extensions" yaml:"-"`
}

var _ Compilable = &Interface{}
var _ MarshalInterface = &Interface{}
var _ Savable = &Interface{}

func NewInterface() *Interface {
	return &Interface{RawInterface: RawInterface{
		Hubs:         &HubInterfaces{},
		CompiledHubs: &HubInterfaces{},
	}}
}

func (i *Interface) Save(dom *Domain) error {
	interf, err := dom.GetDomain(i.Domain)
	if err != nil {
		return err
	}
	chDefsBytes, err := os.ReadFile(interf.FilePath)
	if err != nil {
		return fmt.Errorf("reading definitions from %s: %v", interf.FilePath, err)
	}

	defs := Domain{}
	if err := yaml.Unmarshal(chDefsBytes, &defs); err != nil {
		return fmt.Errorf("parsing definitions from %s: %v", interf.FilePath, err)
	}

	for _, inf := range defs.Interfaces {
		if inf.Name == i.Name {
			*inf = *i
		}
	}

	if yamlBts, err := yaml.Marshal(defs); err != nil {
		return fmt.Errorf("parsing definitions from %s: %v", interf.FilePath, err)
	} else {
		if err := os.WriteFile(interf.FilePath, yamlBts, os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

// Blank returns an empty sparkable created from this interface.
func (i *Interface) Blank() Sparkable {
	return Sparkable{RawSparkable: RawSparkable{
		Name:      "Blank" + i.Name,
		Interface: i,
		Domain:    i.Domain,
	}, compilable: compilable{
		compiled: true,
	}}
}

func (i *Interface) Reset() {
	i.compiled = false

	if i.Hubs != nil {
		for _, h := range *i.Hubs {
			if h.Value != nil {
				h.Value.Reset()
			}
			for _, v := range h.Input {
				v.Reset()
			}
			for _, v := range h.Output {
				v.Reset()
			}
		}
	}

	// Reset references.
	for ri := range i.references {
		ri.Reset()
	}

	i.CompiledHubs = nil
	i.CompiledExtends = nil
}

func (i *Interface) Compile(dom *Domain, domName string, resolve bool) error {
	if i == nil {
		return nil
	}
	if i.compiled {
		return nil
	}
	i.compiling = true
	defer func() {
		i.compiling = false
	}()
	if i.references == nil {
		i.references = map[Compilable]bool{}
	}

	if domName != "" {
		i.Domain = domName
	}

	if i.Domain != "" && i.Name != "" {
		i.FullName = i.Domain + "." + i.Name
	}

	i.CompiledHubs = &HubInterfaces{}

	if resolve && len(i.Extends) > 0 {
		cdom, _ := dom.GetDomain(domName)
		if cdom == nil {
			return fmt.Errorf("domain not set")
		}
		for _, ext := range i.Extends {
			ri, err := cdom.GetInterface(ext)
			if err != nil {
				log.Printf("interface %s: %v", ext, err)
			}
			if ri == nil {
				return fmt.Errorf("interface not found: %s", i.Extends)
			}
			if !ri.compiled {
				if err := ri.Compile(dom, ri.Domain, resolve); err != nil {
					return err
				}
			}
			ri.references[i] = true
			i.CompiledExtends = append(i.CompiledExtends, ri.CompiledExtends...)
			i.CompiledExtends = append(i.CompiledExtends, ri.FullName)
			if err := i.CompiledHubs.Extend(ri.CompiledHubs); err != nil {
				return err
			}
			if ri.Extensions == nil {
				ri.Extensions = map[string]bool{}
			}
			ri.Extensions[i.FullName] = true
		}
	}

	if i.Hubs != nil {
		for _, hub := range *i.Hubs {
			if hub.Type == "" {
				return fmt.Errorf("hub %s requires type", hub.Name)
			}
			if hub.Direction == "" {
				return fmt.Errorf("hub %s requires direction", hub.Name)
			}

			hub.Interface = i.FullName
			if hub.Input != nil {
				for _, h := range hub.Input {
					if err := h.Compile(dom, domName, resolve); err != nil {
						return err
					}
				}
			}
			if hub.Output != nil {
				for _, h := range hub.Output {
					if err := h.Compile(dom, domName, resolve); err != nil {
						return err
					}
				}
			}
			if hub.Value != nil {
				if err := hub.Value.Compile(dom, domName, resolve); err != nil {
					return err
				}
			}
			*i.CompiledHubs = append(*i.CompiledHubs, hub)
		}
	}

	i.compiled = true

	// Compile references.
	for rt := range i.references {
		if err := rt.Compile(dom, rt.FullDomain(), true); err != nil {
			return err
		}
	}

	return nil
}

func (i *Interface) FullDomain() string {
	return i.Domain
}

func (i *Interface) Extend(base *Interface) error {
	if base == nil || base.CompiledHubs == nil {
		return nil
	}
	for _, p := range *base.CompiledHubs {
		if err := i.CompiledHubs.AddHub(p); err != nil {
			return err
		}
	}
	return nil
}

func (i *Interface) Validate(val HubItem) (any, error) {
	// TODO: Implement
	return val, nil
}

func (i *Interface) String() string {
	is := "interface"
	if i.Name != "" {
		is += " " + i.Name
	}
	is += " {"
	if i.Hubs != nil {
		is += "\n"
		for _, h := range *i.Hubs {
			if h.Input != nil && h.Output != nil {
				is += "  " + h.Input.String() + " -> " + h.Name + " -> " + h.Output.String() + "\n"
			} else if h.Input != nil && h.Output == nil {
				is += "  " + h.Input.String() + " -> " + h.Name + "\n"
			} else if h.Input == nil && h.Output != nil {
				is += "  " + h.Name + " -> " + h.Output.String() + "\n"
			} else if h.Input == nil && h.Output == nil {
				is += "  " + h.Name + "\n"
			}
		}
	}
	is += "}"
	return is
}

func (i *Interface) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.RawInterface)
}

func (i *Interface) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		if len(str) >= 2 && str[0] == '$' {
			exts := str[1:]
			if exts[0] == '[' && exts[len(exts)-1] == ']' {
				for _, ext := range strings.Split(exts, ",") {
					ext = strings.TrimSpace(ext)
					i.Extends = append(i.Extends, ext)
				}
			} else {
				i.Extends = []string{exts}
			}
			return nil
		}
		return fmt.Errorf("expected interface reference: %s", str)
	}
	return json.Unmarshal(data, &i.RawInterface)
}

func (i *Interface) MarshalYAML() (interface{}, error) {
	return i.RawInterface, nil
}

func (i *Interface) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err == nil {
		if len(str) >= 2 && str[0] == '$' {
			exts := str[1:]
			if exts[0] == '[' && exts[len(exts)-1] == ']' {
				for _, ext := range strings.Split(exts, ",") {
					ext = strings.TrimSpace(ext)
					i.Extends = append(i.Extends, ext)
				}
			} else {
				i.Extends = []string{exts}
			}
			return nil
		}
		return fmt.Errorf("expected interface reference: %s", str)
	}
	return value.Decode(&i.RawInterface)
}

func (i *Interface) FromInterface(a any) error {
	dat, err := json.Marshal(a)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(dat, i); err != nil {
		return err
	}
	return nil
}

func (i *Interface) ToInterface() (any, error) {
	dat, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	var datMp any
	if err := json.Unmarshal(dat, &datMp); err != nil {
		return nil, err
	}
	return datMp, nil
}

func (i *Interface) Contains(i2 *Interface) error {
	if i == nil || i.CompiledHubs == nil {
		if i2 != nil && i2.CompiledHubs != nil {
			return fmt.Errorf("nil interface contains nothing")
		}
		return nil
	}
	return i.CompiledHubs.Contains(i2.CompiledHubs)
}

// HubInterface

type HubInterface struct {
	Name        string            `json:"name" yaml:"name"`
	Type        HubType           `json:"type" yaml:"type"`
	Direction   HubDirection      `json:"direction" yaml:"direction"`
	Description string            `json:"description" yaml:"description"`
	Input       HubItemsInterface `json:"input" yaml:"input"`
	Output      HubItemsInterface `json:"output" yaml:"output"`
	Value       *HubItemInterface `json:"value" yaml:"value"`
	Interface   string            `json:"interface" yaml:"-"`
}

func (i *HubInterface) Contains(i2 *HubInterface) error {
	return nil
}

func (i *HubInterface) MarshalYAML() (interface{}, error) {
	mp := map[string]any{
		"name":        i.Name,
		"type":        i.Type,
		"direction":   i.Direction,
		"description": i.Description,
	}
	if i.Input != nil {
		mp["input"] = i.Input
	}
	if i.Output != nil {
		mp["output"] = i.Output
	}
	if i.Value != nil {
		mp["value"] = i.Value
	}
	return mp, nil
}

type HubInterfaces []*HubInterface

var _ MarshalInterface = &HubInterfaces{}

func (i *HubInterfaces) AddHub(hub *HubInterface) error {
	if hub := i.GetHub(hub.Name); hub != nil {
		return fmt.Errorf("already have hub with that name: %s", hub.Name)
	}
	*i = append(*i, hub)
	return nil
}

func (i *HubInterfaces) GetHub(name string) *HubInterface {
	for _, hub := range *i {
		if hub.Name == name {
			return hub
		}
	}
	return nil
}

func (i *HubInterfaces) Extend(add *HubInterfaces) error {
	for _, p := range *add {
		if err := i.AddHub(p); err != nil {
			return err
		}
	}
	return nil
}

func (i *HubInterfaces) FromInterface(a any) error {
	dat, err := yaml.Marshal(a)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(dat, i); err != nil {
		return err
	}
	return nil
}

func (i *HubInterfaces) ToInterface() (any, error) {
	dat, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	datMp := map[string]any{}
	if err := json.Unmarshal(dat, &datMp); err != nil {
		return nil, err
	}
	return datMp, nil
}

func (i *HubInterfaces) Contains(interf *HubInterfaces) error {
	if i == nil || len(*i) == 0 {
		if interf != nil && len(*interf) != 0 {
			return fmt.Errorf("hub interfaces not not match empty interfaces")
		}
		return nil
	}
	if interf != nil {
		for _, interfHub := range *interf {
			hub := i.GetHub(interfHub.Name)
			if hub == nil {
				return fmt.Errorf("missing hub: %s", interfHub.Name)
			}
			if interfHub.Input == nil {

			}
		}
	}
	return nil
}

// HubItemInterface

type HubItemInterface struct {
	Name        string `json:"name,omitempty" yaml:"name,omitempty" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	Value *Type `json:"value,omitempty" yaml:"value,omitempty"`

	Domain string `json:"-" yaml:"-"`
}

var _ Compilable = &HubItemInterface{}
var _ MarshalInterface = &HubItemInterface{}

func (i *HubItemInterface) Reset() {
	if i.Value != nil {
		i.Value.Reset()
	}
}

func (i *HubItemInterface) Compile(dom *Domain, domName string, resolve bool) error {
	if i.Value == nil {
		return nil
	}
	i.Domain = domName
	return i.Value.Compile(dom, domName, resolve)
}

func (i *HubItemInterface) FullDomain() string {
	return i.Domain
}

func (i *HubItemInterface) String() string {
	if i == nil {
		return ""
	}

	hStr := ""
	if i.Value != nil {
		hStr = i.Value.String()
	}

	if i.Name == "" {
		return hStr
	} else {
		return i.Name + ": " + hStr
	}
}

func (i *HubItemInterface) FromInterface(a any) error {
	dat, err := yaml.Marshal(a)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(dat, i); err != nil {
		return err
	}
	return nil
}

func (i *HubItemInterface) ToInterface() (any, error) {
	dat, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	datMp := map[string]any{}
	if err := json.Unmarshal(dat, &datMp); err != nil {
		return nil, err
	}
	return datMp, nil
}

func (i *HubItemInterface) Contains(h2 *HubItemInterface) error {
	if i == nil {
		if h2 != nil {
			return fmt.Errorf("nil hub interface does not match %s", h2.String())
		}
		return nil
	}
	if h2 == nil {
		return fmt.Errorf("hub interface %s does not match nil interface", i.String())
	}
	if i.Name != h2.Name {
		return fmt.Errorf("hub name %s does not name %s", i.Name, h2.Name)
	}
	if i.Value != nil {
		if h2.Value == nil {
			return fmt.Errorf("value hub interface does not match nil value interface")
		}
		return i.Value.Compiled.Contains(h2.Value.Compiled)
	}
	return nil
}

func (i *HubItemInterface) ApplyMiddlewares(mws Middlewares, val HubItem, out bool) (HubItem, error) {
	if i.Value == nil {
		return nil, fmt.Errorf("empty interface")
	}
	return i.Value.Compiled.ApplyMiddlewares(mws, val, out)
}

type HubItemsInterface []*HubItemInterface

var _ MarshalInterface = &HubItemsInterface{}

func (m *HubItemsInterface) FromInterface(a any) error {
	dat, err := json.Marshal(a)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(dat, m); err != nil {
		return err
	}
	return nil
}

func (m *HubItemsInterface) ToInterface() (any, error) {
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

func (m *HubItemsInterface) Contains(h2 *HubItemsInterface) error {
	if len(*m) != len(*h2) {
		return fmt.Errorf("lengths do not match")
	}
	for i, hi := range *m {
		if err := hi.Contains((*h2)[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *HubItemsInterface) ApplyMiddlewares(validators Middlewares, out bool, vals ...HubItem) ([]HubItem, error) {
	if m == nil {
		return nil, fmt.Errorf("require interface")
	}
	if len(vals) != len(*m) {
		return nil, fmt.Errorf("lengths do not match")
	}
	vvals := []HubItem{}
	for i, hi := range *m {
		if v, err := hi.ApplyMiddlewares(validators, vals[i], out); err != nil {
			return nil, err
		} else {
			vvals = append(vvals, v)
		}
	}
	return vvals, nil
}

func (m *HubItemsInterface) Equals(m2 *HubItemsInterface) error {
	if m == nil {
		if m2 == nil {
			return nil
		}
		return fmt.Errorf("nil interface does not match %s", m2.String())
	}
	if m2 == nil {
		return fmt.Errorf("%s interface does not nil interface", m.String())
	}
	if len(*m) != len(*m2) {
		return fmt.Errorf("%d-interface does not match %d-interface", len(*m), len(*m2))
	}
	for i, h := range *m {
		h2 := (*m2)[i]
		if err := h.Contains(h2); err != nil {
			return fmt.Errorf("hub %d does not match: %s", i, err.Error())
		}
	}
	return nil
}

func (m *HubItemsInterface) String() string {
	if m == nil {
		return ""
	}
	hubStrs := []string{}
	for _, h := range *m {
		hubStrs = append(hubStrs, h.String())
	}
	return "(" + strings.Join(hubStrs, ",") + ")"
}
