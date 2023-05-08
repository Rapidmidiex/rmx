package rmx_test

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	authDB "github.com/rapidmidiex/rmx/internal/auth/postgres/sqlc"
	authSQLC "github.com/rapidmidiex/rmx/internal/auth/postgres/sqlc"
	jamDB "github.com/rapidmidiex/rmx/internal/jam/postgres/sqlc"
	jamSQLC "github.com/rapidmidiex/rmx/internal/jam/postgres/sqlc"
	"github.com/stretchr/testify/require"
)

//go:embed db/migration/*.sql
var migrations embed.FS

var pgDB *sql.DB
var jamTestQueries *jamSQLC.Queries
var authTestQueries *authSQLC.Queries
var dbName = "rmx-test"
var pgUser = "rmx-test"
var pgPass = "password123dev"
var databaseURL string

func TestMain(m *testing.M) {
	if os.Getenv("TEST_POSTGRES_URL") != "" {
		databaseURL = os.Getenv("TEST_POSTGRES_URL")
		var err error // Avoid shadowing for pgdb
		pgDB, err = sql.Open("postgres", databaseURL)
		if err != nil {
			log.Fatalf("cannot connect to db: %s\nconnection string: %s", err, databaseURL)
		}

		jamTestQueries = jamSQLC.New(pgDB)
		authTestQueries = authSQLC.New(pgDB)
		// Run tests
		code := m.Run()
		os.Exit(code)
	}

	// *** Dockertest (default) ***
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14.6-alpine",
		Env: []string{
			fmt.Sprintf("POSTGRES_PASSWORD=%s", pgPass),
			fmt.Sprintf("POSTGRES_USER=%s", pgUser),
			fmt.Sprintf("POSTGRES_DB=%s", dbName),
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseURL = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", pgUser, pgPass, hostAndPort, dbName)

	log.Println("Connecting to database on url: ", databaseURL)

	err = resource.Expire(120) // Tell docker to hard kill the container in 120 seconds
	if err != nil {
		log.Fatalf("could not set resource expiration time: %s", err)
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	pool.MaxWait = 120 * time.Second
	if err = pool.Retry(func() error {
		pgDB, err = sql.Open("postgres", databaseURL)
		if err != nil {
			return err
		}
		return pgDB.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	// Instantiate testQueries
	jamTestQueries = jamSQLC.New(pgDB)
	authTestQueries = authSQLC.New(pgDB)

	err = migrateUp()
	if err != nil {
		log.Fatalf("Could not run migrations: %s", err)
	}

	// Run tests
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func migrateUp() error {
	// Migrations
	// https://pkg.go.dev/github.com/golang-migrate/migrate/v4/source/iofs#example-package
	d, err := iofs.New(migrations, "db/migration")
	if err != nil {
		return fmt.Errorf("iofs: %w", err)
	}

	mg, err := migrate.NewWithSourceInstance("iofs", d, databaseURL)
	if err != nil {
		return fmt.Errorf("NewWithSourceInstance: %w", err)
	}

	err = mg.Up()
	if err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}

// CleanDB runs the down migrations to drop all tables, then runs up migrations to reset database.
func cleanDB(conn *sql.DB) error {
	d, err := iofs.New(migrations, "db/migration")
	if err != nil {
		return fmt.Errorf("iofs: %w", err)
	}

	mg, err := migrate.NewWithSourceInstance("iofs", d, databaseURL)
	if err != nil {
		return fmt.Errorf("NewWithSourceInstance: %w", err)
	}

	if err = mg.Down(); err != nil {
		return fmt.Errorf("drop: %w", err)
	}

	if err = mg.Up(); err != nil {
		return fmt.Errorf("up: %w", err)
	}

	return nil
}

func TestCreateJam(t *testing.T) {
	jamName := gofakeit.NounAbstract()
	want := jamDB.Jam{
		Name: jamName,
		Bpm:  90,
		// Defaults
		Capacity: 5,
	}
	arg := jamDB.CreateJamParams{
		Name:     want.Name,
		Bpm:      want.Bpm,
		Capacity: want.Capacity,
	}
	got, err := jamTestQueries.CreateJam(context.Background(), &arg)
	require.NoError(t, err)

	require.NotEmpty(t, got.ID, "ID should have a value")
	require.Equal(t, want.Name, got.Name)
	require.Equal(t, want.Bpm, got.Bpm)
	require.Equal(t, want.Capacity, got.Capacity)
}

func TestCreateAuth(t *testing.T) {
	want := authDB.User{
		Username: gofakeit.Username(),
		Email:    gofakeit.Email(),
	}
	arg := authDB.CreateUserParams{
		Username: want.Username,
		Email:    want.Email,
	}
	got, err := authTestQueries.CreateUser(context.Background(), &arg)
	require.NoError(t, err)

	require.NotEmpty(t, got.ID, "ID should have a value")
	require.Equal(t, want.Username, got.Username)
	require.Equal(t, want.Email, got.Email)
}
