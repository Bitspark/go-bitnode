package bitnode

import (
	"testing"
	"time"
)

func TestCredentials1(t *testing.T) {
	creds := Credentials{
		Authority: "bitspark",
		User:      User{},
		Timestamp: time.Now().Unix(),
		Signature: "",
	}

	if err := creds.IsValid(""); err == nil {
		t.Fatal()
	}
	if err := creds.IsValid("test"); err == nil {
		t.Fatal()
	}

	creds.Sign("test")

	if err := creds.IsValid("test"); err != nil {
		t.Fatal(err)
	}
	if err := creds.IsValid("test2"); err == nil {
		t.Fatal()
	}

	creds.Authority = "evil"
	if err := creds.IsValid("test"); err == nil {
		t.Fatal()
	}
}

func TestParseCredentials1(t *testing.T) {
	id := ComposeIDs(GenerateSystemID(), GenerateObjectID())

	creds1 := Credentials{
		Authority: "a",
		User: User{
			ID:   id,
			Name: "u",
		},
		Timestamp: 123,
		Signature: "",
	}
	creds1.Sign("test")

	str := creds1.Tokenize()

	t.Log(str)

	creds2, err := ParseCredentials(str)
	if err != nil {
		t.Fatal(err)
	}

	if err := creds2.IsValid("test"); err != nil {
		t.Fatal(err)
	}
}
