package store

import (
	"os"
	"testing"
)

func TestKeyValue1(t *testing.T) {
	kv := newKeyValue()

	if err := kv.Set("a", "1"); err != nil {
		t.Fatal(err)
	}

	v, err := kv.Get("a")
	if err != nil {
		t.Fatal(err)
	}
	if string(v) != "1" {
		t.Fatal(v)
	}
}

func TestKeyValue2(t *testing.T) {
	_ = os.MkdirAll("./test", os.ModePerm)

	kv1 := newKeyValue()

	if err := kv1.Set("a", "1"); err != nil {
		t.Fatal(err)
	}

	if err := kv1.Write("./test", "TestKeyValue1"); err != nil {
		t.Fatal(err)
	}

	kv2 := newKeyValue()

	if err := kv2.Read("./test", "TestKeyValue1"); err != nil {
		t.Fatal(err)
	}

	v, err := kv2.Get("a")
	if err != nil {
		t.Fatal(err)
	}
	if string(v) != "1" {
		t.Fatal(v)
	}
}

func TestStores1(t *testing.T) {
	kv := newStores()

	if err := kv.Add(NewStore("a")); err != nil {
		t.Fatal(err)
	}

	if err := kv.Add(NewStore("b")); err != nil {
		t.Fatal(err)
	}

	v1, err := kv.Get("a")
	if err != nil {
		t.Fatal(err)
	}
	if v1.Name() != "a" {
		t.Fatal(v1)
	}

	v2, err := kv.Get("b")
	if err != nil {
		t.Fatal(err)
	}
	if v2.Name() != "b" {
		t.Fatal(v2)
	}
}

func TestKeyStore1(t *testing.T) {
	st := NewStore("test1")

	_, err := st.Ensure("kv1", DSKeyValue)
	if err != nil {
		t.Fatal(err)
	}

	kv2, err := st.Get("kv1")
	if err != nil {
		t.Fatal(err)
	}

	kv := kv2.KeyValue()

	if err := kv.Set("a", "1"); err != nil {
		t.Fatal(err)
	}

	v, err := kv.Get("a")
	if err != nil {
		t.Fatal(err)
	}
	if string(v) != "1" {
		t.Fatal(v)
	}
}

func TestKeyStore2(t *testing.T) {
	st1 := NewStore("test2")

	kv1, err := st1.Create("kv1", DSKeyValue)
	if err != nil {
		t.Fatal(err)
	}

	kv := kv1.KeyValue()

	if err := kv.Set("a", "1"); err != nil {
		t.Fatal(err)
	}

	if err := st1.Write("./test"); err != nil {
		t.Fatal(err)
	}

	st2 := NewStore("test2")

	if err := st2.Read("./test"); err != nil {
		t.Fatal(err)
	}

	kv2, err := st2.Get("kv1")
	if err != nil {
		t.Fatal(err)
	}
	if kv2.Type != DSKeyValue {
		t.Fatal(kv2.Type)
	}

	kv = kv2.KeyValue()

	v, err := kv.Get("a")
	if err != nil {
		t.Fatal(err)
	}
	if string(v) != "1" {
		t.Fatal(v)
	}
}

func TestKeyStore3(t *testing.T) {
	st1 := NewStore("test3")

	kv1, err := st1.Create("kv1", DSStores)
	if err != nil {
		t.Fatal(err)
	}

	kv := kv1.Stores()

	if err := kv.Add(NewStore("a")); err != nil {
		t.Fatal(err)
	}

	if err := st1.Write("./test"); err != nil {
		t.Fatal(err)
	}

	st2 := NewStore("test3")

	if err := st2.Read("./test"); err != nil {
		t.Fatal(err)
	}

	kv2, err := st2.Get("kv1")
	if err != nil {
		t.Fatal(err)
	}
	if kv2.Type != DSStores {
		t.Fatal(kv2.Type)
	}

	kv = kv2.Stores()

	v, err := kv.Get("a")
	if err != nil {
		t.Fatal(err)
	}
	if v.Name() != "a" {
		t.Fatal(v)
	}
}
