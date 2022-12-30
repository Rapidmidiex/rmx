package service_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rog-golang-buddies/rmx/docker/container"
	service "github.com/rog-golang-buddies/rmx/domain/user/http"
	"github.com/rog-golang-buddies/rmx/domain/user/postgres"

	"github.com/hyphengolang/prelude/testing/is"
)

const (
	dbName         = "user"
	pgUser         = "postgres"
	pgPass         = "postgres"
	dbPort         = "5432"
	occurrence     = 2
	startUpTimeout = 5 * time.Second
)

var postgresContainer *container.PostgresContainer
var server *httptest.Server
var conn *pgxpool.Pool

func init() {
	ctx := context.Background()

	var err error
	postgresContainer, conn,
		err = container.NewDefaultPostgresConnection(ctx, dbPort, pgUser, pgPass, dbName, occurrence, startUpTimeout)
	if err != nil {
		panic(err)
	}

	db := postgres.New(conn)

	server = httptest.NewServer(service.NewService(db))
}

func TestService(t *testing.T) {
	is := is.New(t)

	ctx := context.Background()

	t.Cleanup(func() {
		conn.Close()

		err := postgresContainer.Terminate(ctx)
		is.NoErr(err) // terminate the container

		server.Close()
	})

	t.Run("ping service", func(t *testing.T) {
		res, err := server.Client().Get(server.URL + "/ping")
		is.NoErr(err)                           // ping server
		is.Equal(res.StatusCode, http.StatusOK) // return ok status
	})
}
