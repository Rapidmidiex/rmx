package websocket

import (
	"io"
	"sync"

	"github.com/hyphengolang/prelude/types/suid"
)

// A Web-Socket Connection
type Conn[CI any] struct {
	sid  suid.UUID
	rwc  io.ReadWriteCloser
	lock sync.RWMutex

	Info *CI
}
