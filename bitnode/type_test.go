package bitnode

import "testing"

func TestValueType_Accepts__Incompatible1(t *testing.T) {
	v1 := mustParseType(`{"listOf": {"leaf": "string"}}`, nil)
	v2 := mustParseType(`{"leaf": "string"}`, nil)
	if ok, err := v1.Accepts(v2); ok || err == nil {
		t.Fatal()
	}
	if ok, err := v2.Accepts(v1); ok || err == nil {
		t.Fatal()
	}
}

func TestValueType_Accepts__Incompatible2(t *testing.T) {
	v1 := mustParseType(`{"mapOf": {"a": {"leaf": "string"}}}`, nil)
	v2 := mustParseType(`{"leaf": "string"}`, nil)
	if ok, err := v1.Accepts(v2); ok || err == nil {
		t.Fatal()
	}
	if ok, err := v2.Accepts(v1); ok || err == nil {
		t.Fatal()
	}
}

func TestValueType_Accepts__Incompatible3(t *testing.T) {
	v1 := mustParseType(`{"mapOf": {"a": {"leaf": "string"}}}`, nil)
	v2 := mustParseType(`{"leaf": "string"}`, nil)
	if ok, err := v1.Accepts(v2); ok || err == nil {
		t.Fatal()
	}
	if ok, err := v2.Accepts(v1); ok || err == nil {
		t.Fatal()
	}
}

func TestValueType_Accepts__Leaf1(t *testing.T) {
	v1 := mustParseType(`{"leaf": "string"}`, nil)
	v2 := mustParseType(`{"leaf": "integer"}`, nil)
	if ok, err := v1.Accepts(v2); ok || err == nil {
		t.Fatal()
	}
	if ok, err := v2.Accepts(v1); ok || err == nil {
		t.Fatal()
	}
}

func TestValueType_Accepts__Leaf2(t *testing.T) {
	v1 := mustParseType(`{"leaf": "integer"}`, nil)
	v2 := mustParseType(`{"leaf": "integer"}`, nil)
	if ok, err := v1.Accepts(v2); !ok || err != nil {
		t.Fatal(err)
	}
	if ok, err := v2.Accepts(v1); !ok || err != nil {
		t.Fatal(err)
	}
}

func TestValueType_Accepts__Map1(t *testing.T) {
	v1 := mustParseType(`{"mapOf": {"a": {"leaf": "string"}}}`, nil)
	v2 := mustParseType(`{"mapOf": {"a": {"leaf": "string"}, "b": {"leaf": "string"}}}`, nil)
	if ok, err := v1.Accepts(v2); ok || err == nil {
		t.Fatal()
	}
	if ok, err := v2.Accepts(v1); !ok || err != nil {
		t.Fatal(err)
	}
}

func TestValueType_Accepts__Map2(t *testing.T) {
	v1 := mustParseType(`{"mapOf": {"a": {"leaf": "raw"}, "b": {"leaf": "boolean"}}}`, nil)
	v2 := mustParseType(`{"mapOf": {"a": {"leaf": "string"}, "b": {"leaf": "boolean"}}}`, nil)
	if ok, err := v1.Accepts(v2); ok || err == nil {
		t.Fatal()
	}
	if ok, err := v2.Accepts(v1); ok || err == nil {
		t.Fatal()
	}
}

func TestValueType_Accepts__List1(t *testing.T) {
	v1 := mustParseType(`{"listOf": {"leaf": "string"}}`, nil)
	v2 := mustParseType(`{"listOf": {"leaf": "integer"}}`, nil)
	if ok, err := v1.Accepts(v2); ok || err == nil {
		t.Fatal()
	}
	if ok, err := v2.Accepts(v1); ok || err == nil {
		t.Fatal()
	}
}

func TestValueType_Accepts__List2(t *testing.T) {
	v1 := mustParseType(`{"listOf": {"leaf": "float"}}`, nil)
	v2 := mustParseType(`{"listOf": {"leaf": "float"}}`, nil)
	if ok, err := v1.Accepts(v2); !ok || err != nil {
		t.Fatal(err)
	}
	if ok, err := v2.Accepts(v1); !ok || err != nil {
		t.Fatal(err)
	}
}

func TestValueType_Accepts__Optional1(t *testing.T) {
	v1 := mustParseType(`{"leaf": "boolean", "optional": true}`, nil)
	v2 := mustParseType(`{"leaf": "boolean"}`, nil)
	if ok, err := v1.Accepts(v2); !ok || err != nil {
		t.Fatal()
	}
	if ok, err := v2.Accepts(v1); ok || err == nil {
		t.Fatal()
	}
}

func TestValueType_Accepts__Optional2(t *testing.T) {
	v1 := mustParseType(`{"mapOf": {"a": {"leaf": "string", "optional": true}}}`, nil)
	v2 := mustParseType(`{"mapOf": {"a": {"leaf": "string"}}}`, nil)
	if ok, err := v1.Accepts(v2); !ok || err != nil {
		t.Fatal()
	}
	if ok, err := v2.Accepts(v1); ok || err == nil {
		t.Fatal()
	}
}

func TestValueType_Accepts__Optional3(t *testing.T) {
	v1 := mustParseType(`{"mapOf": {"a": {"leaf": "string"}}, "optional": true}`, nil)
	v2 := mustParseType(`{"mapOf": {"a": {"leaf": "string"}}}`, nil)
	if ok, err := v1.Accepts(v2); !ok || err != nil {
		t.Fatal()
	}
	if ok, err := v2.Accepts(v1); ok || err == nil {
		t.Fatal()
	}
}

func TestValueCompile__General(t *testing.T) {
	v := &Type{
		RawType: RawType{
			Name:        "a",
			Description: "b",
		},
	}

	err := v.Compile(nil, "test", false)
	if err != nil {
		t.Fatal(err)
	}

	if v.Compiled.Name != "a" {
		t.Fatal()
	}
	if v.Compiled.Description != "b" {
		t.Fatal()
	}
	if v.Domain != "test" {
		t.Fatal()
	}
	if v.FullName != "test.a" {
		t.Fatal()
	}
}

func TestValueCompile__Map(t *testing.T) {
	v := &Type{
		RawType: RawType{
			MapOf: map[string]*RawType{
				"a": {
					Leaf: LeafInteger,
				},
			},
		},
	}

	err := v.Compile(nil, "", false)
	if err != nil {
		t.Fatal(err)
	}

	if v.Compiled.MapOf["a"].Leaf != LeafInteger {
		t.Fatal()
	}
}

func TestValueCompile__List(t *testing.T) {
	v := &Type{
		RawType: RawType{
			ListOf: &RawType{
				Leaf: LeafBoolean,
			},
		},
	}

	err := v.Compile(nil, "", false)
	if err != nil {
		t.Fatal(err)
	}

	if v.Compiled.ListOf.Leaf != LeafBoolean {
		t.Fatal()
	}
}

func TestValueCompile__Tuple(t *testing.T) {
	v := &Type{
		RawType: RawType{
			TupleOf: []*RawType{
				{
					Leaf: LeafString,
				},
			},
		},
	}

	err := v.Compile(nil, "", false)
	if err != nil {
		t.Fatal(err)
	}

	if len(v.Compiled.TupleOf) != 1 {
		t.Fatal()
	}
	if v.Compiled.TupleOf[0].Leaf != LeafString {
		t.Fatal()
	}
}

func TestValueCompile__Extensions(t *testing.T) {
	v := &Type{
		RawType: RawType{
			Extensions: map[string]any{
				"testf": 1,
			},
		},
	}

	h := NewDomain()
	//h.Middlewares().PushBack(&testMW{})

	err := v.Compile(h, "", false)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := v.Compiled.Extensions["testf"]; !ok {
		t.Fatal()
	}
}

func TestValueCompile__ExtensionsList(t *testing.T) {
	v := &Type{
		RawType: RawType{
			ListOf: &RawType{
				Extensions: map[string]any{
					"testf": 1,
				},
			},
		},
	}

	h := NewDomain()
	//h.Middlewares().PushBack(&testMW{})

	err := v.Compile(h, "", false)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := v.Compiled.ListOf.Extensions["testf"]; !ok {
		t.Fatal()
	}
}

func TestValueApplyMiddlewares__Leaf1(t *testing.T) {
	v1 := mustParseType(`{"leaf": "string"}`, nil)
	v, err := v1.ApplyMiddlewares(nil, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	if v != "test" {
		t.Fatal(v)
	}
}

func TestValueApplyMiddlewares__Leaf2(t *testing.T) {
	v1 := mustParseType(`{"leaf": "integer"}`, nil)
	_, err := v1.ApplyMiddlewares(nil, "test", true)
	if err == nil {
		t.Fatal()
	}
	v, err := v1.ApplyMiddlewares(nil, 12, true)
	if err != nil {
		t.Fatal(err)
	}
	if v != int64(12) {
		t.Fatal(v)
	}
}

func TestValueApplyMiddlewares__Leaf3(t *testing.T) {
	v1 := mustParseType(`{"leaf": "boolean"}`, nil)
	_, err := v1.ApplyMiddlewares(nil, "test", true)
	if err == nil {
		t.Fatal()
	}
	v, err := v1.ApplyMiddlewares(nil, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if v != true {
		t.Fatal(v)
	}
}

func TestValueApplyMiddlewares__Leaf__Optional1(t *testing.T) {
	v1 := mustParseType(`{"leaf": "string", "optional": true}`, nil)
	v, err := v1.ApplyMiddlewares(nil, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	if v != "test" {
		t.Fatal(v)
	}
	v, err = v1.ApplyMiddlewares(nil, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if v != nil {
		t.Fatal(v)
	}
}

func TestValueApplyMiddlewares__ListOf1(t *testing.T) {
	v1 := mustParseType(`{"listOf": { "leaf": "boolean" }}`, nil)
	_, err := v1.ApplyMiddlewares(nil, []any{false, "test"}, true)
	if err == nil {
		t.Fatal()
	}
	v, err := v1.ApplyMiddlewares(nil, []any{true, false}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.([]HubItem)) != 2 {
		t.Fatal(v)
	}
}

func TestValueApplyMiddlewares__ListOf__Optional1(t *testing.T) {
	v1 := mustParseType(`{"listOf": { "leaf": "boolean", "optional": true }}`, nil)
	_, err := v1.ApplyMiddlewares(nil, []any{false, "test"}, true)
	if err == nil {
		t.Fatal()
	}
	v, err := v1.ApplyMiddlewares(nil, []any{true, false, nil}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.([]HubItem)) != 3 {
		t.Fatal(v)
	}
}

func TestValueApplyMiddlewares__MapOf1(t *testing.T) {
	v1 := mustParseType(`{"mapOf": { "a": { "leaf": "boolean" }}}`, nil)
	_, err := v1.ApplyMiddlewares(nil, map[string]any{"a": ""}, true)
	if err == nil {
		t.Fatal()
	}
	v, err := v1.ApplyMiddlewares(nil, map[string]any{"a": true}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.(map[string]HubItem)) != 1 {
		t.Fatal(v)
	}
}

type testMW struct {
}

var _ Middleware = &testMW{}

func (tmw testMW) Name() string {
	return "testf"
}

func (tmw testMW) Middleware(ext any, val HubItem, in bool) (HubItem, error) {
	return val, nil
}
