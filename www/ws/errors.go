package websocket

import "errors"

var ErrNoPool = errors.New("ws: pool does not exist")
var ErrMaxConn = errors.New("ws: maximum number of connections reached")
