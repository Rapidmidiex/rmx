package user

import (
	"context"
	"log"
	"sync"
	"time"

	psql "github.com/hyphengolang/prelude/sql/postgres"
	"github.com/hyphengolang/prelude/types/email"
	"github.com/hyphengolang/prelude/types/password"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/suid"
)

const (
	qryInsert = `insert into "user" (id, email, username, password) values (@id, @email, @username, @password)`

	qrySelectMany = `select id, email, username, password from "user" order by id`

	qrySelectByID       = `select id, email, username, password from "user" where id = $1`
	qrySelectByEmail    = `select id, email, username, password from "user" where email = $1`
	qrySelectByUsername = `select id, email, username, password from "user" where username = $1`

	qryDeleteByID       = `delete from "user" where id = $1`
	qryDeleteByEmail    = `delete from "user" where email = $1`
	qryDeleteByUsername = `delete from "user" where username = $1`
)

// Definition of our User in the DB layer
type User struct {
	// Primary key.
	ID suid.UUID
	// Unique. Stored as text.
	Username string
	// Unique. Stored as case-sensitive text.
	Email email.Email
	// Required. Stored as case-sensitive text.
	Password password.PasswordHash
	// Required. Defaults to current time.
	CreatedAt time.Time
	// TODO nullable, currently inactive
	// UpdatedAt *time.Time
	// TODO nullable, currently inactive
	// DeletedAt *time.Time
}

type Repo struct {
	ctx context.Context
	c   *pgxpool.Pool
}

// This is not really required
func NewRepo(ctx context.Context, conn *pgxpool.Pool) *Repo {
	return &Repo{ctx, conn}
}

func (s *Repo) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

func (s *Repo) Close() { s.c.Close() }

func (s *Repo) Insert(ctx context.Context, u *internal.User) error {
	args := pgx.NamedArgs{
		"id":       u.ID,
		"email":    u.Email,
		"username": u.Username,
		"password": u.Password,
	}

	return psql.Exec(s.c, qryInsert, args)
}

func (s *Repo) SelectMany(ctx context.Context) ([]internal.User, error) {
	return psql.Query(s.c, qrySelectMany, func(r pgx.Rows, u *internal.User) error {
		return r.Scan(&u.ID, &u.Email, &u.Username, &u.Password)
	})
}

func (s *Repo) Select(ctx context.Context, key any) (*internal.User, error) {
	var qry string
	switch key.(type) {
	case suid.UUID:
		qry = qrySelectByID
	case email.Email:
		qry = qrySelectByEmail
	case string:
		qry = qrySelectByUsername
	default:
		return nil, internal.ErrInvalidType
	}
	var u internal.User
	return &u, psql.QueryRow(s.c, qry, func(r pgx.Row) error { return r.Scan(&u.ID, &u.Username, &u.Email, &u.Password) }, key)
}

func (s *Repo) Delete(ctx context.Context, key any) error {
	var qry string
	switch key.(type) {
	case suid.UUID:
		qry = qryDeleteByID
	case email.Email:
		qry = qryDeleteByEmail
	case string:
		qry = qryDeleteByUsername
	default:
		return internal.ErrInvalidType
	}
	return psql.Exec(s.c, qry, key)
}

// NOTE in-memory implementation not required anymore

var DefaultRepo = &store{
	miu:  make(map[suid.UUID]*User),
	mei:  make(map[string]*User),
	log:  log.Println,
	logf: log.Printf,
}

func (s *store) Close() {}

func (s *store) Delete(ctx context.Context, key any) error { return nil }

type store struct {
	mu  sync.Mutex
	miu map[suid.UUID]*User
	mei map[string]*User

	log  func(v ...any)
	logf func(format string, v ...any)
}

// Remove implements internal.UserRepo
func (s *store) Remove(ctx context.Context, key any) error {
	panic("unimplemented")
}

func (s *store) Insert(ctx context.Context, iu *internal.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, found := s.mei[iu.Email.String()]; found {
		return internal.ErrAlreadyExists
	}

	u := &User{
		ID:        iu.ID,
		Username:  iu.Username,
		Email:     iu.Email,
		Password:  iu.Password,
		CreatedAt: time.Now(),
	}
	s.mei[iu.Email.String()], s.miu[iu.ID] = u, u

	return nil
}

func (s *store) SelectMany(ctx context.Context) ([]internal.User, error) {
	panic("not implemented")
}

func (s *store) Select(ctx context.Context, key any) (*internal.User, error) {
	switch key := key.(type) {
	case suid.UUID:
		return s.selectUUID(key)
	case email.Email:
		return s.selectEmail(key)
	case string:
		return s.selectUsername(key)
	default:
		return nil, internal.ErrInvalidType
	}
}

func (s *store) selectUUID(uid suid.UUID) (*internal.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if u, ok := s.miu[uid]; ok {
		return &internal.User{
			ID:       u.ID,
			Username: u.Username,
			Email:    email.Email(u.Email),
			Password: password.PasswordHash(u.Password),
		}, nil
	}

	return nil, internal.ErrNotFound
}

func (s *store) selectUsername(username string) (*internal.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, u := range s.mei {
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

func (s *store) selectEmail(email email.Email) (*internal.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if u, ok := s.mei[email.String()]; ok {
		return &internal.User{
			ID:       u.ID,
			Username: u.Username,
			Email:    u.Email,
			Password: password.PasswordHash(u.Password),
		}, nil
	}

	return nil, internal.ErrNotFound
}
