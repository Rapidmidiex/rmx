package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rog-golang-buddies/rmx/internal/suid"
)

type userRepo struct {
	DBConn *sql.DB
}

var (
	errTodo     = errors.New("not yet implemented")
	errNotFound = errors.New("user not found")
	errExists   = errors.New("user already exists")
)

func UserRepo(c *sql.DB) *userRepo {
	r := &userRepo{
		DBConn: c,
	}
	return r
}

func (r *userRepo) ListAll() ([]User, error) {
	ctx := context.Background()
	q := New(r.DBConn)
	return q.ListUsers(ctx)
}

func (r *userRepo) Lookup(uid *suid.UUID) (User, error) {
	ctx := context.Background()
	q := New(r.DBConn)
	return q.GetUserByID(ctx, uid.String())
}

func (r *userRepo) LookupEmail(email string) (User, error) {
	ctx := context.Background()
	q := New(r.DBConn)
	return q.GetUserByEmail(ctx, email)
}

func (r *userRepo) Insert(u *CreateUserParams) error {
	ctx := context.Background()
	q := New(r.DBConn)
	_, err := q.CreateUser(ctx, u)
	if err != nil {
		return err
	}

	return nil
}

func (r *userRepo) Remove(uid *suid.UUID) error {
	ctx := context.Background()
	q := New(r.DBConn)
	return q.DeleteUser(ctx, uid.String())
}
