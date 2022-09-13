package api

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

func TestConnect(t *testing.T) {
	t.Run("we get a list of available Jam Session when we connect over websocket", func(t *testing.T) {

		server := httptest.NewServer(NewServer().Router)

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/v1/jam"

		conn, _, _, err := ws.DefaultDialer.Dial(context.Background(), wsURL)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		defer server.Close()
		defer conn.Close()

		listJamsReq := []byte(`{"type": "LIST_JAMS"}`)
		_, err = conn.Write(listJamsReq)
		if err != nil {
			t.Fatal("could not write to WS connection")
		}

		within(t, time.Millisecond*10, func() {
			msg, err := wsutil.ReadServerText(conn)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			want := `{"Jams":[]}` + "\n"
			if string(msg) != want {
				t.Errorf(`got "%s", want "%s"`, string(msg), want)
			}
		})
	})
}

func within(t testing.TB, d time.Duration, assert func()) {
	t.Helper()

	done := make(chan struct{}, 1)

	go func() {
		assert()
		done <- struct{}{}
	}()

	select {
	case <-time.After(d):
		t.Error("timed out")
	case <-done:
	}
}
