package websocket

import (
	"sync"

	"github.com/rog-golang-buddies/rmx/internal/suid"
)

type Multiplexer struct {
	ps map[suid.SUID]*Pool
}

func (mux *Multiplexer) Append(sid suid.SUID, pool Pool) {}

type Pool struct {
	Capacity uint

	seq uint
	cs  sync.Map
}

func (p *Pool) Append(conn Conn) {
	p.cs.Store(conn, struct{}{})
	p.seq++
}

func (p *Pool) IsCap() bool { return p.seq >= p.Capacity }

func (p *Pool) Remove(conn Conn) error {
	p.cs.Delete(conn)
	p.seq--
	return conn.Close()
}

// General broadcast method that can be used with any message type
func (p *Pool) Broadcast(msg any) {
	var f func(key, value any) bool

	switch msg := msg.(type) {
	case []byte:
		f = func(key, value any) bool { return key.(Conn).Write(msg) == nil }
	case string:
		f = func(key, value any) bool { return key.(Conn).WriteString(msg) == nil }
	default:
		f = func(key, value any) bool { return key.(Conn).WriteJSON(msg) == nil }
	}

	p.cs.Range(f)
}
