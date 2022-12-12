// TODO - use the standard sql package instead of pgx
package user

import (
	"context"
	"testing"

	psql "github.com/hyphengolang/prelude/sql/postgres"
	"github.com/hyphengolang/prelude/testing/is"
	"github.com/hyphengolang/prelude/types/email"
	"github.com/hyphengolang/prelude/types/password"
	"github.com/hyphengolang/prelude/types/suid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rog-golang-buddies/rmx/internal"
)

/*
https://www.covermymeds.com/main/insights/articles/on-update-timestamps-mysql-vs-postgres/
*/
var db Repo

const migration = `
begin;

create extension if not exists "uuid-ossp";
create extension if not exists "citext";

create temp table if not exists "user" (
	id uuid primary key default uuid_generate_v4(),
	username text unique not null check (username <> ''),
	email citext unique not null check (email ~ '^[a-zA-Z0-9.!#$%&’*+/=?^_\x60{|}~-]+@[a-zA-Z0-9-]+(?:\.[a-zA-Z0-9-]+)*$'),
	password citext not null check (password <> ''),
	created_at timestamp not null default now()
);

commit;
`

var pool *pgxpool.Pool
var err error

func init() {
	// create pool connection
	pool, err = pgxpool.New(context.Background(), `postgres://postgres:postgrespw@localhost:49153/testing`)
	if err != nil {
		panic(err)
	}

	// setup migration
	if err := psql.Exec(pool, migration); err != nil {
		panic(err)
	}

	db = NewRepo(context.Background(), pool)
}

func TestPSQL(t *testing.T) {
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
