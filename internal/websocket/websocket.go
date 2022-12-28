package websocket

import (
	"context"

	"github.com/hyphengolang/prelude/types/suid"
)

func NewBroker[SI, CI any](cap uint, ctx context.Context) *Broker[SI, CI] {
	return &Broker[SI, CI]{
		ss:       make(map[suid.UUID]*Subscriber[SI, CI]),
		Capacity: cap,
		Context:  ctx,
	}
}

type Reader interface {
	ReadText() (string, error)
	ReadJSON() (interface{}, error)
}

type Writer interface {
	WriteText(s string)
	WriteJSON(i any) error
}
