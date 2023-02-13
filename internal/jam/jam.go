package jam

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/hyphengolang/prelude/types/suid"
	"github.com/rapidmidiex/rmx/internal/http/websocket/v2"
)

const (
	defaultCapacity = 5
	defaultBPM      = 120
)

type User struct {
	ID       suid.UUID
	Username string `json:"username"`
}

func NewUser(username string) *User {
	if strings.TrimSpace(username) == "" {
		username = gofakeit.Username()
	}

	u := &User{
		ID:       suid.NewUUID(),
		Username: username,
	}

	return u
}

type Jam struct {
	ID       uuid.UUID `json:"id"`
	Owner    *User     `json:"owner,omitempty"`
	Name     string    `json:"name,omitempty"`
	Capacity uint      `json:"capacity,omitempty"`
	BPM      uint      `json:"bpm,omitempty"`

	cli *websocket.Client
}

// NOTE this should not be empty but panic if it is
func (j *Jam) Client() *websocket.Client {
	if j.cli == nil {
		j.cli = websocket.NewClient(j.Capacity)
	}

	return j.cli
}

func (j *Jam) Close() error {
	return j.cli.Close()
}

// NOTE -- should init on creation as this is just spinning up excessive goroutines
func (j *Jam) SetClient(cli *websocket.Client) {
	if j.cli == nil {
		j.cli = cli
	}
}

func (j *Jam) String() string {
	return "jam no: " + j.ID.String()
}

func (j *Jam) UnmarshalJSON(data []byte) error {
	type Alias Jam
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(j),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Capacity == 0 {
		j.Capacity = defaultCapacity
	}

	if aux.BPM == 0 {
		j.BPM = defaultBPM
	}

	return nil
}

// FIXME deprecated, PLEASE DELETE
// SetDefaults set default values for BPM, Name, and Capacity.
func (j *Jam) SetDefaults() {
	// We probably want to declare these defaults somewhere else
	if j.BPM == 0 {
		j.BPM = 120
	}
	if j.Name == "" {
		j.Name = fmt.Sprintf("%s %s", gofakeit.AdjectiveDescriptive(), gofakeit.NounAbstract())
	}
	if j.Capacity == 0 {
		j.Capacity = 5
	}
}
