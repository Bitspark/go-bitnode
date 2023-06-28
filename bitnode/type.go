package bitnode

import (
	"encoding/json"
	"fmt"
	"github.com/Bitspark/go-bitnode/util"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"reflect"
)

// Get type

type Type struct {
	RawType `json:"base"`

	Compiled *RawType `json:"compiled,omitempty" yaml:"-"`

	// FullName of this type, can be used to reference it from elsewhere.
	FullName string `json:"fullName,omitempty" yaml:"-"`

	compiling bool

	references map[Compilable]bool
}

var _ Compilable = &Type{}
var _ Savable = &Type{}

func (t *Type) Reset() {
	t.Compiled = nil

	// Reset references.
	for rt := range t.references {
		rt.Reset()
	}
}

func (t *Type) Compile(dom *Domain, domName string, resolve bool) error {
	t.Domain = domName
	t.FullName = t.Name
	if t.Domain != "" {
		t.FullName = t.Domain + "." + t.FullName
	}
	if t.references == nil {
		t.references = map[Compilable]bool{}
	}

	var err error
	t.Compiled, err = t.RawType.Compile(dom, domName, resolve, t)

	// Compile references.
	for rt := range t.references {
		if err := rt.Compile(dom, rt.FullDomain(), true); err != nil {
			return err
		}
	}

	return err
}

func (t *Type) FullDomain() string {
	return t.Domain
}

type RawType struct {
	// Name of this type, can be used to reference it from elsewhere.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// Description of this type.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Leaf definition (e.g, string)
	Leaf LeafType `json:"leaf,omitempty" yaml:"leaf,omitempty"`

	// Reference to another type.
	Reference string `json:"reference,omitempty" yaml:"reference,omitempty"`

	// Permissions of this interface.
	Permissions *Permissions `json:"permissions,omitempty" yaml:"permissions,omitempty"`

	// Extends contains types this type extends.
	Extends []string `json:"extends" yaml:"extends,omitempty"`

	// ListOf creates a list type from that type.
	ListOf *RawType `json:"listOf,omitempty" yaml:"listOf,omitempty"`

	// TupleOf creates a tuple type from these types.
	TupleOf []*RawType `json:"tupleOf,omitempty" yaml:"tupleOf,omitempty"`

	// MapOf creates a map type from these types.
	MapOf map[string]*RawType `json:"mapOf" yaml:"mapOf,omitempty"`

	// Optional is true when the value can be nil.
	Optional bool `json:"optional,omitempty" yaml:"optional,omitempty"`

	//// IDOf is an ID of a model object.
	//IDOf *IDType `json:"idOf,omitempty" yaml:"idOf,omitempty"`

	//// Generic represents a slot in the value to be replaced by another value
	//Generic string `json:"generic,omitempty" yaml:"generic,omitempty"`
	//
	//// GenericMap contains types for generic slots
	//GenericMap GenericTypeMap `json:"genericMap,omitempty" yaml:"genericMap,omitempty"`

	// Options contains valid options for this type. They each must have this type. Has no impact if not specified.
	Options []any `json:"options,omitempty" yaml:"options,omitempty"`

	// Extensions can contain additional constraints about the type, particularly in combination with Reference.
	Extensions map[string]any `json:"extensions,omitempty" yaml:"extensions,omitempty"`

	// Domain this type resides in.
	Domain string `json:"domain,omitempty" yaml:"-"`
}

type TypeExtension any

func (t *RawType) Save(dom *Domain) error {
	tp, err := dom.GetDomain(t.Domain)
	if err != nil {
		return err
	}
	chDefsBytes, err := os.ReadFile(tp.FilePath)
	if err != nil {
		return fmt.Errorf("reading definitions from %s: %v", tp.FilePath, err)
	}

	defs := Domain{}
	if err := yaml.Unmarshal(chDefsBytes, &defs); err != nil {
		return fmt.Errorf("parsing definitions from %s: %v", tp.FilePath, err)
	}

	for _, tp2 := range defs.Types {
		if tp2.Name == t.Name {
			tp2.RawType = *t
		}
	}

	if yamlBts, err := yaml.Marshal(defs); err != nil {
		return fmt.Errorf("parsing definitions from %s: %v", tp.FilePath, err)
	} else {
		if err := os.WriteFile(tp.FilePath, yamlBts, os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

func (t *RawType) Compile(dom *Domain, domName string, resolve bool, rootType *Type) (*RawType, error) {
	compiled := t.Copy()

	// Recursive compile

	if t.MapOf != nil {
		compiled.MapOf = map[string]*RawType{}
		for k, ct := range t.MapOf {
			if k[0] < 'a' || k[0] > 'z' {
				return nil, fmt.Errorf("map keys names must start with a lower case character (a-z)")
			}
			if !util.IsAlphanumeric(k) {
				return nil, fmt.Errorf("map keys must not contain special characters")
			}
			if ctt, err := ct.Compile(dom, domName, resolve, rootType); err != nil {
				return nil, err
			} else {
				ctt.Name = k
				compiled.MapOf[k] = ctt
			}
		}
	}
	if t.ListOf != nil {
		if cl, err := t.ListOf.Compile(dom, domName, resolve, rootType); err != nil {
			return nil, err
		} else {
			compiled.ListOf = cl
		}
	}
	if t.TupleOf != nil {
		compiled.TupleOf = make([]*RawType, len(t.TupleOf))
		for i, ct := range t.TupleOf {
			if ctt, err := ct.Compile(dom, domName, resolve, rootType); err != nil {
				return nil, err
			} else {
				compiled.TupleOf[i] = ctt
			}
		}
	}

	compiled.Domain = domName

	if resolve && compiled.Reference != "" {
		dom, _ := dom.GetDomain(domName)
		if dom == nil {
			return nil, fmt.Errorf("domain not set")
		}
		rt, err := dom.GetType(compiled.Reference)
		if err != nil {
			log.Printf("type %s: %v", t.Reference, err)
		}
		if rt == nil {
			return nil, fmt.Errorf("type not found: %s", t.Reference)
		}
		if rt.Compiled == nil {
			if err := rt.Compile(dom, rt.Domain, resolve); err != nil {
				return nil, err
			}
		}
		tcpy := *t
		rt.references[rootType] = true
		optional := compiled.Optional
		*compiled = *rt.Compiled
		compiled.Optional = optional
		if tcpy.Extensions != nil {
			// For now, we override extensions
			compiled.Extensions = tcpy.Extensions
		}
		if tcpy.Options != nil {
			// TODO: Check if tcpy.Options are contained in t.Options
			compiled.Options = tcpy.Options
		}
		// TODO: Handle generics
	}

	return compiled, nil
}

func (t *RawType) String() string {
	if t == nil {
		return ""
	}
	jsonBts, _ := json.Marshal(t)
	return string(jsonBts)
}

func (t *RawType) Contains(t2 *RawType) error {
	if t.Leaf != t2.Leaf {
		return fmt.Errorf("incompatible leaf types")
	} else if t.Leaf != 0 {
		return nil
	}
	if t.ListOf != nil {
		if t2.ListOf == nil {
			return fmt.Errorf("incompatible list types")
		}
		return t.ListOf.Contains(t2.ListOf)
	}
	if t.MapOf != nil {
		if t2.MapOf == nil {
			return fmt.Errorf("incompatible map types")
		}
		for k2, v2 := range t2.MapOf {
			if v, ok := t.MapOf[k2]; !ok {
				return fmt.Errorf("missing map entry: %s", k2)
			} else if err := v.Contains(v2); err != nil {
				return err
			}
		}
		return nil
	}
	ts := t.String()
	t2s := t2.String()
	if ts == t2s {
		return nil
	}
	return fmt.Errorf("types do not match")
}

func (t *Type) Accepts(src *Type) (bool, error) {
	return t.Compiled.accepts(src.Compiled, "")
}

func (t *Type) Extend(base *Type) error {
	if base == nil {
		return nil
	}
	if base.MapOf != nil {
		if t.Compiled.MapOf == nil {
			return fmt.Errorf("base is a map, require map")
		}
		// TODO
		return nil
	} else if base.ListOf != nil {
		if t.Compiled.ListOf == nil {
			return fmt.Errorf("base is a list, require list")
		}

	} else if base.TupleOf != nil {
		if t.Compiled.TupleOf == nil {
			return fmt.Errorf("base is a tuple, require tuple")
		}

	}
	return fmt.Errorf("cannot extend this type")
}

func (t *RawType) ApplyMiddlewares(mws Middlewares, val HubItem, out bool) (any, error) {
	validated := false
	for f, ext := range t.Extensions {
		for i := 0; i < len(mws); i++ {
			var vs Middleware
			if out {
				vs = mws[i]
			} else {
				vs = mws[len(mws)-i-1]
			}
			if vs.Name() == f {
				var err error
				val, err = vs.Middleware(ext, val, out)
				if err != nil {
					return nil, err
				}
				validated = true
			}
		}
	}
	if validated {
		return val, nil
	}

	if t.Optional && val == nil {
		return nil, nil
	}

	if t.Leaf != 0 {
		switch t.Leaf {
		case LeafString:
			if val, ok := val.(string); ok {
				return val, nil
			}
			return nil, fmt.Errorf("not a string: %v", val)
		case LeafInteger:
			if val, ok := val.(int64); ok {
				return val, nil
			}
			if val, ok := val.(uint64); ok {
				return int64(val), nil
			}
			if val, ok := val.(int32); ok {
				return int64(val), nil
			}
			if val, ok := val.(uint32); ok {
				return int64(val), nil
			}
			if val, ok := val.(int16); ok {
				return int64(val), nil
			}
			if val, ok := val.(uint16); ok {
				return int64(val), nil
			}
			if val, ok := val.(int); ok {
				return int64(val), nil
			}
			if val, ok := val.(uint); ok {
				return int64(val), nil
			}
			if val, ok := val.(float64); ok {
				return int64(val), nil
			}
			return nil, fmt.Errorf("not an integer: %v", val)
		case LeafFloat:
			if val, ok := val.(int64); ok {
				return float64(val), nil
			}
			if val, ok := val.(uint64); ok {
				return float64(val), nil
			}
			if val, ok := val.(int32); ok {
				return float64(val), nil
			}
			if val, ok := val.(uint32); ok {
				return float64(val), nil
			}
			if val, ok := val.(int16); ok {
				return float64(val), nil
			}
			if val, ok := val.(uint16); ok {
				return float64(val), nil
			}
			if val, ok := val.(int); ok {
				return float64(val), nil
			}
			if val, ok := val.(uint); ok {
				return float64(val), nil
			}
			if val, ok := val.(float64); ok {
				return val, nil
			}
			return nil, fmt.Errorf("not a float: %v", val)
		case LeafBoolean:
			if val, ok := val.(bool); ok {
				return val, nil
			}
			return nil, fmt.Errorf("not a boolean: %v", val)
		case LeafRaw:
			if val, ok := val.([]byte); ok {
				return val, nil
			}
			return nil, fmt.Errorf("not raw bytes: %v", val)
		case LeafAny:
			return val, nil
		default:
			return nil, fmt.Errorf("invalid leaf type: %d", t.Leaf)
		}
	}

	if t.ListOf != nil {
		vals := []HubItem{}
		if val != nil {
			if reflect.TypeOf(val).Kind() == reflect.Slice {
				s := reflect.ValueOf(val)
				for i := 0; i < s.Len(); i++ {
					val2, err := t.ListOf.ApplyMiddlewares(mws, s.Index(i).Interface(), out)
					if err != nil {
						return nil, err
					}
					vals = append(vals, val2)
				}
			} else {
				return nil, fmt.Errorf("not a valid slice")
			}
		}
		return vals, nil
	}

	if len(t.MapOf) > 0 {
		vals := map[string]HubItem{}
		if mval, ok := val.(map[string]HubItem); ok {
			for k, kt := range t.MapOf {
				if kv, ok := mval[k]; !ok {
					if !kt.Optional {
						return nil, fmt.Errorf("missing map entry: %s", k)
					}
				} else {
					kv, err := kt.ApplyMiddlewares(mws, kv, out)
					if err != nil {
						return nil, err
					}
					vals[k] = kv
				}
			}
		} else if mval, ok := val.(map[string]any); ok {
			for k, kt := range t.MapOf {
				if kv, ok := mval[k]; !ok {
					if !kt.Optional {
						return nil, fmt.Errorf("missing map entry: %s", k)
					}
				} else {
					kv, err := kt.ApplyMiddlewares(mws, kv, out)
					if err != nil {
						return nil, err
					}
					vals[k] = kv
				}
			}
		}
		return vals, nil
	}

	//if t.IDOf != nil {
	//	if val, ok := val.(string); !ok {
	//		return nil, fmt.Errorf("not an ID")
	//	} else {
	//		if t.IDOf.Full {
	//			id := ParseID(val)
	//			return id, nil
	//		} else {
	//			id := ParseObjectID(val)
	//			return id, nil
	//		}
	//	}
	//}

	return val, nil
}

func (t *Type) MarshalYAML() (interface{}, error) {
	rt := t.RawType
	if rt.Reference != "" {
		rt.ListOf = nil
		rt.TupleOf = nil
		rt.MapOf = nil
		rt.Leaf = 0
		rt.Extensions = nil
	}
	if rt.Name != "" || rt.Description != "" || len(rt.Options) != 0 {
		return rt, nil
	}
	if rt.Leaf != 0 {
		return rt.Leaf, nil
	} else if rt.Reference != "" {
		return "$" + rt.Reference, nil
	} else if rt.MapOf != nil {
		return rt.MapOf, nil
	} else if rt.ListOf != nil {
		return []*RawType{rt.ListOf}, nil
	} else if rt.TupleOf != nil {
		return rt.TupleOf, nil
	}
	//if len(t.Extensions) == 0 {
	//	return nil, fmt.Errorf("unknown type")
	//}
	return rt, nil
}

func (t *Type) UnmarshalYAML(value *yaml.Node) error {
	var leaf LeafType
	if err := value.Decode(&leaf); err == nil {
		*t = Type{
			RawType: RawType{
				Leaf: leaf,
			},
		}
		return nil
	}
	var str string
	if err := value.Decode(&str); err == nil {
		if len(str) >= 2 && str[0] == '$' {
			t.Reference = str[1:]
			return nil
		}
		if len(str) >= 3 && str[0] == '<' && str[len(str)-1] == '>' {
			t.Reference = str[1 : len(str)-1]
			return nil
		}
		return fmt.Errorf("expected type reference or generic: %s", str)
	}
	var mp map[string]*RawType
	if err := value.Decode(&mp); mp["mapOf"] == nil && mp["listOf"] == nil && mp["leaf"] == nil && err == nil {
		t.MapOf = mp
		return nil
	}
	var lst []*RawType
	if err := value.Decode(&lst); err == nil {
		if len(lst) == 1 {
			t.ListOf = lst[0]
		} else {
			t.TupleOf = lst
		}
		return nil
	}
	return value.Decode(&t.RawType)
}

func (t *Type) FromInterface(a any) error {
	dat, err := json.Marshal(a)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(dat, t); err != nil {
		return err
	}
	return nil
}

func (t *Type) ToInterface() (any, error) {
	dat, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	datMp := map[string]any{}
	if err := json.Unmarshal(dat, &datMp); err != nil {
		return nil, err
	}
	return datMp, nil
}

func (t *Type) Copy() *Type {
	rt := t.RawType.Copy()
	return &Type{
		RawType:  *rt,
		Compiled: t.Compiled.Copy(),
	}
}

func (t *RawType) Copy() *RawType {
	t2 := &RawType{}
	*t2 = *t
	t2.Extensions = map[string]any{}
	for k, v := range t.Extensions {
		t2.Extensions[k] = v
	}
	if t.ListOf != nil {
		t2.ListOf = t.ListOf.Copy()
	} else if t.MapOf != nil {
		t2.MapOf = map[string]*RawType{}
		for k, v := range t.MapOf {
			t2.MapOf[k] = v.Copy()
		}
	} else if t.TupleOf != nil {
		t2.TupleOf = []*RawType{}
		for _, v := range t.TupleOf {
			t2.TupleOf = append(t2.TupleOf, v.Copy())
		}
	} /*else if t.IDOf != nil {
		t2.IDOf = &IDType{}
		*t2.IDOf = *t.IDOf
	}*/
	return t2
}

func (t *RawType) accepts(src *RawType, path string) (bool, error) {
	if !t.Optional && src.Optional {
		return false, fmt.Errorf("%s: cannot accept optional value", path)
	}

	return t.acceptsNonOptional(src, path)
}

func (t *RawType) acceptsNonOptional(src *RawType, path string) (bool, error) {
	// TODO: Consider options
	if src == nil {
		// Check for optional, once implemented
		return true, nil
	}

	if t.MapOf != nil {
		if src.MapOf == nil {
			return false, fmt.Errorf("%s: source should be a map", path)
		}
		for kSrc, vSrc := range src.MapOf {
			if vSelf, ok := t.MapOf[kSrc]; !ok {
				return false, fmt.Errorf("%s: source has additional map entry: %s", path, kSrc)
			} else {
				if ok, err := vSelf.accepts(vSrc, path+"."+kSrc); !ok {
					return false, err
				}
			}
		}
		return true, nil
	}

	if t.ListOf != nil {
		if src.ListOf == nil {
			return false, fmt.Errorf("%s: source should be a list", path)
		}
		return t.ListOf.accepts(src.ListOf, path+".%")
	}

	if t.TupleOf != nil {
		if src.TupleOf == nil {
			return false, fmt.Errorf("%s: source should be a tuple", path)
		}
		if len(t.TupleOf) != len(src.TupleOf) {
			return false, fmt.Errorf("%s: source tuple should have same length", path)
		}
		for i, tt := range t.TupleOf {
			if ok, err := tt.accepts(src.TupleOf[i], ""); err != nil || !ok {
				return false, err
			}
		}
		return true, nil
	}

	//if t.IDOf != nil {
	//	if src.IDOf == nil {
	//		return false, fmt.Errorf("%s: source should be an id", path)
	//	}
	//	if t.IDOf.Full != src.IDOf.Full {
	//		return false, fmt.Errorf("%s: source and target should both be full or half", path)
	//	}
	//	// TODO: Check if source model is a special of target model
	//	return true, nil
	//}

	if t.Leaf != src.Leaf {
		return false, fmt.Errorf("%s: differing leaves: %d != %d", path, t.Leaf, src.Leaf)
	} else {
		return true, nil
	}
}

func mustParseType(typeJSON string, dom *Domain) *Type {
	vt, err := parseType(typeJSON, dom)
	if err != nil {
		panic(err)
	}
	return vt
}

func parseType(typeJSON string, dom *Domain) (*Type, error) {
	vt := RawType{}
	if err := yaml.Unmarshal([]byte(typeJSON), &vt); err != nil {
		return nil, err
	}
	domName := ""
	if dom != nil {
		domName = dom.FullName
	}
	var err error
	vtt := &Type{RawType: vt}
	if err = vtt.Compile(dom, domName, true); err != nil {
		return nil, err
	}
	return vtt, nil
}

// Generic map

type GenericTypeMap map[string]*Type

// A Middleware validates and potentially transforms a HubItem into another HubItem.
type Middleware interface {
	Name() string
	Middleware(ext any, val HubItem, in bool) (HubItem, error)
}

type Middlewares []Middleware

func (mws *Middlewares) PushFront(f Middleware) {
	*mws = append([]Middleware{f}, *mws...)
}

func (mws *Middlewares) PushBack(f Middleware) {
	*mws = append(*mws, f)
}

func (mws *Middlewares) Copy() *Middlewares {
	mws2 := &Middlewares{}
	for _, mw := range *mws {
		*mws2 = append(*mws2, mw)
	}
	return mws2
}

// Leaf type

type LeafType int

const (
	LeafString = LeafType(iota + 1)
	LeafFloat
	LeafInteger
	LeafBoolean
	LeafRaw
	LeafAny
)

func (tb *LeafType) String() string {
	str := ""
	if tb == nil {
		return str
	}
	switch *tb {
	case LeafString:
		str = "string"
	case LeafFloat:
		str = "float"
	case LeafInteger:
		str = "integer"
	case LeafBoolean:
		str = "boolean"
	case LeafRaw:
		str = "raw"
	case LeafAny:
		str = "any"
	}
	return str
}

func (tb *LeafType) FromString(str string) error {
	switch str {
	case "string":
		*tb = LeafString
	case "float":
		*tb = LeafFloat
	case "integer":
		*tb = LeafInteger
	case "boolean":
		*tb = LeafBoolean
	case "raw":
		*tb = LeafRaw
	case "any":
		*tb = LeafAny
	default:
		return fmt.Errorf("unknown leaf type: %s", str)
	}
	return nil
}

func (tb LeafType) MarshalJSON() ([]byte, error) {
	return json.Marshal(tb.String())
}

func (tb *LeafType) UnmarshalJSON(data []byte) error {
	var i int
	err := json.Unmarshal(data, &i)
	if err == nil {
		*tb = LeafType(i)
		return nil
	}
	var str string
	err = json.Unmarshal(data, &str)
	if err != nil {
		return fmt.Errorf("leaf type must be string, got %s", string(data))
	}
	return tb.FromString(str)
}

func (tb LeafType) MarshalYAML() (interface{}, error) {
	return tb.String(), nil
}

func (tb *LeafType) UnmarshalYAML(value *yaml.Node) error {
	var i int
	err := value.Decode(&i)
	if err == nil {
		*tb = LeafType(i)
		return nil
	}
	var str string
	err = value.Decode(&str)
	if err != nil {
		return err
	}
	return tb.FromString(str)
}
