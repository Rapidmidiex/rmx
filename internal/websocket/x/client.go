package websocket

import (
	"sync"
)

type Pool struct {
	Capacity uint

	seq uint
	cs  sync.Map
}

func NewPool(cap uint) *Pool {
	return &Pool{Capacity: cap}
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
