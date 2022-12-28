package v2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/hyphengolang/prelude/testing/is"
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

	var firstJam string
	t.Run("Create a new Jam room", func(t *testing.T) {
		payload := `{
			"name":     "John Doe",
			"capacity": 5,
			"bpm":      100
		}`

		res, _ := srv.Client().Post(srv.URL+"/api/v1/jam", "application/json", strings.NewReader(payload))
		is.Equal(res.StatusCode, http.StatusCreated) // created a new resource

		loc, err := res.Location()
		is.NoErr(err) // retrieve location

		firstJam = resource(loc.Path)
	})

	t.Run(`Connect to Jam room with id: `+firstJam, func(t *testing.T) {
		c1, _, _, err := ws.DefaultDialer.Dial(ctx, stripPrefix(srv.URL+"/ws/jam/"+firstJam))
		is.NoErr(err) // found first jam Session

		t.Cleanup(func() { c1.Close() })

		data := []byte("Hello World!")
		typ := []byte{1}
		m := append(typ, data...)

		err = wsutil.WriteClientBinary(c1, m)
		is.NoErr(err) // write to pool

		res, err := wsutil.ReadServerBinary(c1)
		is.NoErr(err) // read from pool

		is.Equal(res, m)
	})
}
