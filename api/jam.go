package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/google/uuid"
)

// struct type to store info related to a websocket connection
type jamConn struct {
	mu   sync.Mutex
	conn io.ReadWriter

	id       string
	username string
}

// struct type to store info related to a Jam session
// also contains a map of current connections to the session
type jamSession struct {
	mu    sync.RWMutex
	conns map[string]*jamConn
	out   chan []interface{}

	id    string
	name  string
	tempo uint
	owner string
}

// struct type for the Jam service
// also contains current available sessions created by users
type JamService struct {
	mu       sync.RWMutex
	sessions map[string]*jamSession
}

// request types
type (
	newSessionReq struct {
		Username    string `json:"username"`
		SessionName string `json:"session_name"`
		Tempo       uint   `json:"tempo"`
	}

	joinSessionReq struct {
		Username  string `json:"username"`
		SessionID string `json:"session_id"`
	}
)

// new session handler
func (s *JamService) NewSession(w http.ResponseWriter, r *http.Request) {
	// get values from the request
	si := newSessionReq{}
	if err := parse(r, &si); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// upgrade http connection to websocket
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// create a new connection between the owner and the server
	// then add it to the session connections
	c := jamConn{}
	c.conn = conn
	c.id = uuid.NewString()
	if isEmptyString(si.Username) {
		c.username = c.id
	} else {
		c.username = si.Username
	}

	// create a new session and set the session owner
	session := jamSession{
		conns: make(map[string]*jamConn),
		out:   make(chan []interface{}),

		id:    uuid.NewString(),
		name:  si.SessionName,
		tempo: si.Tempo,
		owner: c.id,
	}
	// add session to sessions map
	s.addSession(&session)
	// check for errors
	// err isn't nil if the username is already used
	session.addConn(&c)
	session.broadcast("Welcome to Rapidmidiex!")

	w.WriteHeader(http.StatusOK)
}

// join session handler
func (s *JamService) JoinSession(w http.ResponseWriter, r *http.Request) {
	// get values from the request
	ji := joinSessionReq{}
	if err := parse(r, &ji); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// check if session exists
	session, err := s.getSession(ji.SessionID)
	if err != nil {
		handlerError(w, err)
		return
	}

	// upgrade http connection to websocket
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// create a new connection
	c := jamConn{}
	c.conn = conn
	c.id = uuid.NewString()
	if isEmptyString(ji.Username) {
		c.username = c.id
	} else {
		c.username = ji.Username
	}
	session.addConn(&c)
	session.broadcast("Welcome " + ji.Username + "!")

	w.WriteHeader(http.StatusOK)
}

func (s *JamService) addSession(js *jamSession) {
	s.mu.Lock()
	s.sessions[js.id] = js
	s.mu.Unlock()
}

func (s *JamService) getSession(sID string) (*jamSession, error) {
	session, ok := s.sessions[sID]
	if ok {
		return &jamSession{}, &errSessionNotFound
	}

	return session, nil
}

func (s *jamSession) addConn(jc *jamConn) {
	s.mu.Lock()
	s.conns[jc.id] = jc
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

// iterates through session connections
// and send provided message to each of them
func (s *jamSession) broadcast(i interface{}) {
	for _, c := range s.conns {
		select {
		case <-s.out:
			c.write(i)
		default:
			delete(s.conns, c.username)
			// c.close()
		}
	}
}
