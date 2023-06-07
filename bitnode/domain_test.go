package bitnode

import (
	"gopkg.in/yaml.v3"
	"testing"
)

func TestDomain1(t *testing.T) {
	dom := NewDomain()
	a, err := dom.AddDomain("a")
	if err != nil {
		t.Fatal(err)
	}
	_, err = dom.AddDomain("b")
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.AddDomain("1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.AddDomain("2")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDomain2(t *testing.T) {
	dom := NewDomain()
	a, _ := dom.AddDomain("a")
	b, _ := dom.AddDomain("b")
	a1, _ := a.AddDomain("1")
	a2, _ := a.AddDomain("2")

	if d, err := dom.GetDomain("a"); err != nil || d != a {
		t.Fatal(err, d)
	}
	if d, err := dom.GetDomain("b"); err != nil || d != b {
		t.Fatal(err, d)
	}
	if d, err := dom.GetDomain("a.1"); err != nil || d != a1 {
		t.Fatal(err, d)
	}
	if d, err := dom.GetDomain("a.2"); err != nil || d != a2 {
		t.Fatal(err, d)
	}
}

func TestDomain3(t *testing.T) {
	dom := NewDomain()
	a, _ := dom.AddDomain("a")
	b, _ := dom.AddDomain("b")
	a1, _ := a.AddDomain("1")
	a2, _ := a.AddDomain("2")
	_ = b
	_ = a2

	if d, err := a.GetDomain(".1"); err != nil || d != a1 {
		t.Fatal(err, d)
	}
	if d, err := a.GetDomain("a.1"); err != nil || d != a1 {
		t.Fatal(err, d)
	}
	if d, err := a1.GetDomain("."); err != nil || d != a1 {
		t.Fatal(err, d)
	}
}

func TestDomain4(t *testing.T) {
	dom := NewDomain()
	a, _ := dom.AddDomain("a")
	b, _ := dom.AddDomain("b")
	a1, _ := a.AddDomain("1")
	a2, _ := a.AddDomain("2")
	a1x, _ := a1.AddDomain("x")
	_ = b
	_ = a2

	if d, err := a1x.GetDomain("."); err != nil || d != a1x {
		t.Fatal(err, d, d.FullName)
	}
	if d, err := a1x.GetDomain(".$"); err != nil || d != a1 {
		t.Fatal(err, d, d.FullName)
	}
	if d, err := a1x.GetDomain(".$.x"); err != nil || d != a1x {
		t.Fatal(err, d, d.FullName)
	}
}

func TestDomain__YAML1(t *testing.T) {
	dom := NewDomain()
	dom.Name = "test"
	yamlBts, err := yaml.Marshal(dom)
	if err != nil {
		t.Fatal(err)
	}
	var dom2 Domain
	err = yaml.Unmarshal(yamlBts, &dom2)
	if err != nil {
		t.Fatal(err)
	}
	if dom2.Name != "test" {
		t.Fatal(dom2.Name)
	}
}

func TestPermissions__YAML1(t *testing.T) {
	perms := Permissions{
		Owner: ComposeIDs(GenerateSystemID(), GenerateObjectID()),
	}
	yamlBts, err := yaml.Marshal(perms)
	if err != nil {
		t.Fatal(err)
	}
	var perms2 Permissions
	err = yaml.Unmarshal(yamlBts, &perms2)
	if err != nil {
		t.Fatal(err)
	}
	if perms2.Owner != perms.Owner {
		t.Fatal(perms2.Owner)
	}
}
