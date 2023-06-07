package bitnode

import (
	"testing"
	"time"
)

func TestPortPolicyAccept(t *testing.T) {
	vt := &Type{RawType: RawType{
		Leaf: LeafInteger,
	}}
	vt.Compile(nil, "", false)
	p := NewHub(nil, &HubInterface{
		Value: &HubItemInterface{
			Value: vt,
		},
		Type:      HubTypeChannel,
		Direction: HubDirectionIn,
	})
	creds := Credentials{}
	mws := Middlewares{}
	if err := p.Push(creds, mws, "", 1); err != nil {
		t.Fatal(err)
	}
	if err := p.Push(creds, mws, "", 2); err != nil {
		t.Fatal(err)
	}
	if err := p.Push(creds, mws, "", 3); err != nil {
		t.Fatal(err)
	}

	vals1 := []any{}

	p.Subscribe(creds, mws, NewNativeSubscription(func(id string, creds Credentials, val HubItem) {
		vals1 = append(vals1, val)
	}))
	if len(vals1) != 0 {
		t.Fatal()
	}

	p.Push(creds, mws, "", 4)
	p.Push(creds, mws, "", 5)

	if len(vals1) != 2 || vals1[0] != int64(4) || vals1[1] != int64(5) {
		t.Fatal()
	}
}

func TestPortPolicyValue(t *testing.T) {
	vt := &Type{RawType: RawType{
		Leaf: LeafInteger,
	}}
	vt.Compile(nil, "", false)
	p := NewHub(nil, &HubInterface{
		Value: &HubItemInterface{
			Value: vt,
		},
		Type:      HubTypeValue,
		Direction: HubDirectionBoth,
	})

	creds := Credentials{}
	mws := Middlewares{}

	if val, err := p.Get(creds, mws); err != nil || val != nil {
		t.Fatal()
	}

	if err := p.Set(creds, mws, "", 1); err != nil {
		t.Fatal(err)
	}
	if err := p.Set(creds, mws, "", 2); err != nil {
		t.Fatal(err)
	}
	if err := p.Set(creds, mws, "", int64(3)); err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Millisecond)

	vals1 := []any{}

	p.Subscribe(creds, mws, NewNativeSubscription(func(id string, creds Credentials, val HubItem) {
		vals1 = append(vals1, val)
	}))
	if len(vals1) != 1 {
		t.Fatal()
	}

	p.Set(creds, mws, "", 4)
	time.Sleep(5 * time.Millisecond)

	p.Set(creds, mws, "", 5)
	time.Sleep(5 * time.Millisecond)

	if len(vals1) != 3 || vals1[0] != int64(3) || vals1[1] != int64(4) || vals1[2] != int64(5) {
		t.Fatal(vals1)
	}

	if val, err := p.Get(creds, mws); err != nil || val != int64(5) {
		t.Fatal(val)
	}
}
