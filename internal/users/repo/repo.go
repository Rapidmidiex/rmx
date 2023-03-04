package repo

import (
	"context"
	"errors"
	"sync"

	"github.com/rapidmidiex/rmx/internal/users"
)

// THe Repo interface allows read and write access
type Repo interface {
	Reader
	Writer
}

// The Writer interface allows other services to
// write to the database without read access.
type Writer interface {
	// CreateUser adds a new user to the database
	CreateUser(ctx context.Context, user users.User) (users.User, error)
}

// The Reader interface allows other services to
// read from the database without write access.
type Reader interface {
	HasUniqueUsername(ctx context.Context, username string) error
}

// New in-memory ReadWriter
func New() Repo {
	return &repo{
		m: make(map[string]users.User),
	}
}

type repo struct {
	mu sync.Mutex
	// m is a map of username to user
	m map[string]users.User
}

func (r *repo) HasUniqueUsername(ctx context.Context, username string) error {
	panic("unimplemented")
}

func (r *repo) CreateUser(ctx context.Context, user users.User) (users.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[user.Username]; ok {
		return user, errors.New("username already exists")
	}

	r.m[user.Username] = user
	return user, nil
}
