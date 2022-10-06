package store

import "context"

type Store struct{}

func New(ctx context.Context) *Store {
	db := &Store{}
	return db
}
