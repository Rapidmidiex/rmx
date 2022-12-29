package websocket

type wserr[CI any] struct {
	conn *Conn[CI]
	msg  error
}

func (e *wserr[CI]) Error() string {
	return e.msg.Error()
}
