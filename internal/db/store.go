package db

import (
	"context"

	db "github.com/rapidmidiex/rmx/internal/db/sqlc"
	"github.com/rapidmidiex/rmx/internal/jam"
)

type (
	Store struct {
		Q *db.Queries
	}
)

func NewStore(conn db.DBTX) *Store {
	return &Store{Q: db.New(conn)}
}

func (s Store) CreateJam(ctx context.Context, j jam.Jam) error {
	_, err := s.Q.CreateJam(ctx, &db.CreateJamParams{
		Name:     j.Name,
		Bpm:      int32(j.BPM),
		Capacity: int32(j.Capacity),
	})

	return err
}
