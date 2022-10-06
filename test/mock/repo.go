package mock

import (
	"errors"
	"sync"

	"github.com/rog-golang-buddies/rmx/internal/dto"
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
	us map[suid.UUID]dto.User
}

func UserRepo() *userRepo {
	r := &userRepo{
		us: map[suid.UUID]dto.User{},
	}
	return r
}

func (r *userRepo) ListAll() ([]dto.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return maps.Values(r.us), nil
}

func (r *userRepo) Lookup(uid *suid.UUID) (dto.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.us {
		if &u.ID == uid {
			return u, nil
		}
	}

	return dto.User{}, errNotFound
}

func (r *userRepo) LookupEmail(email string) (dto.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.us {
		if string(u.Email) == email {
			return u, nil
		}
	}

	return dto.User{}, errNotFound
}

func (r *userRepo) Add(u *dto.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, v := range r.us {
		if v.Username == u.Username {
			return errExists
		}
	}

	r.us[u.ID] = *u

	return nil
}

func (r *userRepo) Remove(uid *suid.UUID) error {
	return nil
}
