package suid

import (
	"github.com/google/uuid"
	suid "github.com/lithammer/shortuuid/v4"
)

// ? The API for UUID/SUID is still in development
type UUID struct{ uuid.UUID }

func NewUUID() UUID { return UUID{uuid.New()} }

func (u UUID) ShortUUID() SUID { return SUID(suid.DefaultEncoder.Encode(u.UUID)) }

type SUID string

func NewSUID() SUID { return SUID(suid.New()) }

func FromUUID(uid UUID) SUID { return SUID(suid.DefaultEncoder.Encode(uid.UUID)) }

func ParseString(s string) (UUID, error) {
	uid, err := suid.DefaultEncoder.Decode(s)
	return UUID{uid}, err
}

func (s SUID) UUID() (UUID, error) {
	u, err := suid.DefaultEncoder.Decode(string(s))
	return UUID{u}, err
}
