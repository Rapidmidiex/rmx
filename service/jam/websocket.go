package jam

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// default values
const (
	// Maximum message size allowed from peer.
	readBufferSize = 512
	// Time allowed to read the next pong message from the peer.
	readDeadline = 60 * time.Second
	// Time allowed to write a message to the peer.
	writeDeadline = 10 * time.Second
)

var defaultUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func UpgradeHTTP(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return defaultUpgrader.Upgrade(w, r, nil)
}

type Pool struct {
	// for concurrent handling
	mu sync.Mutex
	// map for clients being used as a Set
	cs map[*connHandler]struct{}

	// maximum cap clients allowed
	Capacity uint
	// Maximum message size allowed from peer.
	ReadBufferSize int64
	// Time allowed to read the next pong message from the peer.
	ReadTimeout time.Duration
	// Time allowed to write a message to the peer.
	WriteTimeout time.Duration

	// Send os
	send chan Packet
	r, d chan *connHandler
	err  chan error

	isServing bool
}

// ListenAndServe listens to incoming websocket
func (pool *Pool) ListenAndServe(rwc *websocket.Conn) {
	if !pool.isServing {
		// Setup channels
		pool.r = make(chan *connHandler)
		pool.d = make(chan *connHandler)
		pool.cs = make(map[*connHandler]struct{})
		pool.err = make(chan error)
		pool.send = make(chan Packet)

		if pool.ReadBufferSize == 0 {
			pool.ReadBufferSize = readBufferSize
		}

		if pool.ReadTimeout == 0 {
			pool.ReadTimeout = readDeadline
		}

		if pool.WriteTimeout == 0 {
			pool.WriteTimeout = writeDeadline
		}

		go listen(pool)
		pool.isServing = true
	}

	ch := &connHandler{send: make(chan Packet), pool: pool, rwc: rwc}
	pool.r <- ch

	go read(ch)
	go write(ch)
}

func listen(pool *Pool) {
	for {
		select {
		case ch := <-pool.r:
			if !pool.isFull() {
				pool.cs[ch] = struct{}{}
			} else {
				// TODO write error to client
				// ch.respondError(websocket.CloseInternalServerErr, errors.New("pool is full, connection closed"))
				close(ch.send)
			}
		case ch := <-pool.d:
			if _, ok := pool.cs[ch]; ok {
				delete(pool.cs, ch)
				close(ch.send)
			}
		case p := <-pool.send:
			for ch := range pool.cs {
				select {
				case ch.send <- p:
				default:
					close(ch.send)
					delete(pool.cs, ch)
				}
			}
		// NOTE can this be cleaned up?
		case err := <-pool.err:
			if err != nil {
				pool.Close()
			}
		}
	}
}

func (pool *Pool) Close() {
	pool.mu.Lock()
	{
		for ch := range pool.cs {
			pool.d <- ch
		}
	}
	pool.mu.Unlock()

	pool.isServing = false
	close(pool.d)
	close(pool.r)
	close(pool.send)
	close(pool.err)
}

func (pool *Pool) isFull() bool { return int(pool.Capacity) == len(pool.cs) }

// func (pool *Pool) isEmpty() bool { return len(pool.cs) == 0 }

type connHandler struct {
	send chan Packet
	pool *Pool

	rwc *websocket.Conn
}

func (ch *connHandler) respond(typ int, data []byte) error {
	return ch.rwc.WriteMessage(typ, data)
}

func (ch *connHandler) respondError(code int, reason error) error {
	var msg []byte
	if reason != nil {
		msg = websocket.FormatCloseMessage(code, reason.Error())
	} else {
		msg = websocket.FormatCloseMessage(code, "")
	}
	return ch.respond(websocket.CloseMessage, msg)
}

func read(ch *connHandler) {
	defer func() {
		ch.pool.d <- ch
		ch.rwc.Close()
	}()

	ch.rwc.SetReadLimit(ch.pool.ReadBufferSize)
	ch.rwc.SetReadDeadline(time.Now().Add(ch.pool.ReadTimeout))

	for {
		typ, p, err := ch.rwc.ReadMessage()
		if err != nil {
			ch.respondError(websocket.CloseInternalServerErr, nil)
			break
		}

		ch.pool.send <- Packet{typ, p}
	}
}

func write(ch *connHandler) {
	defer func() {
		// ticker.Stop()
		ch.rwc.Close()
	}()

	for {
		select {
		case p, ok := <-ch.send:
			ch.rwc.SetWriteDeadline(time.Now().Add(ch.pool.WriteTimeout))
			if !ok {
				// the Pool closed ch.send
				ch.respondError(websocket.CloseInternalServerErr, errors.New("pool closed connection"))
				return
			}

			typ, data := p.parse()
			if err := ch.rwc.WriteMessage(typ, data); err != nil {
				ch.respondError(websocket.CloseInternalServerErr, err)
				return
			}
		}
	}
}

type Packet struct {
	typ  int
	data []byte
}

func (p Packet) parse() (int, []byte) { return p.typ, p.data }
