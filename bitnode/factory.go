package bitnode

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
)

// A FactoryImplementation contain all required data for the Factory to implement a system.
type FactoryImplementation interface {
	Implement(sys System) (FactorySystem, error)
}

// A Factory allows adding custom implementations to a system.
type Factory interface {
	// The System providing system-level access to this Factory.
	//System() System

	// Parse parses the provided interface into a FactoryImplementation.
	Parse(data any) (FactoryImplementation, error)

	// Serialize serializes a FactoryImplementation.
	Serialize(impl FactoryImplementation) (any, error)
}

type Implementations map[string][]FactoryImplementation

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
			impl, err := f.Parse(implData)
			if err != nil {
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
			impl, err := f.Parse(implData)
			if err != nil {
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
			impl, err := f.Parse(implData)
			if err != nil {
				return err
			}
			iimpls, _ := (*m)[fName]
			iimpls = append(iimpls, impl)
			(*m)[fName] = iimpls
		}
	}
	return nil
}
