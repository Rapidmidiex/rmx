package suid

import (
	"github.com/google/uuid"
	suid "github.com/lithammer/shortuuid/v4"
)

type UUID struct{ uuid.UUID }

func NewUUID() UUID { return UUID{uuid.New()} }

func (u UUID) ShortUUID() SUID { return SUID(suid.DefaultEncoder.Encode(u.UUID)) }

type SUID string

func NewSUID() SUID { return SUID(suid.New()) }

func (sid SUID) String() string { return string(sid) }

func FromUUID(uid UUID) SUID { return SUID(suid.DefaultEncoder.Encode(uid.UUID)) }

// Decode decodes a short uuid string into a suid.UUID.
// If s is too short, its most significant bits (MSB) will be padded with 0 (zero).
func ParseString(s string) (UUID, error) {
	uid, err := suid.DefaultEncoder.Decode(s)
	return UUID{uid}, err
}

// Decode decodes a short uuid string into a suid.UUID.
// If s is too short, its most significant bits (MSB) will be padded with 0 (zero).
//
// This will panic if there is an error.
func MustParse(s string) UUID {
	uid, err := ParseString(s)
	if err != nil {
		panic(err)
	}
	return uid
}

func (s SUID) UUID() (UUID, error) {
	u, err := suid.DefaultEncoder.Decode(string(s))
	return UUID{u}, err
}
