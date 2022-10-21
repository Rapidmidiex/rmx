package websocket

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type Conn interface {
	Reader
	Writer
}

type Reader interface {
	Close() error
	State() ws.State

	Read() ([]byte, error)
	ReadString() (string, error)
	ReadJSON(v any) error
}

type Writer interface {
	Close() error
	State() ws.State

	Write(p []byte) error
	WriteString(p string) error
	WriteJSON(v any) error
}

type serverConn struct {
	r *wsutil.Reader
	w *wsutil.Writer

	rwc net.Conn
}

func (c *serverConn) State() ws.State { return ws.StateServerSide }

func (c *serverConn) Close() error { return c.rwc.Close() }

func (c *serverConn) Read() ([]byte, error) {
	b, err := wsutil.ReadClientBinary(c.rwc)
	return b, err
}

func (c *serverConn) ReadString() (string, error) {
	h, err := c.r.NextFrame()
	if err != nil {
		return "", err
	}

	// Reset writer to write frame with right operation code.
	c.w.Reset(c.rwc, c.State(), h.OpCode)

	b, err := io.ReadAll(c.r)
	return string(b), err
}

func (c *serverConn) ReadJSON(v any) error {
	h, err := c.r.NextFrame()
	if err != nil {
		return err
	}

	if h.OpCode == ws.OpClose {
		return io.EOF
	}
	return json.NewDecoder(c.r).Decode(v)
}

func (c *serverConn) Write(p []byte) error {
	return wsutil.WriteMessage(c.rwc, c.State(), ws.OpBinary, p)
}

func (c *serverConn) WriteString(s string) error {
	_, err := io.WriteString(c.w, s)
	if err != nil {
		return err
	}
	return c.w.Flush()
}

func (c *serverConn) WriteJSON(v any) error {
	if err := json.NewEncoder(c.w).Encode(v); err != nil {
		return err
	}
	return c.w.Flush()
}

func UpgradeHTTP(w http.ResponseWriter, r *http.Request) (conn Conn, err error) {
	rwc, _, _, err := ws.UpgradeHTTP(r, w)
	c := &serverConn{rwc: rwc}
	c.r = wsutil.NewReader(rwc, c.State())
	c.w = wsutil.NewWriter(rwc, c.State(), ws.OpText)
	return c, err
}

type clientConn struct {
	r *wsutil.Reader
	w *wsutil.Writer

	rwc net.Conn
}

func (c *clientConn) State() ws.State { return ws.StateClientSide }

func (c *clientConn) Close() error { return c.rwc.Close() }

func (c *clientConn) Read() ([]byte, error) {
	b, err := wsutil.ReadServerBinary(c.rwc)
	return b, err
}

func (c *clientConn) ReadString() (string, error) {
	h, err := c.r.NextFrame()
	if err != nil {
		return "", err
	}
	// Reset writer to write frame with right operation code.
	c.w.Reset(c.rwc, c.State(), h.OpCode)

	b, err := io.ReadAll(c.r)
	return string(b), err
}

func (c *clientConn) ReadJSON(v any) error {
	h, err := c.r.NextFrame()
	if err != nil {
		return err
	}
	if h.OpCode == ws.OpClose {
		return io.EOF
	}
	return json.NewDecoder(c.r).Decode(v)
}

func (c *clientConn) Write(p []byte) error {
	return wsutil.WriteMessage(c.rwc, c.State(), ws.OpBinary, p)
}

func (c *clientConn) WriteString(s string) error {
	_, err := io.WriteString(c.w, s)
	if err != nil {
		return err
	}
	return c.w.Flush()
}

func (c *clientConn) WriteJSON(v any) error {
	if err := json.NewEncoder(c.w).Encode(v); err != nil {
		return err
	}
	return c.w.Flush()
}

func Dial(ctx context.Context, urlStr string) (conn Conn, err error) {
	rwc, _, _, err := ws.Dial(context.Background(), urlStr)
	c := &clientConn{rwc: rwc}
	c.r = wsutil.NewReader(rwc, c.State())
	c.w = wsutil.NewWriter(rwc, c.State(), ws.OpText)
	return c, err
}
