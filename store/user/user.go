package user

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/suid"

	"github.com/rog-golang-buddies/rmx/store/sql/user"
)

type Repo struct {
	c *sql.DB
}

func New(ctx context.Context) internal.UserRepo {
	var conn *sql.DB

	user.New(conn)
	return nil
}

type User struct {
	ID        suid.UUID
	Username  string
	Email     internal.Email
	Password  internal.PasswordHash
	CreatedAt time.Time
}

var DefaultRepo = &repo{miu: make(map[suid.UUID]*User), mei: make(map[string]*User), log: log.Println, logf: log.Printf}

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

	u := &User{iu.ID, iu.Username, iu.Email, iu.Password, time.Now()}
	r.mei[iu.Email.String()], r.miu[iu.ID] = u, u

	return nil
}

func (r *repo) Select(ctx context.Context, key any) (*internal.User, error) {
	switch key := key.(type) {
	case suid.UUID:
		return r.selectUUID(key)
	case internal.Email:
		return r.selectEmail(key)
	case string:
		return r.selectUsername(key)
	default:
		return nil, internal.ErrInvalidType
	}
}

func (r *repo) selectUUID(uid suid.UUID) (*internal.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if u, ok := r.miu[uid]; ok {
		return &internal.User{
			ID:       u.ID,
			Username: u.Username,
			Email:    internal.Email(u.Email),
			Password: u.Password,
		}, nil
	}

	return nil, internal.ErrNotFound
}

func (r *repo) selectUsername(username string) (*internal.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.mei {
		if u.Username == username {
			return &internal.User{
				ID:       u.ID,
				Username: u.Username,
				Email:    internal.Email(u.Email),
				Password: u.Password,
			}, nil
		}
	}

	return nil, internal.ErrNotFound
}

func (r *repo) selectEmail(email internal.Email) (*internal.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if u, ok := r.mei[email.String()]; ok {
		return &internal.User{
			ID:       u.ID,
			Username: u.Username,
			Email:    internal.Email(u.Email),
			Password: u.Password,
		}, nil
	}

	return nil, internal.ErrNotFound
}
