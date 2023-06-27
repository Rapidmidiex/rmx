package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/rapidmidiex/rmx/internal/auth/store/sqlc"
	"github.com/rapidmidiex/rmx/internal/cache"
	"github.com/rapidmidiex/rmx/internal/events"
)

type Repo interface {
	CreateUser(context.Context, *auth.User) (*auth.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*auth.User, error)
	GetUserByEmail(ctx context.Context, email string) (*auth.User, error)
	ListUsers(ctx context.Context) ([]auth.User, error)
	UpdateUserByID(ctx context.Context, id uuid.UUID, username string) (*auth.User, error)
	UpdateUserByEmail(ctx context.Context, email, username string) (*auth.User, error)
	DeleteUserByID(ctx context.Context, id uuid.UUID) error
	DeleteUserByEmail(ctx context.Context, email string) error

	GetSession(sid string) ([]byte, error)
	SaveSession(sess []byte) (string, error)

	BlacklistToken(ctx context.Context, id string) error
	VerifyToken(ctx context.Context, id string) (string, error)
}

type store struct {
	q  *sqlc.Queries
	sc *cache.Cache
	tc *cache.Cache
}

func New(conn sqlc.DBTX, sessionCache, tokenCache *cache.Cache) Repo {
	return &store{
		q:  sqlc.New(conn),
		sc: sessionCache,
		tc: tokenCache,
	}
}

func (s *store) CreateUser(ctx context.Context, u *auth.User) (*auth.User, error) {
	created, err := s.q.CreateUser(ctx, &sqlc.CreateUserParams{
		Username: u.Username,
		Email:    u.Email,
	})
	if err != nil {
		return nil, err
	}

	return &auth.User{
		ID:       created.ID,
		Username: created.Username,
		Email:    created.Email,
	}, err
}

func (s *store) GetUserByID(ctx context.Context, id uuid.UUID) (*auth.User, error) {
	found, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &auth.User{
		ID:       found.ID,
		Username: found.Username,
		Email:    found.Email,
	}, nil
}

func (s *store) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	found, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return &auth.User{
		ID:       found.ID,
		Username: found.Username,
		Email:    found.Email,
	}, nil
}

func (s *store) ListUsers(ctx context.Context) ([]auth.User, error) {
	res := make([]auth.User, 0)
	users, err := s.q.ListUsers(ctx, &sqlc.ListUsersParams{
		// TODO: Paginate results.
		Limit: 20,
	})
	if err != nil {
		return nil, fmt.Errorf("listUsers: %w", err)
	}

	for _, u := range users {
		res = append(res, auth.User{
			ID:       u.ID,
			Username: u.Username,
			Email:    u.Email,
		})
	}
	return res, nil
}

func (s *store) UpdateUserByID(ctx context.Context, id uuid.UUID, username string) (*auth.User, error) {
	updated, err := s.q.UpdateUserByID(ctx, &sqlc.UpdateUserByIDParams{
		ID:       id,
		Username: username,
	})
	if err != nil {
		return nil, err
	}

	return &auth.User{
		ID:       updated.ID,
		Username: updated.Username,
		Email:    updated.Email,
	}, nil
}

func (s *store) UpdateUserByEmail(ctx context.Context, email string, username string) (*auth.User, error) {
	updated, err := s.q.UpdateUserByEmail(ctx, &sqlc.UpdateUserByEmailParams{
		Email:    email,
		Username: username,
	})
	if err != nil {
		return nil, err
	}

	return &auth.User{
		ID:       updated.ID,
		Username: updated.Username,
		Email:    updated.Email,
	}, nil
}

func (s *store) DeleteUserByID(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteUserByID(ctx, id)
}

func (s *store) DeleteUserByEmail(ctx context.Context, email string) error {
	return s.q.DeleteUserByEmail(ctx, email)
}

func (s *store) GetSession(sid string) ([]byte, error) { return s.sc.Get(sid) }

func (s *store) SaveSession(sess []byte) (string, error) {
	sid := uuid.NewString()
	if err := s.sc.Set(sid, sess); err != nil {
		return "", err
	}

	return sid, nil
}

func (s *store) BlacklistToken(ctx context.Context, id string) error { return s.tc.Set(id, nil) }

func (s *store) VerifyToken(ctx context.Context, id string) (string, error) {
	_, err := s.tc.Get(id)
	if err != nil {
		if errors.Is(err, nats.ErrNoKeysFound) {
			return events.TokenAccepted, nil
		}

		return events.TokenRejected, err
	}

	return events.TokenRejected, nil
}
