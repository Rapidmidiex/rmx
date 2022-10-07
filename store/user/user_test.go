package user

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/is"
	"github.com/rog-golang-buddies/rmx/internal/suid"
)

var db internal.UserRepo

func init() {
	ctx := context.WithValue(context.Background(), internal.DevMode, true)
	c, err := pgx.Connect(context.Background(), `postgres://postgres:postgrespw@localhost:49153/postgres`)
	if err != nil {
		panic(err)
	}

	db = NewRepo(ctx, c)
}

func TestSQLRepo(t *testing.T) {
	t.Parallel()

	is, ctx := is.New(t), context.Background()

	t.Run("insert two users to database", func(t *testing.T) {
		fizz := internal.User{
			ID:       suid.NewUUID(),
			Email:    "fizz@mail.com",
			Username: "fizz",
			Password: internal.Password("fizz_pw_1").MustHash(),
		}

		err := db.Insert(ctx, &fizz)
		is.NoErr(err) // insert new user "fizz"

		buzz := internal.User{
			ID:       suid.NewUUID(),
			Email:    "buzz@mail.com",
			Username: "buzz",
			Password: internal.Password("buzz_pw_1").MustHash(),
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
			Email:    "fuzz@mail.com",
			Username: "fizz",
			Password: internal.Password("fuzz_pw_1").MustHash(),
		}

		err := db.Insert(ctx, &fizz)
		is.True(err != nil) // duplicate user with username "fizz"
	})

	t.Run("select a user from the database using email/username", func(t *testing.T) {
		u, err := db.Select(ctx, "fizz")
		is.NoErr(err)                             // select user where username = "fizz"
		is.NoErr(u.Password.Compare("fizz_pw_1")) // valid login

		_, err = db.Select(ctx, internal.Email("buzz@mail.com"))
		is.NoErr(err) // select user where email = "buzz@mail.com"
	})

	t.Run("delete user from database, return 1 user in database", func(t *testing.T) {
		err := db.Delete(ctx, "fizz")
		is.NoErr(err) // delete user where username == "fizz"

		us, err := db.SelectMany(ctx)
		is.NoErr(err)        // select all users
		is.Equal(len(us), 1) // should be a length of 1
	})
}
