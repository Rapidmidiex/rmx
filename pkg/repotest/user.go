package repotest

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/hyphengolang/prelude/types/email"
	"github.com/hyphengolang/prelude/types/password"
	"github.com/hyphengolang/prelude/types/suid"

	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/store/user"
)

type UserRepo interface {
	user.Repo
}

func (r *repo) Close() {}

func (r *repo) Delete(ctx context.Context, key any) error { return nil }

type repo struct {
	mu  sync.Mutex
	miu map[suid.UUID]*user.User
	mei map[string]*user.User

	log  func(v ...any)
	logf func(format string, v ...any)
}

func NewUserRepo() UserRepo {
	r := &repo{
		miu:  make(map[suid.UUID]*user.User),
		mei:  make(map[string]*user.User),
		log:  log.Println,
		logf: log.Printf,
	}

	return r
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

	u := &user.User{
		ID:        iu.ID,
		Username:  iu.Username,
		Email:     iu.Email,
		Password:  iu.Password,
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
	case email.Email:
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
			Email:    email.Email(u.Email),
			Password: password.PasswordHash(u.Password),
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
				Email:    email.Email(u.Email),
				Password: password.PasswordHash(u.Password),
			}, nil
		}
	}

	return nil, internal.ErrNotFound
}

func (r *repo) selectEmail(email email.Email) (*internal.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if u, ok := r.mei[email.String()]; ok {
		return &internal.User{
			ID:       u.ID,
			Username: u.Username,
			Email:    u.Email,
			Password: password.PasswordHash(u.Password),
		}, nil
	}

	return nil, internal.ErrNotFound
}
