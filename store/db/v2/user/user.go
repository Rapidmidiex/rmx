package user

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/suid"
)

type User struct {
	ID        suid.UUID
	Username  string
	Email     string
	Password  internal.PasswordHash
	CreatedAt time.Time
}

var MapRepo = &repo{miu: make(map[suid.UUID]*User), mei: make(map[string]*User), log: log.Println, logf: log.Printf}

type repo struct {
	mu  sync.Mutex
	miu map[suid.UUID]*User
	mei map[string]*User

	log  func(v ...any)
	logf func(format string, v ...any)
}

func (r *repo) Insert(ctx context.Context, iu *internal.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, found := r.mei[iu.Email.String()]; found {
		return internal.ErrAlreadyExists
	}

	u := &User{iu.ID, iu.Username, iu.Email.String(), iu.Password, time.Now()}
	r.mei[iu.Email.String()], r.miu[iu.ID] = u, u

	return nil
}
