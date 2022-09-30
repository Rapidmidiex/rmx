package internal

import (
	"errors"

	"github.com/rog-golang-buddies/rmx/internal/suid"
)

var ErrTodo = errors.New("rmx: not yet implemented")

type MessageTyp int

const (
	Unknown = iota

	Create
	Delete

	Join
	Leave
	Message

	NoteOn
	NoteOff
)

func (t MessageTyp) String() string {
	switch t {
	case Create:
		return "CREATE"
	case Delete:
		return "DELETE"
	case Join:
		return "JOIN"
	case Leave:
		return "LEAVE"
	case Message:
		return "MESSAGE"
	case NoteOn:
		return "NOTE_ON"
	case NoteOff:
		return "NOTE_OFF"

	default:
		return "UNKNOWN"
	}
}

func (t *MessageTyp) UnmarshalJSON(b []byte) error {
	switch s := string(b[1 : len(b)-1]); s {
	case "CREATE":
		*t = Create
	case "DELETE":
		*t = Delete
	case "JOIN":
		*t = Join
	case "LEAVE":
		*t = Leave
	case "MESSAGE":
		*t = Message
	case "NOTE_ON":
		*t = NoteOn
	case "NOTE_OFF":
		*t = NoteOff
	default:
		*t = Unknown
	}

	return nil
}

func (t MessageTyp) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

// TODO
type Email string

type JamRepo interface {
}

type User struct {
	ID    suid.UUID
	Email Email
}

type UserRepo interface {
	RUserRepo
	WUserRepo
}

type RUserRepo interface {
	Lookup(id suid.UUID) (*User, error)
	LookupEmail(email Email) (*User, error)
	ListAll() ([]*User, error)
}

type WUserRepo interface {
	SignUp(u *User) error
}
