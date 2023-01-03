package websocket

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
