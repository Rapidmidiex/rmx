package websocket

import "github.com/gobwas/ws"

type Reader interface {
	ReadText() (string, error)
	ReadJSON() (interface{}, error)
}

type Writer interface {
	WriteText(s string)
	WriteJSON(i any) error
}

func OpCodeToString(o ws.OpCode) string {
	switch o {
	case ws.OpContinuation:
		return "OpContinuation"
	case ws.OpText:
		return "OpText"
	case ws.OpBinary:
		return "OpBinary"
	case ws.OpClose:
		return "OpClose"
	case ws.OpPing:
		return "OpPing"
	case ws.OpPong:
		return "OpPong"
	}
	return "Unknown"
}
