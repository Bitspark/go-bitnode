package bitnode

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
)

type MarshalInterface interface {
	FromInterface(any) error
	ToInterface() (any, error)
}

type Implementation interface {
	MarshalInterface
	Implement(node *NativeNode, sys System) error
	Validate() error
}

// A Factory allows adding custom implementations to a system.
type Factory interface {
	Name() string

	Implementation(implData Implementation) (Implementation, error)
}

type Implementations map[string][]Implementation

var yamlFactories map[string]Factory

func (m *Implementations) FromInterface(i any) error {
	dat, err := yaml.Marshal(i)
	if err != nil {
		return err
	}
	impls := map[string][]any{}
	if err := yaml.Unmarshal(dat, &impls); err != nil {
		return err
	}
	for fName, implDatas := range impls {
		if yamlFactories == nil {
			return fmt.Errorf("factories not set")
		}
		f, ok := yamlFactories[fName]
		if !ok || f == nil {
			log.Printf("Factory not found: %s", fName)
			continue
		}
		for _, implData := range implDatas {
			impl, err := f.Implementation(nil)
			if err != nil {
				return err
			}
			if err := impl.FromInterface(implData); err != nil {
				return err
			}
			iimpls, _ := (*m)[fName]
			iimpls = append(iimpls, impl)
			(*m)[fName] = iimpls
		}
	}
	return nil
}

func (m *Implementations) UnmarshalYAML(value *yaml.Node) error {
	impls := map[string][]any{}
	err := value.Decode(&impls)
	if err != nil {
		return err
	}
	for fName, implDatas := range impls {
		if yamlFactories == nil {
			return fmt.Errorf("factories not set")
		}
		f, ok := yamlFactories[fName]
		if !ok || f == nil {
			log.Printf("Factory not found: %s", fName)
			continue
		}
		for _, implData := range implDatas {
			impl, err := f.Implementation(nil)
			if err != nil {
				return err
			}
			if err := impl.FromInterface(implData); err != nil {
				return err
			}
			iimpls, _ := (*m)[fName]
			iimpls = append(iimpls, impl)
			(*m)[fName] = iimpls
		}
	}
	return nil
}

func (m *Implementations) UnmarshalJSON(bts []byte) error {
	if m == nil {
		return nil
	}
	if len(*m) == 0 {
		*m = Implementations{}
	}
	impls := map[string][]any{}
	err := json.Unmarshal(bts, &impls)
	if err != nil {
		return err
	}
	for fName, implDatas := range impls {
		if yamlFactories == nil {
			return fmt.Errorf("factories not set")
		}
		f, ok := yamlFactories[fName]
		if !ok || f == nil {
			log.Printf("Factory not found: %s", fName)
			continue
		}
		for _, implData := range implDatas {
			impl, err := f.Implementation(nil)
			if err != nil {
				return err
			}
			if err := impl.FromInterface(implData); err != nil {
				return err
			}
			iimpls, _ := (*m)[fName]
			iimpls = append(iimpls, impl)
			(*m)[fName] = iimpls
		}
	}
	return nil
}
