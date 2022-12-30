package websocket_test

// ok
import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/hyphengolang/prelude/testing/is"
	"github.com/hyphengolang/prelude/types/suid"
	"github.com/rog-golang-buddies/rmx/internal/websocket"
)

// so I am defining a simple echo server here for testing
func testServerPartA() http.Handler {
	ctx := context.Background()

	s := websocket.NewSubscriber[any, any](
		ctx,
		2,
		512,
		2*time.Second,
		2*time.Second,
		nil,
	)

	mux := http.NewServeMux()

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			return
		}

		wsc := s.NewConn(conn, nil)
		s.Subscribe(wsc)
	})

	return mux
}

func TestSubscriber(t *testing.T) {
	// t.Skip()

	is := is.New(t)
	ctx := context.Background()

	srv := httptest.NewServer(testServerPartA())

	t.Cleanup(func() {
		srv.Close()
	})

	// NOTE - can you try this test for me please? passed, really?
	t.Run("create a new client and connect to echo server", func(t *testing.T) {
		wsPath := stripPrefix(srv.URL + "/ws") // this correct right? yup

		cli1, _, _, err := ws.DefaultDialer.Dial(ctx, wsPath)
		is.NoErr(err)      // connect cli1 to server
		defer cli1.Close() // ok

		cli2, _, _, err := ws.DefaultDialer.Dial(ctx, wsPath)
		is.NoErr(err)      // connect cli2 to server
		defer cli2.Close() // ok

		_, _, _, err = ws.DefaultDialer.Dial(ctx, wsPath)
		is.NoErr(err) // cannot connect to the server

		data := []byte("Hello World!")
		typ := []byte{1}
		m := append(typ, data...)

		err = wsutil.WriteClientBinary(cli1, m)
		is.NoErr(err) // send message to server

		// now I want to read the message from the server
		msg, err := wsutil.ReadServerBinary(cli2)
		is.NoErr(err)    // read message from server
		is.Equal(m, msg) // check if message is correct
	})
}

func testServerPartB() http.Handler {
	ctx := context.Background()

	type Info struct {
		Username string
	}

	b := websocket.NewBroker[Info, any](3, ctx)

	mux := http.NewServeMux()

	mux.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		s := websocket.NewSubscriber[Info, any](ctx, 2, 512, 2*time.Second, 2*time.Second, &Info{
			Username: "John Doe",
		})

		w.Header().Set("Location", "/"+s.GetID().ShortUUID().String())
	})

	// so I need to get the subscriber from parsing here right? yes
	mux.HandleFunc("/ws/{suid}", func(w http.ResponseWriter, r *http.Request) {
		sid, err := parseSUID(w, r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// can you see me?

		s, err := b.GetSubscriber(sid)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			return
		}

		wsc := s.NewConn(conn, nil)
		s.Subscribe(wsc)
	})

	return mux
}

func TestBroker(t *testing.T) {
	is := is.New(t)

	srv := httptest.NewServer(testServerPartB())

	t.Cleanup(func() {
		srv.Close()
	})

	t.Run("create a new subscriber", func(t *testing.T) {
		srv.Client().Post(srv.URL+"/create", "application/json", nil)

		is.NoErr(nil)
	})

	t.Run("connect to subscriber", func(t *testing.T) {
		t.Skip()

		is.NoErr(nil)
	})

	t.Run("delete a subscriber", func(t *testing.T) {
		t.Skip()
		is.NoErr(nil)
	})
}

var resource = func(s string) string {
	return s[strings.LastIndex(s, "/")+1:]
}

var stripPrefix = func(s string) string {
	return "ws" + strings.TrimPrefix(s, "http")
}

func parseSUID(w http.ResponseWriter, r *http.Request) (suid.UUID, error) {
	return suid.ParseString(chi.URLParam(r, "uuid"))
}

/*

 */
