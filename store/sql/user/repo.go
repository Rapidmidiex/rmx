package user

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/fp"
	"github.com/rog-golang-buddies/rmx/internal/suid"
)

type Repo struct {
	q *Queries
}

func NewRepo(ctx context.Context, conn *sql.DB) internal.UserRepo {
	r := &Repo{New(conn)}
	return r
}

func UserRepo(c *sql.DB) *Repo {
	r := &Repo{q: New(c)}
	return r
}

// We need to setup the config to avoid needing to do this
func (r *Repo) SelectMany(ctx context.Context) ([]internal.User, error) {
	us, err := r.q.ListUsers(ctx)
	ius := fp.FMap(us, func(u User) internal.User {
		return internal.User{
			ID: suid.MustParse(u.ID), Username: u.Username, Email: internal.Email(u.Email),
			Password: internal.PasswordHash(u.Password),
		}
	})

	return ius, err
}

func (r *Repo) Select(ctx context.Context, key any) (u *internal.User, err error) {
	var v User
	switch key := key.(type) {
	case suid.UUID:
		v, err = r.q.GetUserByID(ctx, key.String())
	case internal.Email:
		v, err = r.q.GetUserByEmail(ctx, key.String())
	default:
		return nil, internal.ErrInvalidType
	}

	u = &internal.User{
		ID:       suid.MustParse(v.ID),
		Username: v.Username,
		Email:    internal.Email(v.Email),
		Password: internal.PasswordHash(v.Password),
	}

	return u, err
}

func (r *Repo) Insert(ctx context.Context, u *internal.User) error {
	v := CreateUserParams{
		Username:  u.Username,
		Email:     u.Email.String(),
		Password:  u.Password.String(),
		CreatedAt: time.Now(),
	}

	_, err := r.q.CreateUser(ctx, v)
	return err
}

func (r *Repo) Remove(ctx context.Context, key any) error {
	switch key := key.(type) {
	case suid.UUID:
		return r.q.DeleteUser(ctx, key.String())
	// case internal.Email:
	// v, err = r.q.GetUserByEmail(ctx, key.String())
	default:
		return internal.ErrInvalidType
	}
}

var DefaultRepo = &repo{
	miu:  make(map[suid.UUID]*User),
	mei:  make(map[string]*User),
	log:  log.Println,
	logf: log.Printf,
}

type repo struct {
	mu  sync.Mutex
	miu map[suid.UUID]*User
	mei map[string]*User

	log  func(v ...any)
	logf func(format string, v ...any)
}

// Remove implements internal.UserRepo
func (r *repo) Remove(ctx context.Context, key any) error {
	panic("unimplemented")
}

func (r *repo) Insert(ctx context.Context, iu *internal.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, found := r.mei[iu.Email.String()]; found {
		return internal.ErrAlreadyExists
	}

	u := &User{
		ID:        iu.ID.String(),
		Username:  iu.Username,
		Email:     iu.Email.String(),
		Password:  iu.Password.String(),
		CreatedAt: time.Now(),
	}
	r.mei[iu.Email.String()], r.miu[iu.ID] = u, u

	return nil
}

func (r *repo) SelectMany(ctx context.Context) ([]internal.User, error) {
	panic("not implemented")
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
			ID:       suid.MustParse(u.ID),
			Username: u.Username,
			Email:    internal.Email(u.Email),
			Password: internal.PasswordHash(u.Password),
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
				ID:       suid.MustParse(u.ID),
				Username: u.Username,
				Email:    internal.Email(u.Email),
				Password: internal.PasswordHash(u.Password),
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
			ID:       suid.MustParse(u.ID),
			Username: u.Username,
			Email:    internal.Email(u.Email),
			Password: internal.PasswordHash(u.Password),
		}, nil
	}

	return nil, internal.ErrNotFound
}
