package jam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hyphengolang/prelude/testing/is"
	"github.com/rog-golang-buddies/rmx/internal/suid"
	w2 "github.com/rog-golang-buddies/rmx/internal/websocket/x"
	// "github.com/rog-golang-buddies/rmx/internal/websocket"
)

func TestService(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	h := NewService(context.Background(), chi.NewMux())
	srv := httptest.NewServer(h)

	t.Cleanup(func() { srv.Close() })

	t.Run(`create a new jam room`, func(t *testing.T) {
		r, _ := srv.Client().Get(srv.URL + "/api/v1/jam")
		is.Equal(r.StatusCode, http.StatusOK) // successfully created a new room
	})

	t.Run(`connect users to room websocket`, func(t *testing.T) {

		jamId := suid.NewSUID()

		c1, err := w2.Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/jam/"+jamId.String())
		is.NoErr(err) // connect client 1
		c2, err := w2.Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/jam/"+jamId.String())
		is.NoErr(err) // connect client 2

		t.Cleanup(func() {
			c1.Close()
			c2.Close()
		})

		_, err = w2.Dial(context.Background(), "ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/jam/"+jamId.String())
		is.True(err != nil) // cannot connect client 2

		err = c1.WriteString("Hello, World!")
		is.NoErr(err) // write string to pool

		msg, err := c2.ReadString()
		is.NoErr(err) // read message sent by c1

		is.Equal(msg, "Hello, World!")

	})
}
