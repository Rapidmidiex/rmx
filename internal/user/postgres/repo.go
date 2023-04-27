package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rapidmidiex/rmx/internal/user"
	"github.com/rapidmidiex/rmx/internal/user/postgres/sqlc"
)

type Repo interface {
	CreateUser(context.Context, user.User) (user.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (user.User, error)
	GetUserByEmail(ctx context.Context, email string) (user.User, error)
	ListUsers(context.Context) ([]user.User, error)
	UpdateUserByID(ctx context.Context, id uuid.UUID, username string) (user.User, error)
	UpdateUserByEmail(ctx context.Context, email, username string) (user.User, error)
	DeleteUserByID(ctx context.Context, id uuid.UUID) error
	DeleteUserByEmail(ctx context.Context, email string) error
}

type store struct {
	q *sqlc.Queries
}

func New(conn sqlc.DBTX) Repo {
	return &store{q: sqlc.New(conn)}
}

func (s *store) CreateUser(ctx context.Context, u user.User) (user.User, error) {
	created, err := s.q.CreateUser(ctx, &sqlc.CreateUserParams{
		Username: u.Username,
		Email:    u.Email,
	})

	return user.User{
		ID:       created.ID,
		Username: created.Username,
		Email:    created.Email.(string),
	}, err
}

func (s *store) GetUserByID(ctx context.Context, id uuid.UUID) (user.User, error) {
	found, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return user.User{}, err
	}

	return user.User{
		ID:       found.ID,
		Username: found.Username,
		Email:    found.Email.(string),
	}, nil
}

func (s *store) GetUserByEmail(ctx context.Context, email string) (user.User, error) {
	found, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return user.User{}, err
	}

	return user.User{
		ID:       found.ID,
		Username: found.Username,
		Email:    found.Email.(string),
	}, nil
}
