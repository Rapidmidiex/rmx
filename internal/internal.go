package internal

import (
	"errors"
	"regexp"
	"strings"

	"github.com/rog-golang-buddies/rmx/internal/suid"
	gpv "github.com/wagslane/go-password-validator"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidEmail = errors.New("rmx: invalid email")
var ErrNotImplemented = errors.New("rmx: not yet implemented")

type JamRepo interface {
}

type User struct {
	ID       suid.UUID    `json:"id"`
	Email    Email        `json:"email"`
	Username string       `json:"username"`
	Password PasswordHash `json:"-"`
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
	SignUp(u User) error
}

type MsgTyp int

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

func (t MsgTyp) String() string {
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

func (t *MsgTyp) UnmarshalJSON(b []byte) error {
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

func (t MsgTyp) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(t.String())
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&’*+/=?^_\x60{|}~-]+@[a-zA-Z0-9-]+(?:\.[a-zA-Z0-9-]+)*$`)

type Email string

func (e *Email) UnmarshalJSON(b []byte) error {
	if s := b[1 : len(b)-1]; emailRegex.Match(s) {
		*e = Email(s)
		return nil
	}

	return ErrInvalidEmail
}

const minEntropy float64 = 10.0

type Password string

func (p *Password) UnmarshalJSON(b []byte) error {
	*p = Password(b[1 : len(b)-1])
	return gpv.Validate(string(*p), minEntropy)
}

func (p Password) MarshalJSON() (b []byte, err error) {
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(string(p))
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

func (p Password) Hash() (PasswordHash, error) { return newPasswordHash(p) }

type PasswordHash []byte

func newPasswordHash(pw Password) (PasswordHash, error) {
	return bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
}

func (pw PasswordHash) Compare(cmp Password) error {
	return bcrypt.CompareHashAndPassword(pw, []byte(cmp))
}
