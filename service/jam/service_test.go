package jam

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hyphengolang/prelude/testing/is"
	w2 "github.com/rog-golang-buddies/rmx/internal/websocket/x"
	// "github.com/rog-golang-buddies/rmx/internal/websocket"
)

func TestRoutes(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	h := NewService(context.Background(), chi.NewMux())
	srv := httptest.NewServer(h)

	// setup websocket
	conn, err := w2.Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/echo")
	is.NoErr(err) // dial error

	t.Cleanup(func() { conn.Close(); srv.Close() })

	t.Run(`connect to echo server`, func(t *testing.T) {
		err := conn.WriteString("Hello, World")
		is.NoErr(err) // write string to server

		msg, err := conn.ReadString()
		is.NoErr(err) // read string from server

		is.Equal(msg, "Hello, World") // server message == client message
	})
}
