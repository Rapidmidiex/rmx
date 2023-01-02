// TODO - use the standard sql package instead of pgx
package user

import (
	"context"
	"testing"

	"github.com/hyphengolang/prelude/testing/is"
	"github.com/hyphengolang/prelude/types/email"
	"github.com/hyphengolang/prelude/types/password"
	"github.com/hyphengolang/prelude/types/suid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rapidmidiex/rmx/internal"
)

/*
https://www.covermymeds.com/main/insights/articles/on-update-timestamps-mysql-vs-postgres/
*/
var db Repo

var pool *pgxpool.Pool

func TestPSQL(t *testing.T) {
	t.Skip()
	t.Parallel()

	is, ctx := is.New(t), context.Background()

	t.Cleanup(func() { pool.Close() })

	t.Run(`select * from "user"`, func(t *testing.T) {
		_, err := db.SelectMany(ctx)
		is.NoErr(err) // error reading from database
	})

	t.Run(`insert two new users`, func(t *testing.T) {

		fizz := internal.User{
			ID:       suid.NewUUID(),
			Email:    email.MustParse("fizz@mail.com"),
			Username: "fizz",
			Password: password.MustParse("fizz_pw_1").MustHash(),
		}

		err := db.Insert(ctx, &fizz)
		is.NoErr(err) // insert new user "fizz"

		buzz := internal.User{
			ID:       suid.NewUUID(),
			Email:    email.MustParse("buzz@mail.com"),
			Username: "buzz",
			Password: password.MustParse("buzz_pw_1").MustHash(),
		}

		err = db.Insert(ctx, &buzz)
		is.NoErr(err) // insert new user "buzz"

		us, err := db.SelectMany(ctx)
		is.NoErr(err)        // select all users
		is.Equal(len(us), 2) // should be a length of 2
	})

	t.Run("reject user with duplicate email/username", func(t *testing.T) {
		fizz := internal.User{
			ID:       suid.NewUUID(),
			Email:    email.MustParse("fuzz@mail.com"),
			Username: "fizz",
			Password: password.MustParse("fuzz_pw_1").MustHash(),
		}

		err := db.Insert(ctx, &fizz)
		is.True(err != nil) // duplicate user with username "fizz"
	})

	t.Run("select a user from the database using email/username", func(t *testing.T) {
		u, err := db.Select(ctx, "fizz")
		is.NoErr(err)                             // select user where username = "fizz"
		is.NoErr(u.Password.Compare("fizz_pw_1")) // valid login

		_, err = db.Select(ctx, email.MustParse("buzz@mail.com"))
		is.NoErr(err) // select user where email = "buzz@mail.com"
	})

	t.Run("delete by username from database, return 1 user in database", func(t *testing.T) {
		err := db.Delete(ctx, "fizz")
		is.NoErr(err) // delete user where username == "fizz"

		us, err := db.SelectMany(ctx)
		is.NoErr(err)        // select all users
		is.Equal(len(us), 1) // should be a length of 1
	})
}
