package websocket

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/rapidmidiex/rmx/internal/msg"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

func read(conn *connHandler, cli *Client) {
	defer func() {
		cli.unregister <- conn
		err := conn.rwc.Close()
		if err != nil {
			conn.logF("conn close: %v", err)
		}
		conn.debug("read: conn closed")
	}()

	if err := conn.setReadDeadLine(pongWait); err != nil {
		conn.logF("setReadDeadLine: %v\n", err)
		return
	}

	for {
		wsMsg, err := conn.read()
		if err != nil {
			// TODO: handle error
			conn.logF("read err: %v\n", err)
			break
		}

		// TODO: add a way use custom read validation here unsure how yet
		var envelope msg.Envelope
		if err := json.Unmarshal(wsMsg.Payload, &envelope); err != nil {
			log.Printf("wsMsg unmarshal: %v", err)
			log.Printf("read msg: OpCode: %v\n\n", wsMsg.OpCode)
		} else {
			log.Printf("read msg:\nType: %d\nID: %s\nUserID: %s\n\n", envelope.Typ, envelope.ID, envelope.UserID)
		}

		cli.broadcast <- wsMsg
	}
}

func write(conn *connHandler) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.debug("write: conn closed")
		if err := conn.rwc.Close(); err != nil {
			log.Printf("error closing connection: %v", err)
		}
	}()

	for {
		select {
		case msg, ok := <-conn.send:
			_ = conn.setWriteDeadLine(writeWait)
			if !ok {
				conn.debug("<-conn.send not ok")
				_ = conn.write(&wsutil.Message{OpCode: ws.OpClose, Payload: []byte{}})
				return
			}

			if err := conn.write(msg); err != nil {
				conn.logF("msg err: %v\n", err)
				return
			}
		case <-ticker.C:
			_ = conn.setWriteDeadLine(writeWait)
			if err := conn.write(&wsutil.Message{OpCode: ws.OpPing, Payload: nil}); err != nil {
				conn.logF("ticker err: %v\n", err)
				return
			}
		}
	}
}

type Client struct {
	register, unregister chan *connHandler
	broadcast            chan *wsutil.Message
	lock                 *sync.Mutex
	connections          map[*connHandler]bool
	upgrader             *ws.HTTPUpgrader

	// Capacity of the send channel.
	// If capacity is 0, the send channel is unbuffered.
	Capacity uint
}

// Len returns the number of connections.
func (cli *Client) Len() int {
	cli.lock.Lock()
	defer cli.lock.Unlock()
	return len(cli.connections)
}

// TODO -- should be able to close all connections via their own channels
func (cli *Client) Close() error {
	defer func() {
		log.Print("cli.Close()")
		// close channels
		close(cli.register)
		close(cli.unregister)
		close(cli.broadcast)
	}()

	cli.broadcast <- &wsutil.Message{OpCode: ws.OpClose, Payload: []byte{}} // broadcast close
	return nil
}

/*
NewClient instantiates a new websocket client.

NOTE: these may be useful to set: Capacity, ReadBufferSize, ReadTimeout, WriteTimeout
*/
func NewClient(cap uint) *Client {
	cli := &Client{
		register:    make(chan *connHandler),
		unregister:  make(chan *connHandler),
		broadcast:   make(chan *wsutil.Message),
		lock:        &sync.Mutex{},
		connections: make(map[*connHandler]bool),
		upgrader:    &ws.HTTPUpgrader{
			// TODO: may be fields here that worth setting
		},
		Capacity: cap,
	}

	go cli.listen()
	return cli
}

func (cli *Client) listen() {
	for {
		select {
		case conn := <-cli.register:
			cli.lock.Lock()
			cli.connections[conn] = true
			cli.lock.Unlock()
		case conn := <-cli.unregister:
			conn.debug("unregister channel handler")
			delete(cli.connections, conn)
			close(conn.send)
		case msg := <-cli.broadcast:
			for conn := range cli.connections {
				select {
				case conn.send <- msg:
				default:
					conn.debug("broadcast channel handler: default case")
					close(conn.send)
					delete(cli.connections, conn)
				}
			}
		}
	}
}

func (cli *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO check capacity
	if cli.Capacity > 0 && cli.Len() >= int(cli.Capacity) {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}

	rwc, _, _, err := cli.upgrader.Upgrade(r, w)
	if err != nil {
		// TODO log that there was an error
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	isDebug, _ := strconv.ParseBool(os.Getenv("DEBUG"))

	conn := &connHandler{
		rwc:  rwc,
		send: make(chan *wsutil.Message),
		log:  log.Println,
		logF: log.Printf,
		debug: func(v ...any) {
			if !isDebug {
				return
			}
			log.Println(v)
		},
		debugF: func(format string, v ...any) {
			if !isDebug {
				return
			}
			log.Printf(format, v)
		},
	}

	cli.register <- conn

	go read(conn, cli)
	go write(conn)
}

type connHandler struct {
	rwc net.Conn

	send chan *wsutil.Message

	logF func(format string, v ...any)
	log  func(v ...any)

	debugF func(format string, v ...any)
	debug  func(v ...any)
}

func (c *connHandler) setWriteDeadLine(d time.Duration) error {
	return c.rwc.SetWriteDeadline(time.Now().Add(d))
}

func (c *connHandler) setReadDeadLine(d time.Duration) error {
	return c.rwc.SetReadDeadline(time.Now().Add(d))
}

func (c *connHandler) read() (*wsutil.Message, error) {
	r := wsutil.NewReader(c.rwc, ws.StateServerSide)

	for {
		h, err := r.NextFrame()
		if err != nil {
			return nil, fmt.Errorf("next frame: %w", err)
		}

		if h.OpCode.IsControl() {
			if err := c.controlHandler(h, r); err != nil {
				return nil, fmt.Errorf("control handler: %w", err)
			}
			continue
		}

		/*
			// TODO check if this worth doing
			if !h.OpCode.IsData() {
				if h.OpCode.IsControl() {
					if err := c.controlHandler(h, r); err != nil {
						return nil, fmt.Errorf("control handler: %w", err)
					}
					continue
				}
			 	if err := r.Discard(); err != nil {
			 		return nil, fmt.Errorf("discard: %w", err)
			 	}
			 	continue
			}
		*/

		// where want = ws.OpText|ws.OpBinary
		// NOTE -- eq: h.OpCode != 0 && h.OpCode != want
		if want := (ws.OpText | ws.OpBinary); h.OpCode&want == 0 {
			if err := r.Discard(); err != nil {
				return nil, fmt.Errorf("discard: %w", err)
			}
			continue
		}

		// TODO the custom handler to parse payload could be done here (?)

		p, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("read all: %w", err)
		}
		return &wsutil.Message{OpCode: h.OpCode, Payload: p}, nil
	}
}

func (c *connHandler) write(msg *wsutil.Message) error {
	frame := ws.NewFrame(msg.OpCode, true, msg.Payload)
	return ws.WriteFrame(c.rwc, frame)
}

func (c *connHandler) controlHandler(h ws.Header, r io.Reader) error {
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

func (c *connHandler) handlePing(h ws.Header) error { c.log("ping"); return nil }

func (c *connHandler) handlePong(h ws.Header) error {
	c.log("pong")
	return c.setReadDeadLine(pongWait)
}

func (c *connHandler) handleClose(h ws.Header) error { c.log("close"); return nil }

type Broker[K, V any] interface {
	Load(key K) (value V, ok bool)
	LoadOrStore(key K, value V) (actual V, loaded bool)
	Store(key K, value V)
	Delete(key K)
	LoadAndDelete(key K) (value V, loaded bool)
}
