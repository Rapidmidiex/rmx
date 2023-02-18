package websocket

import (
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

func read(conn *connHander, cli *Client) {
	defer func() {
		cli.d <- conn
		conn.rwc.Close()
	}()

	conn.setReadDeadLine(pongWait)

	log.Printf("read: %v\n", conn)

	for {
		msg, err := conn.read()
		if err != nil {
			// handle error
			log.Printf("read err: %v\n", err)
			break
		}

		log.Printf("read msg: %v\n", msg)

		cli.bc <- msg
	}
}

func write(conn *connHander) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.rwc.Close()
	}()

	log.Printf("write: %v\n", conn)

	for {
		select {
		case msg, ok := <-conn.send:
			conn.setWriteDeadLine(writeWait)
			if !ok {
				conn.write(&wsutil.Message{OpCode: ws.OpClose, Payload: []byte{}})
				return
			}

			if err := conn.write(msg); err != nil {
				conn.logf("msg err: %v\n", err)
				return
			}
		case <-ticker.C:
			conn.setWriteDeadLine(writeWait)
			if err := conn.write(&wsutil.Message{OpCode: ws.OpPing, Payload: nil}); err != nil {
				conn.logf("ticker err: %v\n", err)
				return
			}
		}
	}
}

type Client struct {
	r, d chan *connHander
	bc   chan *wsutil.Message
	cs   map[*connHander]bool
	u    *ws.HTTPUpgrader

	// Capacity of the send channel.
	// If capacity is 0, the send channel is unbuffered.
	Capacity uint
}

// TODO -- should be able to close all connections via their own channels
func (cli *Client) Close() error {
	defer func() {
		// close channels
		close(cli.r)
		close(cli.d)
		close(cli.bc)
	}()

	cli.bc <- &wsutil.Message{OpCode: ws.OpClose, Payload: []byte{}} // broadcast close
	return nil
}

func NewClient(cap uint) *Client {
	cli := &Client{
		r:  make(chan *connHander),
		d:  make(chan *connHander),
		bc: make(chan *wsutil.Message),
		cs: make(map[*connHander]bool),
		u:  &ws.HTTPUpgrader{},
		// Capacity: cap,
	}

	go cli.listen()
	return cli
}

func (cli *Client) listen() {
	for {
		select {
		case conn := <-cli.r:
			cli.cs[conn] = true
		case conn := <-cli.d:
			delete(cli.cs, conn)
			close(conn.send)
		case msg := <-cli.bc:
			for conn := range cli.cs {
				select {
				case conn.send <- msg:
				default:
					close(conn.send)
					delete(cli.cs, conn)
				}
			}
		}
	}
}

func (cli *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO check capacity

	rwc, _, _, err := cli.u.Upgrade(r, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	l := log.Default()

	conn := &connHander{
		rwc:  rwc,
		send: make(chan *wsutil.Message),
		log:  l.Println,
		logf: l.Printf,
	}

	cli.r <- conn

	go read(conn, cli)
	go write(conn)
}

type connHander struct {
	rwc net.Conn

	send chan *wsutil.Message

	logf func(format string, v ...any)
	log  func(v ...any)
}

func (c *connHander) setWriteDeadLine(d time.Duration) error {
	return c.rwc.SetWriteDeadline(time.Now().Add(d))
}

func (c *connHander) setReadDeadLine(d time.Duration) error {
	return c.rwc.SetReadDeadline(time.Now().Add(d))
}

func (c *connHander) read() (*wsutil.Message, error) {
	r := wsutil.NewReader(c.rwc, ws.StateServerSide)

	for {
		h, err := r.NextFrame()
		if err != nil {
			return nil, err
		}

		if h.OpCode.IsControl() {
			if err := c.controlHandler(h, r); err != nil {
				return nil, err
			}
			continue
		}

		// where want = ws.OpText|ws.OpBinary
		// NOTE -- eq: h.OpCode != 0 && h.OpCode != want
		if want := (ws.OpText | ws.OpBinary); h.OpCode&want == 0 {
			if err := r.Discard(); err != nil {
				return nil, err
			}
			continue
		}

		p, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		return &wsutil.Message{OpCode: h.OpCode, Payload: p}, nil
	}
}

func (c *connHander) write(msg *wsutil.Message) error {
	// This is server-side
	frame := ws.NewFrame(msg.OpCode, true, msg.Payload)
	return ws.WriteFrame(c.rwc, frame)
}

func (c *connHander) controlHandler(h ws.Header, r io.Reader) error {
	switch op := h.OpCode; op {
	case ws.OpPing:
		return c.handlePing(h)
	case ws.OpPong:
		return c.handlePong(h)
	case ws.OpClose:
		return c.handleClose(h)
	}

	return wsutil.ErrNotControlFrame
}

func (c *connHander) handlePing(h ws.Header) error {
	c.log("ping")
	return nil
}

func (c *connHander) handlePong(h ws.Header) error {
	c.log("pong")
	return c.setReadDeadLine(pongWait)
}

func (c *connHander) handleClose(h ws.Header) error {
	c.log("close")
	return nil
}
