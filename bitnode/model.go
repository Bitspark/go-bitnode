package bitnode

import (
	"encoding/json"
	"fmt"
	"github.com/Bitspark/go-bitnode/util"
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

// MODEL

// A Model defines an object-oriented structure of fields pointing to other models.
type Model struct {
	RawModel `json:"base"`

	Compiled *RawModel `json:"compiled,omitempty" yaml:"-"`

	// FullName of this type, can be used to reference it from elsewhere.
	FullName string `json:"fullName,omitempty" yaml:"-"`

	compiling bool
}

var _ Compilable = &Model{}
var _ Savable = &Model{}

func (m *Model) Reset() {
	m.Compiled = nil
}

func (m *Model) Compile(dom *Domain, domName string, resolve bool) error {
	m.Domain = domName
	m.FullName = m.Name
	if m.Domain != "" {
		m.FullName = m.Domain + "." + m.FullName
	}

	var err error
	m.Compiled, err = m.RawModel.Compile(dom, domName, resolve)

	return err
}

func (m *Model) FullDomain() string {
	return m.Domain
}

func (m *Model) FromInterface(a any) error {
	dat, err := json.Marshal(a)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(dat, m); err != nil {
		return err
	}
	return nil
}

func (m *Model) ToInterface() (any, error) {
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

func (m *Model) MarshalYAML() (interface{}, error) {
	return m.RawModel, nil
}

func (m *Model) UnmarshalYAML(value *yaml.Node) error {
	return value.Decode(&m.RawModel)
}

// A RawModel is a compiled or uncompiled Model.
type RawModel struct {
	// Name of the model.
	Name string `json:"name" yaml:"name"`

	// Description of the model.
	Description string `json:"description" yaml:"description"`

	// Permissions of this interface.
	Permissions *Permissions `json:"permissions,omitempty" yaml:"permissions,omitempty"`

	// Abstracts contains abstract models.
	Abstracts []string `json:"abstracts" yaml:"abstracts"`

	// Specials contains special models (i.e. extensions).
	Specials []string `json:"specials" yaml:"-"`

	// Fields are relations attached to this model, pointing to other models.
	Fields []*RawField `json:"fields" yaml:"fields"`

	// Domain this model resides in.
	Domain string `json:"domain,omitempty" yaml:"-"`
}

func (m *RawModel) Compile(dom *Domain, domName string, resolve bool) (*RawModel, error) {
	if m.Name != "" {
		if m.Name[0] < 'A' || m.Name[0] > 'Z' {
			return nil, fmt.Errorf("model names must start with an upper case character (A-Z)")
		}
		if !util.CheckString(util.CharsAlphaLowerNum, m.Name, false) {
			return nil, fmt.Errorf("model names must not special characters")
		}
	}

	compiled := m.Copy()

	compiled.Domain = domName
	compiled.Fields = []*RawField{}

	if resolve && len(m.Abstracts) > 0 {
		cdom, _ := dom.GetDomain(domName)
		if cdom == nil {
			return nil, fmt.Errorf("domain not set")
		}
		for _, ext := range m.Abstracts {
			ri, err := cdom.GetModel(ext)
			if err != nil {
				log.Printf("model %s: %v", ext, err)
			}
			if ri == nil {
				return nil, fmt.Errorf("model not found: %s", ext)
			}
			if ri.Compiled == nil {
				if err := ri.Compile(dom, ri.Domain, resolve); err != nil {
					return nil, err
				}
			}
			for _, rf := range ri.Compiled.Abstracts {
				compiled.Abstracts = append(compiled.Abstracts, rf)
			}
			for _, rf := range ri.Compiled.Fields {
				compiled.Fields = append(compiled.Fields, rf)
			}
			ri.Compiled.Specials = append(ri.Compiled.Specials, domName+"."+m.Name)
		}
	}

	for _, f := range m.Fields {
		f2, err := f.Compile(dom, domName, resolve)
		if err != nil {
			return nil, err
		}
		compiled.Fields = append(compiled.Fields, f2)
	}

	return compiled, nil
}

func (m *RawModel) Save(dom *Domain) error {
	tp, err := dom.GetDomain(m.Domain)
	if err != nil {
		return err
	}
	chDefsBytes, err := os.ReadFile(tp.filePath)
	if err != nil {
		return fmt.Errorf("reading definitions from %s: %v", tp.filePath, err)
	}

	defs := Domain{}
	if err := yaml.Unmarshal(chDefsBytes, &defs); err != nil {
		return fmt.Errorf("parsing definitions from %s: %v", tp.filePath, err)
	}

	for _, tp2 := range defs.Models {
		if tp2.Name == m.Name {
			tp2.RawModel = *m
		}
	}

	if yamlBts, err := yaml.Marshal(defs); err != nil {
		return fmt.Errorf("parsing definitions from %s: %v", tp.filePath, err)
	} else {
		if err := os.WriteFile(tp.filePath, yamlBts, os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

func (m *RawModel) Copy() *RawModel {
	t2 := &RawModel{}
	*t2 = *m
	t2.Fields = []*RawField{}
	for _, f := range m.Fields {
		t2.Fields = append(t2.Fields, f.Copy())
	}
	return t2
}

// FIELD

type RawField struct {
	// RawModel base of this field.
	RawModel `json:"model" yaml:"model"`

	// Origin model of the field.
	Origin string `json:"origin" yaml:"origin"`

	// Target model of the field.
	Target string `json:"target" yaml:"target"`
}

func (m *RawField) Compile(dom *Domain, domName string, resolve bool) (*RawField, error) {
	m2 := m.Copy()
	return m2, nil
}

func (m *RawField) Copy() *RawField {
	t2 := &RawField{}
	*t2 = *m
	return t2
}
