package store

import (
	"context"

	"github.com/rog-golang-buddies/rmx/internal"
)

type Store struct {
	tc internal.TokenClient
	ur internal.UserRepo
}

func (s *Store) UserRepo() internal.UserRepo {
	return s.ur
}

func (s *Store) TokenClient() internal.TokenClient {
	return s.tc
}

func New(ctx context.Context, connString string) *Store {
	s := &Store{}
	return s
}
