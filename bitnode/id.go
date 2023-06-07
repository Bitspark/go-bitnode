package bitnode

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
)

const SystemIDSize = 6
const ObjectIDSize = 8

// FULL ID

// ID is composed of a SystemID and an ObjectID.
type ID [SystemIDSize + ObjectIDSize]byte
type IDList []ID
type IDPair [2]ID
type IDPairList []IDPair

// Decompose splits the ID into its SystemID and ObjectID.
func (id ID) Decompose() (SystemID, ObjectID) {
	var sid SystemID
	copy(sid[:], id[:SystemIDSize])
	var oid ObjectID
	copy(oid[:], id[SystemIDSize:])
	return sid, oid
}

// System returns the system ID where the object is stored.
func (id ID) System() SystemID {
	var sid SystemID
	copy(sid[:], id[:SystemIDSize])
	return sid
}

// Object returns the ID of the object on the system.
func (id ID) Object() ObjectID {
	var oid ObjectID
	copy(oid[:], id[SystemIDSize:])
	return oid
}

// Hex representation of the ID.
func (id ID) Hex() string {
	return hex.EncodeToString(id[:])
}

// IsNull returns true if the ID is made of zeros.
func (id ID) IsNull() bool {
	return id == ID{}
}

func (id ID) MarshalYAML() (interface{}, error) {
	return hex.EncodeToString(id[:]), nil
}

func (id *ID) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err == nil {
		*id = ParseID(str)
		return nil
	} else {
		return err
	}
}

func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(id[:]))
}

func (id *ID) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	} else {
		*id = ParseID(str)
		return nil
	}
}

func (ids IDList) MarshalYAML() (interface{}, error) {
	str := []string{}
	if ids != nil {
		for _, i := range ids {
			str = append(str, i.Hex())
		}
	}
	return str, nil
}

func (ids *IDList) UnmarshalYAML(value *yaml.Node) error {
	var str []string
	if err := value.Decode(&str); err == nil {
		*ids = nil
		for _, s := range str {
			id := ParseID(s)
			*ids = append(*ids, id)
		}
		return nil
	} else {
		return err
	}
}

func (ids IDList) MarshalJSON() ([]byte, error) {
	str := []string{}
	if ids != nil {
		for _, i := range ids {
			str = append(str, i.Hex())
		}
	}
	return json.Marshal(str)
}

func (ids *IDList) UnmarshalJSON(data []byte) error {
	var str []string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	} else {
		*ids = nil
		for _, s := range str {
			id := ParseID(s)
			*ids = append(*ids, id)
		}
		return nil
	}
}

func ComposeIDs(sysID SystemID, objID ObjectID) ID {
	id := ID{}
	copy(id[:], sysID[:])
	copy(id[SystemIDSize:], objID[:])
	return id
}

// ParseID returns an ID for the given string, which must be in hexadecimal format.
func ParseID(s string) ID {
	var id ID
	bts, _ := hex.DecodeString(s)
	copy(id[:], bts)
	return id
}

// SYSTEM ID

type SystemID [SystemIDSize]byte

// Hex representation of the SystemID.
func (id SystemID) Hex() string {
	return hex.EncodeToString(id[:])
}

// IsNull returns true if the SystemID is made of zeros.
func (id SystemID) IsNull() bool {
	return id == SystemID{}
}

func (id SystemID) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(id[:]))
}

func (id *SystemID) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	} else {
		*id = ParseSystemID(str)
		return nil
	}
}

func (id SystemID) MarshalYAML() (interface{}, error) {
	return hex.EncodeToString(id[:]), nil
}

func (id *SystemID) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err == nil {
		*id = ParseSystemID(str)
		return nil
	} else {
		return err
	}
}

func GenerateSystemID() SystemID {
	var id SystemID
	rand.Read(id[:])
	return id
}

// ParseSystemID returns an SystemID for the given string, which must be in hexadecimal format.
func ParseSystemID(s string) SystemID {
	var id SystemID
	bts, _ := hex.DecodeString(s)
	copy(id[:], bts)
	return id
}

// OBJECT ID

type ObjectID [ObjectIDSize]byte

// Hex representation of the ObjectID.
func (id ObjectID) Hex() string {
	return hex.EncodeToString(id[:])
}

// IsNull returns true if the ObjectID is made of zeros.
func (id ObjectID) IsNull() bool {
	return id == ObjectID{}
}

func (id ObjectID) MarshalJSON() ([]byte, error) {
	return json.Marshal(hex.EncodeToString(id[:]))
}

func (id *ObjectID) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	} else {
		*id = ParseObjectID(str)
		return nil
	}
}

func (id ObjectID) MarshalYAML() (interface{}, error) {
	return hex.EncodeToString(id[:]), nil
}

func (id *ObjectID) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err == nil {
		*id = ParseObjectID(str)
		return nil
	} else {
		return err
	}
}

func GenerateObjectID() ObjectID {
	var id ObjectID
	rand.Read(id[:])
	return id
}

// ParseObjectID returns an ObjectID for the given string, which must be in hexadecimal format.
func ParseObjectID(s string) ObjectID {
	var id ObjectID
	bts, _ := hex.DecodeString(s)
	copy(id[:], bts)
	return id
}

// DATA STRUCTURES

type ObjectIDs map[ObjectID]bool

// SYSTEM

type idWrapper struct {
	h *NativeNode
}

var _ Middleware = &idWrapper{}

func (s idWrapper) Name() string {
	return "id"
}

func (s idWrapper) Middleware(ext any, val HubItem, out bool) (HubItem, error) {
	extC := ext.(map[string]any)
	if out {
		tp, _ := extC["type"]
		if tp == nil {
			if i, ok := val.(ID); ok {
				return i.Hex(), nil
			}
		} else {
			tps := tp.(string)
			if tps == "object" {
				if i, ok := val.(ObjectID); ok {
					return i.Hex(), nil
				}
			} else if tps == "system" {
				if i, ok := val.(SystemID); ok {
					return i.Hex(), nil
				}
			}
		}
		return nil, fmt.Errorf("not an ID: %v", val)
	} else {
		if i, ok := val.(string); ok {
			tp, _ := extC["type"]
			if tp == nil {
				return ParseID(i), nil
			} else {
				tps := tp.(string)
				if tps == "object" {
					return ParseObjectID(i), nil
				} else if tps == "system" {
					return ParseSystemID(i), nil
				}
			}
		}
		return nil, fmt.Errorf("not an ID: %v", val)
	}
}
