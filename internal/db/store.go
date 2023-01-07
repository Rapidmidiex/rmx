package db

import (
	"context"
	"fmt"

	"github.com/hyphengolang/prelude/types/suid"
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

// GetJams fetches all Jams from the store.
func (s Store) GetJams(ctx context.Context) ([]jam.Jam, error) {
	res := make([]jam.Jam, 0)
	jams, err := s.Q.ListJams(ctx, &db.ListJamsParams{
		// TODO: Paginate results.
		Limit: 50,
	})
	if err != nil {
		return res, fmt.Errorf("listJams: %w", err)
	}

	for _, j := range jams {
		res = append(res, jam.Jam{
			ID: suid.UUID{
				UUID: j.ID,
			},
			Name:     j.Name,
			BPM:      uint(j.Bpm),
			Capacity: uint(j.Capacity),
		})
	}
	return res, nil
}
