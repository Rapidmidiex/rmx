package websocket

type wsErr[CI any] struct {
	conn *Conn[CI]
	msg  error
}

func (e *wsErr[CI]) Error() string {
	return e.msg.Error()
}
