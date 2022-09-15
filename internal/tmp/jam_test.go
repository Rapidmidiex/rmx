// * Will be adding the testing back once we can adapt it for the `gorilla/websocket` package
package tmp

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

		js := NewJamService()
		server := httptest.NewServer(NewServer(js).Router)

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/v1/jam"

		conn, _, _, err := ws.DefaultDialer.Dial(context.Background(), wsURL)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		defer server.Close()
		defer conn.Close()

		listJamsReq := []byte(`{"messageType": "JAM_LIST"}`)
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

func TestJamConnect(t *testing.T) {
	t.Run("we receive a welcome message when entering a Jam", func(t *testing.T) {
		// Hardcoding the Jam ID for now.
		// In the future we'll connect through a "JAM_JOIN" message sequence.
		// An empty payload denotes the client wants to join any available Jam.
		// Req: { "messageType": "JAM_JOIN", "payload": {} } → server
		// Response: client ← { "jamId": "123456789" }
		// Client makes a new ws request @ /ws/v1/jam/123456789
		jamId := "123456789"

		js := NewJamService()
		server := httptest.NewServer(NewServer(js).Router)
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/v1/jam/" + jamId

		conn, _, _, err := ws.DefaultDialer.Dial(context.Background(), wsURL)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		defer server.Close()
		defer conn.Close()

		listJamsReq := []byte(`{"messageType": "JAM_HELLO"}`)
		_, err = conn.Write(listJamsReq)
		if err != nil {
			t.Fatal("could not write to WS connection")
		}

		within(t, time.Millisecond*10, func() {
			msg, err := wsutil.ReadServerText(conn)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			want := `{"messageText":"Welcome to Jam 123456789!"}` + "\n"
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
