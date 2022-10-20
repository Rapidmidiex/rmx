package store

import (
	"context"

	"github.com/rog-golang-buddies/rmx/internal"
)

type Store struct {
	ctx context.Context

	tc internal.TokenClient
	ur internal.UserRepo
}

func (s *Store) UserRepo() internal.UserRepo {
	return s.ur
}

func (s *Store) TokenClient() internal.TokenClient {
	return s.tc
}

// FIXME this needs to be fleshed out properly
func New(ctx context.Context, connString string) *Store {
	s := &Store{ctx: ctx}
	return s
}

func (s *Store) Context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}
