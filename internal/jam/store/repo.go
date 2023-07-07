package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rapidmidiex/rmx/internal/jam"
	"github.com/rapidmidiex/rmx/internal/jam/store/sqlc"
)

type Repo interface {
	CreateJam(context.Context, jam.Jam) (jam.Jam, error)
	GetJams(context.Context) ([]jam.Jam, error)
	GetJamByID(ctx context.Context, id uuid.UUID) (jam.Jam, error)
}

type store struct {
	q *sqlc.Queries
}

func New(conn sqlc.DBTX) Repo {
	return &store{q: sqlc.New(conn)}
}

func (s *store) GetJams(ctx context.Context) ([]jam.Jam, error) {
	res := make([]jam.Jam, 0)
	jams, err := s.q.ListJams(ctx, &sqlc.ListJamsParams{
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

func (s *store) GetJamByID(ctx context.Context, id uuid.UUID) (jam.Jam, error) {
	found, err := s.q.GetJam(ctx, id)
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

func (s *store) CreateJam(ctx context.Context, j jam.Jam) (jam.Jam, error) {
	created, err := s.q.CreateJam(ctx, &sqlc.CreateJamParams{
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
