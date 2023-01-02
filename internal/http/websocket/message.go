package websocket

import (
	"encoding/json"
)

type WSMsgTyp int

const (
	Text WSMsgTyp = iota + 1
	JSON
	Leave
)

// type for parsing bytes into messages
type message struct {
	typ  WSMsgTyp
	data []byte
}

// Parses the bytes into the message type
func (m *message) parse(b []byte) {
	// the first byte represents the data type (Text, JSON, Leave)
	m.typ = WSMsgTyp(b[0])
	// and others represent the data itself
	m.data = b[1:]
}

func (m *message) marshall() []byte {
	return append([]byte{byte(m.typ)}, m.data...)
}

// Converts the given bytes to string
func (m *message) readText() (string, error) {
	return string(m.data), nil
}

// Converts the given bytes to JSON
func (m *message) readJSON() (any, error) {
	var v any
	if err := json.Unmarshal(m.data, v); err != nil {
		return nil, err
	}

	return v, nil
}
