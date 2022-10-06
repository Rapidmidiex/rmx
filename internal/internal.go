package internal

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/rog-golang-buddies/rmx/internal/suid"
	gpv "github.com/wagslane/go-password-validator"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidEmail   = errors.New("invalid email")
	ErrNotImplemented = errors.New("not implemented")
	ErrInvalidType    = errors.New("invalid type")
	ErrAlreadyExists  = errors.New("already exists")
	ErrNotFound       = errors.New("not found")
	ErrContextValue   = errors.New("failed to retrieve value from context")
)

type ContextKey string

const (
	EmailKey = ContextKey("rmx-email")
)

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

func (t *MsgTyp) String() string {
	switch *t {
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

func (t *MsgTyp) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(t.String())
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

type UserRepo interface {
	RUserRepo
	WUserRepo
}

type RUserRepo interface {
	// Lookup(uid *suid.UUID) (User, error)
	// LookupEmail(email string) (User, error)
	// ListAll() ([]User, error)
}

type WUserRepo interface {
	Insert(ctx context.Context, u *User) error
	// Remove(uid *suid.UUID) error
}

type User struct {
	ID       suid.UUID    `json:"id"`
	Username string       `json:"username"`
	Email    Email        `json:"email"`
	Password PasswordHash `json:"-"`
}

// Custom email type required
type Email string

func (e *Email) String() string { return string(*e) }

func (e *Email) IsValid() bool { return e.Valid() == nil }

func (e *Email) Valid() error {
	_, err := mail.ParseAddress(e.String())
	if err != nil {
		return err
	}
	return nil
}

func (e *Email) UnmarshalJSON(b []byte) error {
	*e = Email(b[1 : len(b)-1])
	return e.Valid()
}

// during production, this value needs to be > 40
const minEntropy float64 = 10.0

// Custom password type required
type Password string

func (p *Password) String() string { return string(*p) }

func (p *Password) IsValid() bool { return p.Valid() == nil }

func (p *Password) Valid() error {
	return gpv.Validate(p.String(), minEntropy)
}

func (p *Password) UnmarshalJSON(b []byte) error {
	*p = Password(b[1 : len(b)-1])
	return p.Valid()
}

func (p *Password) MarshalJSON() (b []byte, err error) {
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(string(*p))
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}

func (p *Password) Hash() (PasswordHash, error) { return newPasswordHash(*p) }

type PasswordHash []byte

func (h *PasswordHash) String() string { return string(*h) }

func newPasswordHash(pw Password) (PasswordHash, error) {
	return bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
}

func (h *PasswordHash) Compare(cmp string) error {
	return bcrypt.CompareHashAndPassword(*h, []byte(cmp))
}
