package bitnode

import "testing"

func TestInterface_Contains1(t *testing.T) {
	ih1 := &HubInterfaces{
		{
			Name: "a",
		},
	}
	ih2 := &HubInterfaces{
		{
			Name: "a",
		},
	}
	ih3 := &HubInterfaces{
		{
			Name: "b",
		},
	}
	if err := ih1.Contains(ih2); err != nil {
		t.Fatal(err)
	}
	if err := ih1.Contains(ih3); err == nil {
		t.Fatal()
	}
}

func TestInterfaceMap_Contains1(t *testing.T) {
	ih1 := &HubItemInterface{
		Value: &Type{RawType: RawType{
			MapOf: map[string]*RawType{
				"a": {
					Leaf: LeafString,
				},
				"c": {
					Leaf: LeafString,
				},
			},
		}},
	}
	ih2 := &HubItemInterface{
		Value: &Type{RawType: RawType{
			MapOf: map[string]*RawType{
				"a": {
					Leaf: LeafString,
				},
			},
		}},
	}
	ih3 := &HubItemInterface{
		Value: &Type{RawType: RawType{
			MapOf: map[string]*RawType{
				"b": {
					Leaf: LeafString,
				},
			},
		}},
	}
	ih1.Compile(nil, "", false)
	ih2.Compile(nil, "", false)
	ih3.Compile(nil, "", false)
	if err := ih1.Contains(ih2); err != nil {
		t.Fatal(err)
	}
	if err := ih1.Contains(ih3); err == nil {
		t.Fatal()
	}
}

func TestInterfaceList_Contains1(t *testing.T) {
	ih1 := &HubItemInterface{
		Value: &Type{RawType: RawType{
			ListOf: &RawType{
				Leaf: LeafString,
			},
		}},
	}
	ih2 := &HubItemInterface{
		Value: &Type{RawType: RawType{
			ListOf: &RawType{
				Leaf: LeafString,
			},
		}},
	}
	ih3 := &HubItemInterface{
		Value: &Type{RawType: RawType{
			ListOf: &RawType{
				Leaf: LeafBoolean,
			},
		}},
	}
	ih1.Compile(nil, "", false)
	ih2.Compile(nil, "", false)
	ih3.Compile(nil, "", false)
	if err := ih1.Contains(ih2); err != nil {
		t.Fatal(err)
	}
	if err := ih1.Contains(ih3); err == nil {
		t.Fatal()
	}
}
