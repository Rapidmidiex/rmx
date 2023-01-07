package websocket

import (
	"io"
	"sync"

	"github.com/gobwas/ws/wsutil"
	"github.com/hyphengolang/prelude/types/suid"
)

// A Web-Socket Connection
type Conn[CI any] struct {
	sid  suid.UUID
	rwc  io.ReadWriteCloser
	lock sync.RWMutex

	Info *CI
}

// Writes raw bytes to the Connection
func (c *Conn[CI]) write(m *wsutil.Message) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return wsutil.WriteServerMessage(c.rwc, m.OpCode, m.Payload)
}
