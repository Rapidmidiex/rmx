package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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

func (s Store) CreateJam(ctx context.Context, j jam.Jam) (jam.Jam, error) {
	created, err := s.Q.CreateJam(ctx, &db.CreateJamParams{
		Name:     j.Name,
		Bpm:      int32(j.BPM),
		Capacity: int32(j.Capacity),
	})

	return jam.Jam{
		ID:       created.ID,
		Name:     created.Name,
		BPM:      uint(created.Bpm),
		Capacity: uint(created.Capacity),
	}, err
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
			ID:       j.ID,
			Name:     j.Name,
			BPM:      uint(j.Bpm),
			Capacity: uint(j.Capacity),
		})
	}
	return res, nil
}

func (s Store) GetJamByID(ctx context.Context, id uuid.UUID) (jam.Jam, error) {
	found, err := s.Q.GetJam(ctx, id)
	if err != nil {
		return jam.Jam{}, err
	}

	return jam.Jam{
		ID:       found.ID,
		Name:     found.Name,
		BPM:      uint(found.Bpm),
		Capacity: uint(found.Capacity),
	}, nil
}
