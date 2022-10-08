package user

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/suid"
)

/*
CREATE TABLE users (

	id text NOT NULL PRIMARY KEY,
	username text NOT NULL,
	email text NOT NULL,
	password text NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NULL DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP,
	deleted_at TIMESTAMP NULL DEFAULT NULL,
	UNIQUE (email)

);
*/
type User struct {
	ID        suid.UUID
	Username  string
	Email     internal.Email
	Password  internal.PasswordHash
	CreatedAt time.Time
	// UpdatedAt *time.Time
	// DeletedAt *time.Time
}

type Repo struct {
	ctx context.Context

	// connection is using pgx until SQLC is sorted
	c *pgx.Conn
}

func (r *Repo) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}

func (r *Repo) Close(ctx context.Context) error { return r.c.Close(ctx) }

func NewRepo(ctx context.Context, conn *pgx.Conn) internal.UserRepo {
	r := &Repo{ctx, conn}
	return r
}

func (r *Repo) Insert(ctx context.Context, iu *internal.User) error {
	qry := `insert into "user" (id, email, username, password) values ($1, $2, $3, $4)`
	_, err := r.c.Exec(context.Background(), qry, iu.ID, iu.Email, iu.Username, iu.Password.String())
	return err
}

// We need to setup the config to avoid needing to do this
func (r *Repo) SelectMany(ctx context.Context) ([]internal.User, error) {
	qry := `select id, email, username, password from "user" order by id`

	row, err := r.c.Query(context.Background(), qry)
	if err != nil {
		return nil, err
	}
	defer row.Close()

	var ius []internal.User
	for row.Next() {
		var u User
		if err := row.Scan(&u.ID, &u.Email, &u.Username, &u.Password); err != nil {
			return nil, err
		}

		iu := internal.User{ID: u.ID, Email: internal.Email(u.Email), Username: u.Username, Password: u.Password}
		ius = append(ius, iu)
	}

	return ius, nil
}

func (r Repo) Select(ctx context.Context, key any) (*internal.User, error) {
	switch key := key.(type) {
	case suid.UUID:
		return r.selectUUID(ctx, `select id, email, username, password from "user" where id = $1`, key)
	case internal.Email:
		return r.selectEmail(ctx, `select id, email, username, password from "user" where email = $1`, key)
	case string:
		return r.selectUsername(ctx, `select id, email, username, password from "user" where username = $1`, key)
	}

	return nil, internal.ErrInvalidType
}

func (r Repo) selectUUID(ctx context.Context, qry string, uid suid.UUID) (*internal.User, error) {
	row := r.c.QueryRow(ctx, qry, uid)

	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Username, &u.Password)

	iu := &internal.User{ID: u.ID, Email: internal.Email(u.Email), Username: u.Username, Password: u.Password}
	return iu, err
}

func (r Repo) selectEmail(ctx context.Context, qry string, email internal.Email) (*internal.User, error) {
	row := r.c.QueryRow(ctx, qry, email)

	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Username, &u.Password)

	iu := &internal.User{ID: u.ID, Email: internal.Email(u.Email), Username: u.Username, Password: u.Password}
	return iu, err
}

func (r Repo) selectUsername(ctx context.Context, qry string, username string) (*internal.User, error) {
	row := r.c.QueryRow(ctx, qry, username)

	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Username, &u.Password)

	iu := &internal.User{ID: u.ID, Email: internal.Email(u.Email), Username: u.Username, Password: u.Password}
	return iu, err
}

func (r Repo) Delete(ctx context.Context, key any) error {
	switch key := key.(type) {
	case suid.UUID:
		return r.deleteUUID(ctx, `delete from "user" where id = $1`, key)
	case internal.Email:
		return r.deleteEmail(ctx, `delete from "user" where email = $1`, key)
	case string:
		return r.deleteUsername(ctx, `delete from "user" where username = $1`, key)
	}

	return internal.ErrNotImplemented
}

func (r Repo) deleteUUID(ctx context.Context, qry string, uid suid.UUID) error {
	_, err := r.c.Exec(ctx, qry, uid.String())
	return err
}

func (r Repo) deleteEmail(ctx context.Context, qry string, email internal.Email) error {
	_, err := r.c.Exec(ctx, qry, email.String())
	return err
}

func (r Repo) deleteUsername(ctx context.Context, qry string, username string) error {
	_, err := r.c.Exec(ctx, qry, username)
	return err
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
				ID:       u.ID,
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
			ID:       u.ID,
			Username: u.Username,
			Email:    internal.Email(u.Email),
			Password: internal.PasswordHash(u.Password),
		}, nil
	}

	return nil, internal.ErrNotFound
}
