package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hyphengolang/prelude/types/suid"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/rapidmidiex/rmx/internal/auth/postgres/sqlc"
	"github.com/redis/go-redis/v9"
)

type Repo interface {
	CreateUser(context.Context, auth.User) (auth.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (auth.User, error)
	GetUserByEmail(ctx context.Context, email string) (auth.User, error)
	ListUsers(context.Context) ([]auth.User, error)
	UpdateUserByID(ctx context.Context, id uuid.UUID, username string) (auth.User, error)
	UpdateUserByEmail(ctx context.Context, email, username string) (auth.User, error)
	DeleteUserByID(ctx context.Context, id uuid.UUID) error
	DeleteUserByEmail(ctx context.Context, email string) error

	CreateSession(ctx context.Context, email, accessToken, refreshToken string) error
	GetSession(ctx context.Context, email, id string) (*auth.Session, error)
	// only removes one user session
	DeleteSession(ctx context.Context, id uuid.UUID) error
	// removes all sessions owned by a user
	DeleteAllSessions(ctx context.Context, email string) error
}

type store struct {
	q  *sqlc.Queries
	rc *redis.Client
}

func New(conn sqlc.DBTX, rc *redis.Client) Repo {
	return &store{
		q:  sqlc.New(conn),
		rc: rc,
	}
}

func (s *store) CreateUser(ctx context.Context, u auth.User) (auth.User, error) {
	created, err := s.q.CreateUser(ctx, &sqlc.CreateUserParams{
		Username: u.Username,
		Email:    u.Email,
	})

	return auth.User{
		ID:       created.ID,
		Username: created.Username,
		Email:    created.Email,
	}, err
}

func (s *store) GetUserByID(ctx context.Context, id uuid.UUID) (auth.User, error) {
	found, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return auth.User{}, err
	}

	return auth.User{
		ID:       found.ID,
		Username: found.Username,
		Email:    found.Email,
	}, nil
}

func (s *store) GetUserByEmail(ctx context.Context, email string) (auth.User, error) {
	found, err := s.q.GetUserByEmail(ctx, email)
	if err != nil {
		return auth.User{}, err
	}

	return auth.User{
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
		return res, fmt.Errorf("listUsers: %w", err)
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

func (s *store) UpdateUserByID(ctx context.Context, id uuid.UUID, username string) (auth.User, error) {
	updated, err := s.q.UpdateUserByID(ctx, &sqlc.UpdateUserByIDParams{
		ID:       id,
		Username: username,
	})
	if err != nil {
		return auth.User{}, err
	}

	return auth.User{
		ID:       updated.ID,
		Username: updated.Username,
		Email:    updated.Email,
	}, nil
}

func (s *store) UpdateUserByEmail(ctx context.Context, email string, username string) (auth.User, error) {
	updated, err := s.q.UpdateUserByEmail(ctx, &sqlc.UpdateUserByEmailParams{
		Email:    email,
		Username: username,
	})
	if err != nil {
		return auth.User{}, err
	}

	return auth.User{
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

func (s *store) CreateSession(ctx context.Context, email string, session *auth.Session) error {
	// TODO: use HSET and use a real expiration
	bs, err := json.Marshal(session)
	if err != nil {
		return err
	}

	return s.rc.Set(ctx, fmt.Sprint(email, ":", suid.NewSUID().String()), bs, 0).Err()
}

func (s *store) GetSession(context.Context, email string, id suid.UUID) (*auth.Session, error) {
	sbs, err := s.rc.Get(ctx, fmt.Sprint(email, id.String())).Result()
	if err != nil {


		return err
	}


} 

func (s *store) DeleteSession(ctx context.Context, email string, id suid.UUID) error {
	return s.rc.Del(ctx, fmt.Sprint(email, ":", id.String())).Err()
}

func (s *store) DeleteAllSessions(ctx context.Context, email string) error {
	iter := s.rc.Scan(ctx, 0, fmt.Sprint(email, ":*"), 0).Iterator()
	for iter.Next(ctx) {
		err := s.rc.Del(ctx, iter.Val()).Err()
		if err != nil {
			return err
		}
	}

	if err := iter.Err(); err != nil {
		return err
	}

	return nil
}
