package websocket_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rapidmidiex/rmx/pkg/websocket"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/hyphengolang/prelude/testing/is"
)

func testServerPartA() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", websocket.NewClient(2).ServeHTTP)

	return mux
}

func TestSubscriber(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	srv := httptest.NewServer(testServerPartA())

	t.Cleanup(func() { srv.Close() })

	t.Run("create a new client and connect to echo server", func(t *testing.T) {
		wsPath := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

		cli1, _, _, err := ws.DefaultDialer.Dial(ctx, wsPath)
		is.NoErr(err)      // connect cli1 to server
		defer cli1.Close() // ok

		cli2, _, _, err := ws.DefaultDialer.Dial(ctx, wsPath)
		is.NoErr(err)      // connect cli2 to server
		defer cli2.Close() // ok

		_, _, _, err = ws.DefaultDialer.Dial(ctx, wsPath)
		is.True(err != nil) // cannot connect to the server

		payload := []byte("Hello World!")

		err = wsutil.WriteClientText(cli1, payload)
		is.NoErr(err) // send message to server

		response, err := wsutil.ReadServerText(cli2)
		is.NoErr(err)               // read message from server
		is.Equal(payload, response) // check if message is correct
	})
}
