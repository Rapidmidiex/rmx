// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.18.0
// source: sessions.sql

package sqlc

import (
	"context"

	"github.com/google/uuid"
)

const createSession = `-- name: CreateSession :one
INSERT INTO sessions (email, issuer)
    VALUES ($1, $2)
RETURNING
    id, email, issuer, created_at
`

type CreateSessionParams struct {
	Email  string `json:"email"`
	Issuer string `json:"issuer"`
}

func (q *Queries) CreateSession(ctx context.Context, arg *CreateSessionParams) (Session, error) {
	row := q.db.QueryRowContext(ctx, createSession, arg.Email, arg.Issuer)
	var i Session
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Issuer,
		&i.CreatedAt,
	)
	return i, err
}

const deleteSessionByID = `-- name: DeleteSessionByID :exec
DELETE FROM sessions
WHERE id = $1
`

func (q *Queries) DeleteSessionByID(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, deleteSessionByID, id)
	return err
}

const deleteSessionsByEmail = `-- name: DeleteSessionsByEmail :exec
DELETE FROM sessions
WHERE email = $1
`

func (q *Queries) DeleteSessionsByEmail(ctx context.Context, email string) error {
	_, err := q.db.ExecContext(ctx, deleteSessionsByEmail, email)
	return err
}

const getSessionByID = `-- name: GetSessionByID :one
SELECT
    id, email, issuer, created_at
FROM
    sessions
WHERE
    id = $1
LIMIT 1
`

func (q *Queries) GetSessionByID(ctx context.Context, id uuid.UUID) (Session, error) {
	row := q.db.QueryRowContext(ctx, getSessionByID, id)
	var i Session
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Issuer,
		&i.CreatedAt,
	)
	return i, err
}

const getSessionsByEmail = `-- name: GetSessionsByEmail :many
SELECT
    id, email, issuer, created_at
FROM
    sessions
WHERE
    email = $1
LIMIT 1
`

func (q *Queries) GetSessionsByEmail(ctx context.Context, email string) ([]Session, error) {
	rows, err := q.db.QueryContext(ctx, getSessionsByEmail, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Session{}
	for rows.Next() {
		var i Session
		if err := rows.Scan(
			&i.ID,
			&i.Email,
			&i.Issuer,
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