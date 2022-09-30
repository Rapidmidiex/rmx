package mock

import (
	"errors"
	"sync"

	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/suid"
	"golang.org/x/exp/maps"
)

var (
	errTodo     = errors.New("not yet implemented")
	errNotFound = errors.New("user not found")
	errExists   = errors.New("user already exists")
)

type userRepo struct {
	mu sync.Mutex
	us map[suid.UUID]*internal.User
}

func UserRepo() *userRepo {
	r := &userRepo{
		us: map[suid.UUID]*internal.User{},
	}
	return r
}

func (r *userRepo) ListAll() ([]*internal.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return maps.Values(r.us), nil
}

func (r *userRepo) Lookup(uid suid.UUID) (*internal.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.us {
		if u.ID == uid {
			return u, nil
		}
	}

	return nil, errNotFound
}

func (r *userRepo) LookupEmail(email internal.Email) (*internal.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.us {
		if u.Email == email {
			return u, nil
		}
	}

	return nil, errNotFound
}

func (r *userRepo) SignUp(u internal.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, found := r.us[u.ID]; found {
		return errExists
	} else {
		r.us[u.ID] = &u
	}

	return nil
}
