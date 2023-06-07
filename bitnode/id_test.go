package bitnode

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestID_MarshalYAML(t *testing.T) {
	id := ComposeIDs(GenerateSystemID(), GenerateObjectID())
	idBts, err := yaml.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(idBts))
	id2 := ID{}
	err = yaml.Unmarshal(idBts, &id2)
	if err != nil {
		t.Fatal(err)
	}
	if id2 != id {
		t.Fatal()
	}
}

func TestID_MarshalJSON(t *testing.T) {
	id := ComposeIDs(GenerateSystemID(), GenerateObjectID())
	idBts, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(idBts))
	id2 := ID{}
	err = json.Unmarshal(idBts, &id2)
	if err != nil {
		t.Fatal(err)
	}
	if id2 != id {
		t.Fatal()
	}
}

func TestIDList_MarshalYAML(t *testing.T) {
	ids := IDList{
		ComposeIDs(GenerateSystemID(), GenerateObjectID()),
		ComposeIDs(GenerateSystemID(), GenerateObjectID()),
	}
	idsBts, err := yaml.Marshal(ids)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(idsBts))
	ids2 := IDList{}
	err = yaml.Unmarshal(idsBts, &ids2)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids2) != 2 || ids2[0] != ids[0] || ids2[1] != ids[1] {
		t.Fatal()
	}
}

func TestIDList_MarshalJSON(t *testing.T) {
	ids := IDList{
		ComposeIDs(GenerateSystemID(), GenerateObjectID()),
		ComposeIDs(GenerateSystemID(), GenerateObjectID()),
	}
	idsBts, err := json.Marshal(ids)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(idsBts))
	ids2 := IDList{}
	err = json.Unmarshal(idsBts, &ids2)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids2) != 2 || ids2[0] != ids[0] || ids2[1] != ids[1] {
		t.Fatal()
	}
}
