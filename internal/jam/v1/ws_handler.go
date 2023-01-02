package service

import (
	"errors"
	"sync"

	"github.com/hyphengolang/prelude/http/websocket"
	"github.com/hyphengolang/prelude/types/suid"
)

type muxEntry struct {
	sid  suid.UUID
	pool *websocket.Pool
}

func (e muxEntry) String() string { return e.sid.ShortUUID().String() }

type mux struct {
	mu sync.Mutex
	mp map[suid.UUID]muxEntry
}

func (mux *mux) Store(e muxEntry) {
	mux.mu.Lock()
	{
		mux.mp[e.sid] = e
	}
	mux.mu.Unlock()
}

func (mux *mux) Load(sid suid.UUID) (pool *websocket.Pool, err error) {
	mux.mu.Lock()
	e, ok := mux.mp[sid]
	mux.mu.Unlock()

	if !ok {
		return nil, errors.New("pool not found")
	}

	return e.pool, nil
}
