package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/rapidmidiex/rmx/internal/auth/postgres/sqlc"
	"github.com/rapidmidiex/rmx/internal/cache"
)

type Repo interface {
	CreateUser(context.Context, auth.User) (*auth.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*auth.User, error)
	GetUserByEmail(ctx context.Context, email string) (*auth.User, error)
	ListUsers(context.Context) ([]auth.User, error)
	UpdateUserByID(ctx context.Context, id uuid.UUID, username string) (*auth.User, error)
	UpdateUserByEmail(ctx context.Context, email, username string) (*auth.User, error)
	DeleteUserByID(ctx context.Context, id uuid.UUID) error
	DeleteUserByEmail(ctx context.Context, email string) error

	CreateSession(email, issuer, cid string, tokens auth.Tokens) error
	GetSession(email string, cid string) (*auth.Session, error)
	GetAllSessions(email string) ([]auth.Session, error)
	DeleteSession(email string, cid string) error
	DeleteAllSessions(email string) error
}

type store struct {
	q *sqlc.Queries
	c *cache.Cache
}

func New(conn sqlc.DBTX, cache *cache.Cache) Repo {
	return &store{
		q: sqlc.New(conn),
		c: cache,
	}
}

func (s *store) CreateUser(ctx context.Context, u auth.User) (*auth.User, error) {
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

func (s *store) CreateSession(email, issuer, cid string, tokens auth.Tokens) error {
	ssb, err := s.c.Get(email)
	if err != nil {
		return err
	}

	var sessions []auth.Session
	if err = json.Unmarshal(ssb, &sessions); err != nil {
		return err
	}

	newSession := auth.Session{
		ClientID: cid,
		Issuer:   issuer,
		Tokens:   tokens,
	}
	sessions = append(sessions, newSession)

	bs, err := json.Marshal(sessions)
	if err != nil {
		return err
	}

	return s.c.Set(email, bs)
}

func (s *store) GetSession(email string, cid string) (*auth.Session, error) {
	ssb, err := s.c.Get(email)
	if err != nil {
		return nil, err
	}

	var sessions []auth.Session
	if err = json.Unmarshal(ssb, &sessions); err != nil {
		return nil, err
	}

	for _, session := range sessions {
		if session.ClientID == cid {
			return &session, nil
		}
	}

	return nil, errors.New("invalid cid")
}

func (s *store) GetAllSessions(email string) ([]auth.Session, error) {
	sbs, err := s.c.Get(email)
	if err != nil {
		return nil, err
	}

	var sessions []auth.Session
	if err = json.Unmarshal(sbs, &sessions); err != nil {
		return nil, err
	}

	return sessions, nil
}

func (s *store) DeleteSession(email string, cid string) error {
	ssb, err := s.c.Get(email)
	if err != nil {
		return err
	}

	var sessions []auth.Session
	if err = json.Unmarshal(ssb, &sessions); err != nil {
		return err
	}

	for i, session := range sessions {
		if session.ClientID == cid {
			sessions = append(sessions[:i], sessions[i+1:]...) // removes only the matching session
		}
	}

	bs, err := json.Marshal(sessions)
	if err != nil {
		return err
	}

	return s.c.Set(email, bs)
}

func (s *store) DeleteAllSessions(email string) error {
	return s.c.Delete(email)
}
