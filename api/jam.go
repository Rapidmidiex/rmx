package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
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
	newJamReq struct {
		Username    string `json:"username"`
		SessionName string `json:"session_name"`
		Tempo       uint   `json:"tempo"`
	}

	joinJamReq struct {
		Username  string `json:"username"`
		SessionID string `json:"session_id"`
	}
	wsReq struct {
		MessageType string   `json:"messageType"` // Type of application message. Ex: JAM_SESSION_CONNECT, JAM_SESSION_CREATE
		Payload     struct{} `json:"payload"`     // Payload of message, differs according to MessageType. Ex: JAM_SESSION_CREATE may contain Jam Session config (tempo, session name, etc).
	}

	JamSlim struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	}
	listJamsResp struct {
		Jams []JamSlim
	}
)

// new session handler
func (s *JamService) NewSession(w http.ResponseWriter, r *http.Request) {
	// get values from the request
	si := newJamReq{}
	if err := parse(r, &si); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
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

// Connect establishes a WebSocket connection with the application. From there we can communicate with the client about which session to join.
func (s *JamService) Connect(w http.ResponseWriter, r *http.Request) {
	// upgrade http connection to websocket
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		log.Println("connect", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	go func() {
		defer conn.Close()
		var (
			fr      = wsutil.NewReader(conn, ws.StateServerSide)
			fw      = wsutil.NewWriter(conn, ws.StateServerSide, ws.OpText)
			decoder = json.NewDecoder(fr)
			encoder = json.NewEncoder(fw)
		)
		for {
			hdr, err := fr.NextFrame()
			if err != nil {
				log.Println(err)
			}
			if hdr.OpCode == ws.OpClose {
				log.Println(io.EOF)
				// Break out of for and close the connection
				return
			}

			// TODO: Hand off messages to some app level message broker
			// Ex: JAM_SESSION_CONNECT, JAM_SESSION_CREATE
			var req wsReq
			if err := decoder.Decode(&req); err != nil {
				log.Println(err)
			}

			resp := listJamsResp{Jams: make([]JamSlim, 0)}
			if err := encoder.Encode(&resp); err != nil {
				log.Println(err)
			}
			if err = fw.Flush(); err != nil {
				log.Println(err)
			}
		}
	}()
}

// join session handler
func (s *JamService) JoinSession(w http.ResponseWriter, r *http.Request) {
	// get values from the request
	ji := joinJamReq{}
	if err := parse(r, &ji); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sID := chi.URLParam(r, "session_id")

	// err isn't nil if session doesn't exist
	session, err := s.getSession(sID)
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
