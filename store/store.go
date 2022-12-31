package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/store/auth"
	"github.com/rog-golang-buddies/rmx/store/user"
)

type Store struct {
	tc internal.TokenClient
	ur user.Repo
}

func (s *Store) UserRepo() user.Repo {
	if s.ur == nil {
		panic("user repo must not be nil")
	}
	return s.ur
}

func (s *Store) TokenClient() internal.TokenClient {
	if s.tc == nil {
		panic("token client must not be nil")
	}
	return s.tc
}

func New(ctx context.Context, connString string) (*Store, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}

	s := &Store{
		ur: user.NewRepo(ctx, pool),
		tc: auth.DefaultTokenClient,
	}

	return s, nil
}
