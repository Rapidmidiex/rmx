// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.17.0
// source: jam.sql

package sqlc

import (
	"context"

	"github.com/google/uuid"
)

const createJam = `-- name: CreateJam :one
INSERT INTO jam (name, bpm, capacity)
    VALUES ($1, $2, $3)
RETURNING
    id, name, bpm, capacity, created_at
`

type CreateJamParams struct {
	Name     string `json:"name"`
	Bpm      int32  `json:"bpm"`
	Capacity int32  `json:"capacity"`
}

func (q *Queries) CreateJam(ctx context.Context, arg *CreateJamParams) (Jam, error) {
	row := q.db.QueryRowContext(ctx, createJam, arg.Name, arg.Bpm, arg.Capacity)
	var i Jam
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Bpm,
		&i.Capacity,
		&i.CreatedAt,
	)
	return i, err
}

const deleteJam = `-- name: DeleteJam :exec
DELETE FROM jam
WHERE id = $1
`

func (q *Queries) DeleteJam(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteJam, id)
	return err
}

const getJam = `-- name: GetJam :one
SELECT
    id, name, bpm, capacity, created_at
FROM
    jam
WHERE
    id = $1
LIMIT 1
`

func (q *Queries) GetJam(ctx context.Context, id uuid.UUID) (Jam, error) {
	row := q.db.QueryRowContext(ctx, getJam, id)
	var i Jam
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Bpm,
		&i.Capacity,
		&i.CreatedAt,
	)
	return i, err
}

const listJams = `-- name: ListJams :many
SELECT
    id, name, bpm, capacity, created_at
FROM
    jam
ORDER BY
    "name"
LIMIT $1 OFFSET $2
`

type ListJamsParams struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

func (q *Queries) ListJams(ctx context.Context, arg *ListJamsParams) ([]Jam, error) {
	rows, err := q.db.QueryContext(ctx, listJams, arg.Limit, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Jam{}
	for rows.Next() {
		var i Jam
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Bpm,
			&i.Capacity,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateJam = `-- name: UpdateJam :one
UPDATE
    jam
SET
    bpm = $2
WHERE
    id = $1
RETURNING
    id, name, bpm, capacity, created_at
`

type UpdateJamParams struct {
	ID  uuid.UUID `json:"id"`
	Bpm int32     `json:"bpm"`
}

func (q *Queries) UpdateJam(ctx context.Context, arg *UpdateJamParams) (Jam, error) {
	row := q.db.QueryRowContext(ctx, updateJam, arg.ID, arg.Bpm)
	var i Jam
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Bpm,
		&i.Capacity,
		&i.CreatedAt,
	)
	return i, err
}
