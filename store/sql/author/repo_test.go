package author

import (
	"context"
	"testing"

	"github.com/rog-golang-buddies/rmx/internal/is"

	_ "github.com/lib/pq"
)

var r AuthorRepo

func init() {
	r = NewRepo(context.Background(), `postgres://postgres:postgrespw@localhost:49153?sslmode=disable`)
}

func TestRepo(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	t.Run("list all users", func(t *testing.T) {
		authors, err := r.ListAuthors(context.Background())
		is.NoErr(err)             // query all authors
		is.Equal(len(authors), 0) // no authors in database
	})

	t.Run("insert a new author", func(t *testing.T) {
		payload := &InternalAuthor{
			Name: "Brian Kernighan",
			Bio:  "Co-author of The C Programming Language and The Go Programming Language",
		}

		_, err := r.InsertAuthor(context.Background(), payload)
		is.NoErr(err) // inserting author
	})
}
