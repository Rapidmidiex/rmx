package jam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/hyphengolang/prelude/testing/is"
	// "github.com/rog-golang-buddies/rmx/internal/websocket"
)

var resource = func(s string) string {
	return s[strings.LastIndex(s, "/")+1:]
}

var stripPrefix = func(s string) string {
	return "ws" + strings.TrimPrefix(s, "http")
}

func TestService(t *testing.T) {
	is := is.New(t)

	ctx, mux := context.Background(), chi.NewMux()

	h := NewService(ctx, mux)
	srv := httptest.NewServer(h)

	t.Cleanup(func() { srv.Close() })

	var firstPool string
	t.Run(`create a new Room`, func(t *testing.T) {
		payload := `
		{
			"capacity": 2
		}`

		res, _ := srv.Client().Post(srv.URL+"/api/v1/jam", "application/json", strings.NewReader(payload))
		is.Equal(res.StatusCode, http.StatusCreated) // created a new resource

		loc, err := res.Location()
		is.NoErr(err) // retrieve location

		firstPool = resource(loc.Path)
	})

	t.Run(`connect to jam session`, func(t *testing.T) {
		c1, _, err := websocket.DefaultDialer.Dial(stripPrefix(srv.URL+"/ws/jam/"+firstPool), nil)
		is.NoErr(err) // found first jam Session

		t.Cleanup(func() { c1.Close() })

		err = c1.WriteMessage(websocket.TextMessage, []byte("Hello, World!"))
		is.NoErr(err) // write to pool

		_, data, err := c1.ReadMessage()
		is.NoErr(err) // read from pool

		is.Equal(string(data), "Hello, World!")
	})
}

// func TestWebsocket(t *testing.T) {
// 	// t.Parallel()
// 	is := is.New(t)

// 	h := NewService(context.Background(), chi.NewMux())
// 	srv := httptest.NewServer(h)

// 	t.Cleanup(func() { srv.Close() })

// 	t.Run(`list all jam sessions`, func(t *testing.T) {
// 		r, _ := srv.Client().Get(srv.URL + "/api/v1/jam")
// 		is.Equal(r.StatusCode, http.StatusOK) // successfully created a new room
// 	})

// 	var firstPool string
// 	t.Run(`create a new room`, func(t *testing.T) {
// 		payload := `
// {
// 	"capacity":2
// }`
// 		res, err := srv.Client().Post(srv.URL+"/api/v1/jam", "application/json", strings.NewReader(payload))
// 		is.NoErr(err)                                // create a new pool
// 		is.Equal(res.StatusCode, http.StatusCreated) // created a new resource

// 		loc, err := res.Location()
// 		is.NoErr(err) // retrieve location

// 		firstPool = resource(loc.Path)
// 	})

// 	t.Run(`connect users to room websocket`, func(t *testing.T) {
// 		c1, err := w2.Dial(context.Background(), stripPrefix(srv.URL+"/ws/jam/")+firstPool)
// 		is.NoErr(err) // connect client 1
// 		c2, err := w2.Dial(context.Background(), stripPrefix(srv.URL+"/ws/jam/")+firstPool)
// 		is.NoErr(err) // connect client 2

// 		t.Cleanup(func() {
// 			c1.Close()
// 			c2.Close()
// 		})

// 		_, err = w2.Dial(context.Background(), stripPrefix(srv.URL+"/ws/jam/")+firstPool)
// 		is.True(err != nil) // cannot connect client 3

// 		err = c1.WriteString("Hello, World!")
// 		is.NoErr(err) // write string to pool

// 		msg, err := c2.ReadString()
// 		is.NoErr(err) // read message sent by c1

// 		is.Equal(msg, "Hello, World!")

// 	})
// }
