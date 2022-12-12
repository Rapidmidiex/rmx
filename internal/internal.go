package internal

import (
	"context"
	"errors"
	"strings"

	"github.com/hyphengolang/prelude/types/email"
	"github.com/hyphengolang/prelude/types/password"
	"github.com/hyphengolang/prelude/types/suid"
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
	EmailKey   = ContextKey("account-email")
	TokenKey   = ContextKey("jwt-token-key")
	RoomKey    = ContextKey("conn-pool-key")
	UpgradeKey = ContextKey("upgrade-http-key")
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

type TokenClient interface {
	RTokenClient
	WTokenClient
}

type RTokenClient interface {
	ValidateRefreshToken(ctx context.Context, token string) error
	ValidateClientID(ctx context.Context, cid string) error
}

type WTokenClient interface {
	BlackListClientID(ctx context.Context, cid, email string) error
	BlackListRefreshToken(ctx context.Context, token string) error
}

type TokenReader interface {
	ValidateRefreshToken(ctx context.Context, token string) error
	ValidateClientID(ctx context.Context, cid string) error
}

type TokenWriter interface {
	BlackListClientID(ctx context.Context, cid, email string) error
	BlackListRefreshToken(ctx context.Context, token string) error
}

type RepoReader[Entry any] interface {
	// Returns an array of users subject to any filter
	// conditions that are required
	SelectMany(ctx context.Context) ([]Entry, error)
	// Returns a user form the database, the "key"
	// can be either the "id", "email" or "username"
	// as these are all given unique values
	Select(ctx context.Context, key any) (*Entry, error)
}

type RepoWriter[Entry any] interface {
	// Insert a new item to the database
	Insert(ctx context.Context, e *Entry) error
	// Performs a "hard" delete from database
	// Restricted to admin only
	Delete(ctx context.Context, key any) error
}

type RepoCloser interface {
	Close()
}

// Custom user type required
type User struct {
	ID       suid.UUID             `json:"id"`
	Username string                `json:"username"`
	Email    email.Email           `json:"email"`
	Password password.PasswordHash `json:"-"`
}
