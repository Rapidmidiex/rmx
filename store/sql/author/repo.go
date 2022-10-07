package author

import (
	"context"
	"database/sql"
)

const (
	psql  = "postgres"
	mysql = "mysql"
)

type AuthorRepo interface {
	ListAuthors(ctx context.Context) ([]Author, error)
	InsertAuthor(ctx context.Context, a *InternalAuthor) (Author, error)
}

type Repo struct {
	q *Queries
}

func NewRepo(ctx context.Context, connString string) AuthorRepo {
	db, err := sql.Open(psql, connString)
	if err != nil {
		panic(err)
	}

	// test_authors
	_, err = db.ExecContext(context.Background(), `
		CREATE TEMP TABLE authors (
			id bigserial PRIMARY KEY,
			name text NOT NULL,
			bio text
		)`)

	if err != nil {
		panic(err)
	}

	q := New(db)

	return &Repo{
		q,
	}
}

func (r *Repo) ListAuthors(ctx context.Context) ([]Author, error) {
	return r.q.ListAuthors(ctx)
}

func (r *Repo) InsertAuthor(ctx context.Context, a *InternalAuthor) (Author, error) {
	var valid bool
	if a.Bio != "" {
		valid = true
	}

	v := CreateAuthorParams{
		Name: a.Name,
		Bio:  sql.NullString{String: a.Bio, Valid: valid},
	}

	_, err := r.q.CreateAuthor(ctx, v)
	return Author{}, err
}

// this would be inside the internal package
type InternalAuthor struct {
	Name string
	Bio  string
}
