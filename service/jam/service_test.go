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

func TestMultipleClients(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	h := NewService(context.Background(), chi.NewMux())
	srv := httptest.NewServer(h)

	t.Cleanup(func() { srv.Close() })

	t.Run(`connect to echo server`, func(t *testing.T) {
		c1, err := w2.Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/echo")
		is.NoErr(err) // dial error

		err = c1.WriteString("Hello, World!")
		is.NoErr(err) // write string to server

		msg, err := c1.ReadString()
		is.NoErr(err) // read string from server

		is.Equal(msg, "Hello, World!") // server message == client message

		c1.Close()
	})

	t.Run(`communicate between two connections`, func(t *testing.T) {
		c2, _ := w2.Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/echo")
		c3, _ := w2.Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/echo")

		err := c2.WriteString("Hello, World!")
		is.NoErr(err) // write string to pool

		// time.Sleep(time.Second * 1)

		msg, err := c3.ReadString()
		is.NoErr(err) // read message sent by c1

		is.Equal(msg, "Hello, World!")

		c2.Close()
		c3.Close()
	})
}
