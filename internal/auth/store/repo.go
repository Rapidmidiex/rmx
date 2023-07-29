package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/rapidmidiex/oauth"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/rapidmidiex/rmx/internal/auth/store/sqlc"
	"github.com/rapidmidiex/rmx/internal/cache"
)

type Repo interface {
	CreateUser(ctx context.Context, u *sqlc.CreateUserParams) (*auth.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*auth.User, error)
	GetUserByUsername(ctx context.Context, username string) (*auth.User, error)
	GetUserByEmail(ctx context.Context, email string) (*auth.User, error)
	ListUsers(ctx context.Context, p *sqlc.ListUsersParams) ([]auth.User, error)
	UpdateUserByID(ctx context.Context, u *sqlc.UpdateUserByIDParams) (*auth.User, error)
	UpdateUserByUsername(ctx context.Context, u *sqlc.UpdateUserByUsernameParams) (*auth.User, error)
	UpdateUserByEmail(ctx context.Context, u *sqlc.UpdateUserByEmailParams) (*auth.User, error)
	DeleteUserByID(ctx context.Context, id uuid.UUID) error
	DeleteUserByUsername(ctx context.Context, username string) error
	DeleteUserByEmail(ctx context.Context, email string) error

	CreateConnection(ctx context.Context, c *sqlc.CreateConnectionParams) (*auth.Connection, error)
	GetConnection(ctx context.Context, providerID string) (*auth.Connection, error)
	ListUserConnections(ctx context.Context, p *sqlc.ListUserConnectionsParams) ([]auth.Connection, error)
	DeleteConnection(ctx context.Context, providerID string) error
	DeleteUserConnections(ctx context.Context, userID uuid.UUID) error

	GetSession(sid string) ([]byte, error)
	SaveSession(sess oauth.Session) (string, error)

	BlacklistSession(ctx context.Context, sid string) error
	VerifySession(ctx context.Context, sid string) (bool, error)
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

func (s *store) CreateUser(ctx context.Context, u *sqlc.CreateUserParams) (*auth.User, error) {
	created, err := s.q.CreateUser(ctx, u)
	if err != nil {
		return nil, err
	}

	return &auth.User{
		ID:       created.ID,
		Username: created.Username,
		Email:    created.Email,
		IsAdmin:  created.IsAdmin,
		Picture:  created.Picture,
		Blocked:  created.Blocked,
	}, nil
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
		IsAdmin:  found.IsAdmin,
		Picture:  found.Picture,
		Blocked:  found.Blocked,
	}, nil
}

func (s *store) GetUserByUsername(ctx context.Context, username string) (*auth.User, error) {
	found, err := s.q.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	return &auth.User{
		ID:       found.ID,
		Username: found.Username,
		Email:    found.Email,
		IsAdmin:  found.IsAdmin,
		Picture:  found.Picture,
		Blocked:  found.Blocked,
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
		IsAdmin:  found.IsAdmin,
		Picture:  found.Picture,
		Blocked:  found.Blocked,
	}, nil
}

func (s *store) ListUsers(ctx context.Context, p *sqlc.ListUsersParams) ([]auth.User, error) {
	res := make([]auth.User, 0)
	users, err := s.q.ListUsers(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("listUsers: %w", err)
	}

	for _, u := range users {
		res = append(res, auth.User{
			ID:       u.ID,
			Username: u.Username,
			Email:    u.Email,
			IsAdmin:  u.IsAdmin,
			Picture:  u.Picture,
			Blocked:  u.Blocked,
		})
	}
	return res, nil
}

func (s *store) UpdateUserByID(ctx context.Context, u *sqlc.UpdateUserByIDParams) (*auth.User, error) {
	updated, err := s.q.UpdateUserByID(ctx, u)
	if err != nil {
		return nil, err
	}

	return &auth.User{
		ID:       updated.ID,
		Username: updated.Username,
		Email:    updated.Email,
		IsAdmin:  updated.IsAdmin,
		Picture:  updated.Picture,
		Blocked:  updated.Blocked,
	}, nil
}

func (s *store) UpdateUserByUsername(ctx context.Context, u *sqlc.UpdateUserByUsernameParams) (*auth.User, error) {
	updated, err := s.q.UpdateUserByUsername(ctx, u)
	if err != nil {
		return nil, err
	}

	return &auth.User{
		ID:       updated.ID,
		Username: updated.Username,
		Email:    updated.Email,
		IsAdmin:  updated.IsAdmin,
		Picture:  updated.Picture,
		Blocked:  updated.Blocked,
	}, nil
}

func (s *store) UpdateUserByEmail(ctx context.Context, u *sqlc.UpdateUserByEmailParams) (*auth.User, error) {
	updated, err := s.q.UpdateUserByEmail(ctx, u)
	if err != nil {
		return nil, err
	}

	return &auth.User{
		ID:       updated.ID,
		Username: updated.Username,
		Email:    updated.Email,
		IsAdmin:  updated.IsAdmin,
		Picture:  updated.Picture,
		Blocked:  updated.Blocked,
	}, nil
}

func (s *store) DeleteUserByID(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteUserByID(ctx, id)
}

func (s *store) DeleteUserByUsername(ctx context.Context, username string) error {
	return s.q.DeleteUserByUsername(ctx, username)
}

func (s *store) DeleteUserByEmail(ctx context.Context, email string) error {
	return s.q.DeleteUserByEmail(ctx, email)
}

func (s *store) CreateConnection(ctx context.Context, c *sqlc.CreateConnectionParams) (*auth.Connection, error) {
	created, err := s.q.CreateConnection(ctx, c)
	if err != nil {
		return nil, err
	}

	return &auth.Connection{
		ProviderID: created.ProviderID,
		UserID:     created.UserID,
	}, nil
}

func (s *store) GetConnection(ctx context.Context, providerID string) (*auth.Connection, error) {
	found, err := s.q.GetConnection(ctx, providerID)
	if err != nil {
		return nil, err
	}

	return &auth.Connection{
		ProviderID: found.ProviderID,
		UserID:     found.UserID,
	}, nil
}

func (s *store) ListUserConnections(ctx context.Context, p *sqlc.ListUserConnectionsParams) ([]auth.Connection, error) {
	res := make([]auth.Connection, 0)
	connections, err := s.q.ListUserConnections(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("listUserConnections: %w", err)
	}

	for _, c := range connections {
		res = append(res, auth.Connection{
			ProviderID: c.ProviderID,
			UserID:     c.UserID,
		})
	}
	return res, nil
}

func (s *store) DeleteConnection(ctx context.Context, providerID string) error {
	return s.q.DeleteConnection(ctx, providerID)
}

func (s *store) DeleteUserConnections(ctx context.Context, userID uuid.UUID) error {
	return s.q.DeleteUserConnections(ctx, userID)
}

func (s *store) GetSession(sid string) ([]byte, error) {
	return s.sc.Get(sid)
}

func (s *store) SaveSession(sess oauth.Session) (string, error) {
	str, err := sess.Marshal()
	if err != nil {
		return "", err
	}

	sid := uuid.NewString()
	if err := s.sc.Set(sid, []byte(str)); err != nil {
		return "", err
	}

	return sid, nil
}

func (s *store) BlacklistSession(ctx context.Context, sid string) error { return s.tc.Set(sid, nil) }

// VerifySession returns true if the session id is not found and is not blacklisted
func (s *store) VerifySession(ctx context.Context, sid string) (bool, error) {
	_, err := s.tc.Get(sid)
	if err != nil {
		if errors.Is(err, nats.ErrKeyNotFound) {
			return true, nil
		}

		return false, err
	}

	return false, nil
}
