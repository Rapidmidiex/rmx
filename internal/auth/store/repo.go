package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hyphengolang/prelude/types/suid"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/rapidmidiex/rmx/internal/auth/store/sqlc"
	"github.com/rapidmidiex/rmx/internal/cache"
)

type Repo interface {
	CreateUser(context.Context, auth.User) (*auth.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*auth.User, error)
	GetUserByEmail(ctx context.Context, email string) (*auth.User, error)
	ListUsers(ctx context.Context) ([]auth.User, error)
	UpdateUserByID(ctx context.Context, id uuid.UUID, username string) (*auth.User, error)
	UpdateUserByEmail(ctx context.Context, email, username string) (*auth.User, error)
	DeleteUserByID(ctx context.Context, id uuid.UUID) error
	DeleteUserByEmail(ctx context.Context, email string) error

	CreateSession(ctx context.Context, email, issuer string, tokens auth.Session) (string, error)
	GetSession(cid string) (*auth.Session, error)
	GetAllSessions(ctx context.Context, email string) ([]auth.Session, error)
	DeleteSession(ctx context.Context, cid string) error
	DeleteAllSessions(ctx context.Context, email string) error
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

func (s *store) CreateSession(ctx context.Context, email, issuer string, tokens auth.Session) (string, error) {
	params := &sqlc.CreateSessionParams{
		Email:  email,
		Issuer: issuer,
	}

	created, err := s.q.CreateSession(ctx, params)
	if err != nil {
		return "", nil
	}

	bs, err := json.Marshal(tokens)
	if err != nil {
		return "", nil
	}

	return created.ID.String(), s.sc.Set(created.ID.String(), bs)
}

func (s *store) GetSession(cid string) (*auth.Session, error) {
	sb, err := s.sc.Get(cid)
	if err != nil {
		return nil, err
	}

	var session auth.Session
	if err = json.Unmarshal(sb, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *store) GetAllSessions(ctx context.Context, email string) ([]auth.Session, error) {
	sIDs, err := s.q.GetSessionsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	var sessions []auth.Session
	for _, ses := range sIDs {
		bs, err := s.sc.Get(ses.ID.String())
		if err != nil {
			return nil, err
		}

		var session auth.Session
		if err := json.Unmarshal(bs, &session); err != nil {
			return nil, err
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (s *store) DeleteSession(ctx context.Context, cid string) error {
	cidParsed, err := suid.ParseString(cid)
	if err != nil {
		return err
	}

	if err := s.q.DeleteSessionByID(ctx, cidParsed.UUID); err != nil {
		return err
	}

	return s.sc.Delete(cid)
}

func (s *store) DeleteAllSessions(ctx context.Context, email string) error {
	sIDs, err := s.q.GetSessionsByEmail(ctx, email)
	if err != nil {
		return err
	}

	if err := s.q.DeleteSessionsByEmail(ctx, email); err != nil {
		return err
	}

	for _, ses := range sIDs {
		if err := s.sc.Delete(ses.ID.String()); err != nil {
			return err
		}
	}

	return nil
}
