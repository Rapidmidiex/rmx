package user

import (
	"context"
	"time"

	psql "github.com/hyphengolang/prelude/sql/postgres"
	"github.com/hyphengolang/prelude/types/email"
	"github.com/hyphengolang/prelude/types/password"
	"github.com/hyphengolang/prelude/types/suid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rog-golang-buddies/rmx/internal"
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

type Repo interface {
	Closer
	Writer
	Reader
}

type ReadWriter interface {
	Reader
	Writer
}

type Writer interface {
	internal.RepoWriter[internal.User]
}

type Reader interface {
	internal.RepoReader[internal.User]
}

type Closer interface {
	internal.RepoCloser
}

type repo struct {
	ctx context.Context
	c   *pgxpool.Pool
}

// This is not really required
func NewRepo(ctx context.Context, conn *pgxpool.Pool) Repo {
	return &repo{ctx, conn}
}

func (r *repo) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}

func (r *repo) Close() { r.c.Close() }

func (r *repo) Insert(ctx context.Context, u *internal.User) error {
	args := pgx.NamedArgs{
		"id":       u.ID,
		"email":    u.Email,
		"username": u.Username,
		"password": u.Password,
	}

	return psql.Exec(r.c, qryInsert, args)
}

func (r *repo) SelectMany(ctx context.Context) ([]internal.User, error) {
	return psql.Query(r.c, qrySelectMany, func(r pgx.Rows, u *internal.User) error {
		return r.Scan(&u.ID, &u.Email, &u.Username, &u.Password)
	})
}

func (r *repo) Select(ctx context.Context, key any) (*internal.User, error) {
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
	return &u, psql.QueryRow(r.c, qry, func(r pgx.Row) error { return r.Scan(&u.ID, &u.Username, &u.Email, &u.Password) }, key)
}

func (r *repo) Delete(ctx context.Context, key any) error {
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
	return psql.Exec(r.c, qry, key)
}

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
