package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
	"github.com/rog-golang-buddies/rapidmidiex/api/internal/db/user"
)

type jamConn struct {
	mu   sync.Mutex
	conn io.ReadWriter

	user.User
}

type jamSession struct {
	mu    sync.RWMutex
	conns map[string]*jamConn
	out   chan []interface{}

	id    string
	name  string
	tempo uint
}

type JamService struct {
	DBCon    *sql.DB
	mu       sync.RWMutex
	sessions map[string]*jamSession
}

type (
	newSessionReq struct {
		Name  string `json:"name"`
		Tempo uint   `json:"tempo"`
	}

	joinSessionReq struct {
		SessionID string `json:"session_id"`
	}
)

func (s *JamService) NewSession(w http.ResponseWriter, r *http.Request) {
	si := newSessionReq{}
	if err := parse(r, &si); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session := jamSession{
		conns: make(map[string]*jamConn),
		out:   make(chan []interface{}),

		id:    uuid.NewString(),
		name:  si.Name,
		tempo: si.Tempo,
	}
	s.addSession(&session)

	w.WriteHeader(http.StatusOK)
}

func (s *JamService) JoinSession(w http.ResponseWriter, r *http.Request) {
	email, ok := r.Context().Value(emailCtxKey).(string)
	if !ok {
		w.WriteHeader(http.StatusBadRequest) // TODO: shouldn't respond with bad request.
		return
	}

	ji := joinSessionReq{}
	if err := parse(r, &ji); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	session := s.sessions[ji.SessionID]
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	c := jamConn{}
	q := user.New(s.DBCon)
	userInfo, err := q.GetUserByEmail(context.Background(), email)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.User = userInfo
	c.conn = conn
	session.addConn(&c)
	session.broadcast("Welcome " + userInfo.Username + "!")

	w.WriteHeader(http.StatusOK)
}

func (s *JamService) addSession(js *jamSession) {
	s.mu.Lock()
	{
		s.sessions[js.id] = js
	}
	s.mu.Unlock()
}

func (s *jamSession) addConn(jc *jamConn) {
	s.mu.Lock()
	{
		s.conns[jc.Email] = jc
	}
	s.mu.Unlock()
}

func (c *jamConn) write(i interface{}) error {
	w := wsutil.NewWriter(c.conn, ws.StateServerSide, ws.OpText)
	encoder := json.NewEncoder(w)

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := encoder.Encode(i); err != nil {
		return err
	}

	return w.Flush()
}

func (c *jamConn) writeRaw(b []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.conn.Write(b)
	return err
}

// func (s *jamSession) writer(i interface{}) {
// 	for bts := range c.out {
// 		s.mu.RLock()
// 		cs := s.conns
// 		s.mu.RUnlock()
//
// 		for _, c := range cs {
// 			c := c // For closure.
// 			c.writeRaw(bts)
// 		}
// 	}
// }

func (s *jamSession) broadcast(i interface{}) {
	for _, c := range s.conns {
		select {
		case <-s.out:
			c.write(i)
		default:
			delete(s.conns, c.Email)
			// c.close()
		}
	}
}
