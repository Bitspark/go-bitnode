package bitnode

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

// A User represents a client who uses a node.
type User struct {
	// ID of the User.
	ID ID `json:"id"`

	// Name of the user.
	Name string `json:"name"`
}

// Credentials represents user credentials.
type Credentials struct {
	Authority string `json:"authority"`
	Admin     bool   `json:"admin"`
	User      User   `json:"user"`
	Groups    []ID   `json:"groups"`
	Timestamp int64  `json:"timestamp"`

	Signature string `json:"signature"`
}

func (c *Credentials) Sign(secret string) {
	c.Signature = c.GetSignature(secret)
}

func (c *Credentials) IsValid(secret string) error {
	if c.Signature != c.GetSignature(secret) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

func (c *Credentials) GetSignature(secret string) string {
	dig := sha256.New()
	var bts [8]byte

	dig.Write([]byte(c.Authority))

	dig.Write(c.User.ID[:])

	if !c.Admin {
		dig.Write([]byte{0x00})
	} else {
		dig.Write([]byte{0xFF})
	}

	dig.Write([]byte(c.User.Name))
	dig.Write([]byte{0})

	for _, group := range c.Groups {
		dig.Write(group[:])
		dig.Write([]byte{0})
	}

	binary.PutVarint(bts[:], c.Timestamp)
	dig.Write(bts[:])

	dig.Write([]byte(secret))

	return base64.StdEncoding.EncodeToString(dig.Sum(nil))
}

func (c *Credentials) Tokenize() string {
	bts, _ := json.Marshal(c)
	return base64.StdEncoding.EncodeToString(bts)
}

func ParseCredentials(creds string) (Credentials, error) {
	c := Credentials{}
	bts, _ := base64.StdEncoding.DecodeString(creds)
	_ = json.Unmarshal(bts, &c)
	return c, nil
}
