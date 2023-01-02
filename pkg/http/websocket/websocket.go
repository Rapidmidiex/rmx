package websocket

type Reader interface {
	ReadText() (string, error)
	ReadJSON() (interface{}, error)
}

type Writer interface {
	WriteText(s string)
	WriteJSON(i any) error
}
