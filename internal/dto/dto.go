package dto

import (
	"errors"
	"net/mail"

	"github.com/rog-golang-buddies/rmx/internal/suid"
	gpv "github.com/wagslane/go-password-validator"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidEmail = errors.New("rmx: invalid email")
var ErrNotImplemented = errors.New("rmx: not yet implemented")

type JamRepo interface {
}

type User struct {
	ID       suid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    Email     `json:"email"`
	Password Password  `json:"-"`
}

func (u *User) HashPassword() error {
	h, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = Password(h)
	return nil
}

func (u *User) ComparePassword(p string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(p))
}

type UserRepo interface {
	RUserRepo
	WUserRepo
}

type RUserRepo interface {
	Lookup(uid *suid.UUID) (User, error)
	LookupEmail(email string) (User, error)
	ListAll() ([]User, error)
}

type WUserRepo interface {
	Add(u *User) error
	Remove(uid *suid.UUID) error
}

/*
var emailRegex = regexp.MustCompile(
	`^[a-zA-Z0-9.!#$%&’*+/=?^_\x60{|}~-]+@[a-zA-Z0-9-]+(?:\.[a-zA-Z0-9-]+)*$`,
)
*/

type Email string

func (e *Email) String() string { return string(*e) }

func (e *Email) Validate() error {
	_, err := mail.ParseAddress(e.String())
	if err != nil {
		return err
	}
	return nil
}

/*
func (e *Email) UnmarshalJSON(b []byte) error {
	if s := b[1 : len(b)-1]; emailRegex.Match(s) {
		*e = Email(s)
		return nil
	}

	return ErrInvalidEmail
}
*/

// NOTE: better change to something in range 50-70
const minEntropy float64 = 10.0

type Password string

func (p *Password) String() string { return string(*p) }

func (p *Password) Validate() error {
	return gpv.Validate(p.String(), minEntropy)
}

/*
func (p *Password) UnmarshalJSON(b []byte) error {
	*p = Password(b[1 : len(b)-1])
	return gpv.Validate(string(*p), minEntropy)
}

func (p *Password) MarshalJSON() (b []byte, err error) {
	var sb strings.Builder
	sb.WriteRune('"')
	sb.WriteString(string(*p))
	sb.WriteRune('"')
	return []byte(sb.String()), nil
}
*/

func (p *Password) Hash() (PasswordHash, error) { return newPasswordHash(*p) }

type PasswordHash []byte

func newPasswordHash(pw Password) (PasswordHash, error) {
	return bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
}

func (pw *PasswordHash) Compare(cmp Password) error {
	return bcrypt.CompareHashAndPassword(*pw, []byte(cmp))
}
