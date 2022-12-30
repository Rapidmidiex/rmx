// TODO - use the standard sql package instead of pgx
package postgres_test

import (
	"context"
	"testing"
	"time"

	_ "embed"

	"github.com/rog-golang-buddies/rmx/docker/container"
	"github.com/rog-golang-buddies/rmx/docker/options"
	"github.com/rog-golang-buddies/rmx/domain/user"
	"github.com/rog-golang-buddies/rmx/domain/user/postgres"

	"github.com/docker/go-connections/nat"
	"github.com/hyphengolang/prelude/testing/is"
	"github.com/hyphengolang/prelude/types/email"
	"github.com/hyphengolang/prelude/types/password"
	"github.com/hyphengolang/prelude/types/suid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	dbName = "user"
	pgUser = "postgres"
	pgPass = "postgres"
)

//go:embed init/init.sql
var migration string

func TestPostgresContainer(t *testing.T) {
	is := is.New(t)

	ctx := context.Background()

	port, err := nat.NewPort("tcp", "5432")
	is.NoErr(err) // create new instance of the port

	container, err := container.NewPostgres(ctx,
		options.WithPort(port.Port()),
		options.WithInitialDatabase(pgUser, pgPass, dbName),
		options.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	is.NoErr(err) // create new instance of the postgres container

	t.Cleanup(func() {
		err := container.Terminate(ctx)
		is.NoErr(err) // terminate the container
	})

	connStr, err := container.ParseConnStr(ctx, port, pgUser, pgPass, dbName)
	is.NoErr(err) // parse the connection string

	conn, err := pgxpool.New(ctx, connStr)
	is.NoErr(err) // create new instance of the connection pool
	defer conn.Close()

	t.Run("migration", func(t *testing.T) {
		_, err = conn.Exec(ctx, migration)
		is.NoErr(err) // create new table
	})

	db := postgres.New(conn)

	t.Run("write two users to table", func(t *testing.T) {
		fizz := user.User{
			ID:       suid.NewUUID(),
			Email:    email.MustParse("fizz@mail.com"),
			Username: "fizz",
			Password: password.MustParse("fizz_pw_1").MustHash(),
		}

		err := db.Write(ctx, &fizz)
		is.NoErr(err) // insert new user "fizz"

		buzz := user.User{
			ID:       suid.NewUUID(),
			Email:    email.MustParse("buzz@mail.com"),
			Username: "buzz",
			Password: password.MustParse("buzz_pw_1").MustHash(),
		}

		err = db.Write(ctx, &buzz)
		is.NoErr(err) // insert new user "buzz"

		// us, err := db.SelectMany(ctx)
		// is.NoErr(err)        // select all users
		// is.Equal(len(us), 2) // should be a length of 2
	})

	t.Run("read two users from table", func(t *testing.T) {
		us, err := db.ReadAll(ctx)
		is.NoErr(err)        // select all users
		is.Equal(len(us), 2) // should be a length of 2
	})

	t.Run("reject user with duplicate email/username", func(t *testing.T) {
		fizz := user.User{
			ID:       suid.NewUUID(),
			Email:    email.MustParse("fuzz@mail.com"),
			Username: "fizz",
			Password: password.MustParse("fuzz_pw_1").MustHash(),
		}

		err := db.Write(ctx, &fizz)
		is.True(err != nil) // duplicate user with username "fizz"
	})

	t.Run("select a user from the database using email/username", func(t *testing.T) {
		u, err := db.Read(ctx, "fizz")
		is.NoErr(err)                             // select user where username = "fizz"
		is.NoErr(u.Password.Compare("fizz_pw_1")) // valid login

		_, err = db.Read(ctx, email.MustParse("buzz@mail.com"))
		is.NoErr(err) // select user where email = "buzz@mail.com"

	})

	t.Run("delete by username from database, return 1 user in database", func(t *testing.T) {
		err := db.Remove(ctx, "fizz")
		is.NoErr(err) // delete user where username == "fizz"

		us, err := db.ReadAll(ctx)
		is.NoErr(err)        // select all users
		is.Equal(len(us), 1) // should be a length of 1
	})
}

func TestPSQL(t *testing.T) {
	t.Skip()
	// t.Parallel()
	// is, ctx := is.New(t), context.Background()
	// t.Cleanup(func() { pool.Close() })

	// t.Run(`select * from "user"`, func(t *testing.T) {
	// 	_, err := db.SelectMany(ctx)
	// 	is.NoErr(err) // error reading from database
	// })

	// t.Run(`insert two new users`, func(t *testing.T) {

	// 	fizz := internal.User{
	// 		ID:       suid.NewUUID(),
	// 		Email:    email.MustParse("fizz@mail.com"),
	// 		Username: "fizz",
	// 		Password: password.MustParse("fizz_pw_1").MustHash(),
	// 	}

	// 	err := db.Insert(ctx, &fizz)
	// 	is.NoErr(err) // insert new user "fizz"

	// 	buzz := internal.User{
	// 		ID:       suid.NewUUID(),
	// 		Email:    email.MustParse("buzz@mail.com"),
	// 		Username: "buzz",
	// 		Password: password.MustParse("buzz_pw_1").MustHash(),
	// 	}

	// 	err = db.Insert(ctx, &buzz)
	// 	is.NoErr(err) // insert new user "buzz"

	// 	us, err := db.SelectMany(ctx)
	// 	is.NoErr(err)        // select all users
	// 	is.Equal(len(us), 2) // should be a length of 2
	// })

	// t.Run("reject user with duplicate email/username", func(t *testing.T) {
	// 	fizz := internal.User{
	// 		ID:       suid.NewUUID(),
	// 		Email:    email.MustParse("fuzz@mail.com"),
	// 		Username: "fizz",
	// 		Password: password.MustParse("fuzz_pw_1").MustHash(),
	// 	}

	// 	err := db.Insert(ctx, &fizz)
	// 	is.True(err != nil) // duplicate user with username "fizz"
	// })

	// t.Run("select a user from the database using email/username", func(t *testing.T) {
	// 	u, err := db.Select(ctx, "fizz")
	// 	is.NoErr(err)                             // select user where username = "fizz"
	// 	is.NoErr(u.Password.Compare("fizz_pw_1")) // valid login

	// 	_, err = db.Select(ctx, email.MustParse("buzz@mail.com"))
	// 	is.NoErr(err) // select user where email = "buzz@mail.com"
	// })

	// t.Run("delete by username from database, return 1 user in database", func(t *testing.T) {
	// 	err := db.Delete(ctx, "fizz")
	// 	is.NoErr(err) // delete user where username == "fizz"

	// 	us, err := db.SelectMany(ctx)
	// 	is.NoErr(err)        // select all users
	// 	is.Equal(len(us), 1) // should be a length of 1
	// })
}
