package jam

import (
	"fmt"
	"strings"
	"sync"

	fake "github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/hyphengolang/prelude/types/suid"
	"github.com/rapidmidiex/rmx/pkg/websocket"
)

const (
	defaultBPM      = 120
	defaultCapacity = 10
)

type User struct {
	ID       suid.UUID
	Username string `json:"username"`
}

func NewUser(username string) *User {
	if strings.TrimSpace(username) == "" {
		username = fake.Username()
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

func (j *Jam) String() string {
	return "jam no: " + j.ID.String()
}

// SetDefaults set default values for BPM, Name, and Capacity.
func (j *Jam) SetDefaults() {
	if j.BPM == 0 {
		j.BPM = defaultBPM
	}
	if j.Name == "" {
		j.Name = fmt.Sprintf("%s %s", fake.AdjectiveDescriptive(), fake.NounAbstract())
	}
	if j.Capacity == 0 {
		j.Capacity = defaultCapacity
	}
}

// Broker is responsible of delegating the creation of a new Jam and the
// management of the Jam's websocket clients.
type Broker websocket.Broker[uuid.UUID, *Jam]

type jamBroker struct {
	m sync.Map
}

func NewBroker() Broker {
	b := &jamBroker{}
	return b
}

// Delete deletes a jam from the broker.
func (b *jamBroker) Delete(id uuid.UUID) {
	b.m.Delete(id)
}

func (b *jamBroker) LoadAndDelete(id uuid.UUID) (value *Jam, loaded bool) {
	actual, loaded := b.m.LoadAndDelete(id)
	return actual.(*Jam), loaded
}

// Load loads an existing jam from the broker.
func (b *jamBroker) Load(id uuid.UUID) (value *Jam, ok bool) {
	v, ok := b.m.Load(id)
	return v.(*Jam), ok
}

// Store stores the jam in the broker.
func (b *jamBroker) Store(id uuid.UUID, jam *Jam) {
	b.m.Store(id, jam)
}

// LoadOrStore returns the existing jam for the id if present.
// Otherwise, it stores and returns the given jam.
// The loaded result is true if the value was loaded, false if stored.
func (b *jamBroker) LoadOrStore(id uuid.UUID, j *Jam) (*Jam, bool) {
	actual, loaded := b.m.LoadOrStore(id, j)
	if !loaded {
		actual = j
	}

	return actual.(*Jam), loaded
}
