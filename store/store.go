package store

import (
	"fmt"
	"github.com/Bitspark/go-bitnode/util"
	"gopkg.in/yaml.v3"
	"os"
	"path"
	"sync"
)

const (
	DSKeyValue = "keyvalue"
	DSStores   = "stores"
)

type DataStructure struct {
	Name      string
	Type      string
	Structure Structure
}

func (ds *DataStructure) KeyValue() KeyValue {
	if ds.Type != DSKeyValue {
		panic("not a key value structure")
	}
	return ds.Structure.(KeyValue)
}

func (ds *DataStructure) Stores() Stores {
	if ds.Type != DSStores {
		panic("not a key value structure")
	}
	return ds.Structure.(Stores)
}

func (ds *DataStructure) Write(dir string) error {
	return ds.Structure.Write(dir, ds.Name)
}

func (ds *DataStructure) Read(dir string) error {
	return ds.Structure.Read(dir, ds.Name)
}

// STORE

type Store interface {
	Name() string

	Create(name string, dsType string) (*DataStructure, error)
	Get(name string) (*DataStructure, error)
	Ensure(name string, dsType string) (*DataStructure, error)
	Delete(name string) error

	Write(path string) error
	Read(path string) error
}

type store struct {
	name           string
	dataStructures map[string]*DataStructure
	mux            *sync.Mutex
}

var _ Store = &store{}

func NewStore(name string) Store {
	return &store{
		name:           name,
		dataStructures: map[string]*DataStructure{},
		mux:            &sync.Mutex{},
	}
}

func (s *store) Name() string {
	return s.name
}

func (s *store) Create(name string, dsType string) (*DataStructure, error) {
	if !util.CheckString(util.CharsAlphaLowerNum, name, false) {
		return nil, fmt.Errorf("data structure name must only contain alphanumeric characters")
	}

	if _, ok := s.dataStructures[name]; ok {
		return nil, fmt.Errorf("data structure %s already exists", name)
	}
	ds := &DataStructure{
		Name: name,
		Type: dsType,
	}
	switch dsType {
	case DSKeyValue:
		ds.Structure = newKeyValue()
	case DSStores:
		ds.Structure = newStores()
	}
	s.mux.Lock()
	s.dataStructures[name] = ds
	s.mux.Unlock()
	return ds, nil
}

func (s *store) Get(name string) (*DataStructure, error) {
	s.mux.Lock()
	defer s.mux.Unlock()
	if ds, ok := s.dataStructures[name]; !ok {
		return nil, fmt.Errorf("data structure %s does not exist", name)
	} else {
		return ds, nil
	}
}

func (s *store) Ensure(name string, dsType string) (*DataStructure, error) {
	ds, err := s.Get(name)
	if err == nil {
		if ds.Type != dsType {
			return nil, fmt.Errorf("data structure has type %s instead of %s", ds.Type, dsType)
		}
		return ds, nil
	}
	return s.Create(name, dsType)
}

func (s *store) Delete(name string) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	if _, ok := s.dataStructures[name]; !ok {
		return fmt.Errorf("data structure %s does not exist", name)
	} else {
		delete(s.dataStructures, name)
		return nil
	}
}

// storeStruct used for writing and reading
type storeStruct struct {
	DataStructures map[string]string `yaml:"dataStructures"`
}

func (s *store) Write(dir string) error {
	dir = path.Join(dir, s.name)

	_ = os.MkdirAll(dir, os.ModePerm)

	ss := storeStruct{
		DataStructures: map[string]string{},
	}

	for _, v := range s.dataStructures {
		ss.DataStructures[v.Name] = v.Type
	}

	stBts, err := yaml.Marshal(ss)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path.Join(dir, "_store.yml"), stBts, os.ModePerm); err != nil {
		return err
	}

	for _, v := range s.dataStructures {
		if err := v.Write(dir); err != nil {
			return err
		}
	}

	return nil
}

func (s *store) Read(dir string) error {
	dir = path.Join(dir, s.name)

	ss := storeStruct{}

	stBts, err := os.ReadFile(path.Join(dir, "_store.yml"))
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(stBts, &ss); err != nil {
		return err
	}

	for k, v := range ss.DataStructures {
		if ds, err := s.Create(k, v); err != nil {
			return err
		} else if err := ds.Read(dir); err != nil {
			return err
		}
	}

	return nil
}

// STRUCTURE INTERFACE

type Structure interface {
	Write(dir string, name string) error
	Read(dir string, name string) error
}

// KEY VALUE

type KeyValue interface {
	Structure
	Get(key string) (string, error)
	Set(key string, value string) error
	EnumerateKeys() chan string
}

type keyValue struct {
	Entries map[string]string `yaml:"entries"`

	mux *sync.Mutex
}

var _ KeyValue = &keyValue{}

func newKeyValue() KeyValue {
	return &keyValue{
		Entries: map[string]string{},

		mux: &sync.Mutex{},
	}
}

func (k *keyValue) Get(key string) (string, error) {
	k.mux.Lock()
	defer k.mux.Unlock()
	if e, ok := k.Entries[key]; !ok {
		return "", fmt.Errorf("entry does not exist")
	} else {
		return e, nil
	}
}

func (k *keyValue) Set(key string, value string) error {
	k.mux.Lock()
	defer k.mux.Unlock()
	k.Entries[key] = value
	return nil
}

func (k *keyValue) EnumerateKeys() chan string {
	c := make(chan string)
	go func() {
		for k := range k.Entries {
			c <- k
		}
		close(c)
	}()
	return c
}

func (k *keyValue) Write(dir string, name string) error {
	stBts, _ := yaml.Marshal(k)
	_ = os.WriteFile(path.Join(dir, name+".yml"), stBts, os.ModePerm)
	return nil
}

func (k *keyValue) Read(dir string, name string) error {
	stBts, _ := os.ReadFile(path.Join(dir, name+".yml"))
	_ = yaml.Unmarshal(stBts, k)
	return nil
}

// STORES

type Stores interface {
	Structure
	Get(name string) (Store, error)
	Add(value Store) error
	Enumerate() chan Store
}

type stores struct {
	Entries map[string]Store `yaml:"entries"`

	mux *sync.Mutex
}

var _ Stores = &stores{}

func newStores() Stores {
	return &stores{
		Entries: map[string]Store{},

		mux: &sync.Mutex{},
	}
}

func (k *stores) Get(name string) (Store, error) {
	k.mux.Lock()
	defer k.mux.Unlock()
	if e, ok := k.Entries[name]; !ok {
		return nil, fmt.Errorf("entry does not exist")
	} else {
		return e, nil
	}
}

func (k *stores) Add(value Store) error {
	k.mux.Lock()
	defer k.mux.Unlock()
	k.Entries[value.Name()] = value
	return nil
}

func (k *stores) Enumerate() chan Store {
	c := make(chan Store)
	go func() {
		for _, st := range k.Entries {
			c <- st
		}
		close(c)
	}()
	return c
}

func (k *stores) Write(dir string, name string) error {
	sts := []string{}

	for n, st := range k.Entries {
		sts = append(sts, n)
		if err := st.Write(path.Join(dir, name)); err != nil {
			return err
		}
	}

	stBts, _ := yaml.Marshal(sts)
	_ = os.WriteFile(path.Join(dir, name, "_stores.yml"), stBts, os.ModePerm)
	return nil
}

func (k *stores) Read(dir string, name string) error {
	sts := []string{}
	stBts, _ := os.ReadFile(path.Join(dir, name, "_stores.yml"))
	_ = yaml.Unmarshal(stBts, &sts)

	for _, n := range sts {
		st := NewStore(n)
		if err := st.Read(path.Join(dir, name)); err != nil {
			return err
		}
		_ = k.Add(st)
	}

	return nil
}
