package jam

import (
	"fmt"
	"strings"
	"sync"

	fake "github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/hyphengolang/prelude/types/suid"
	"github.com/rapidmidiex/rmx/internal/http/websocket/v2"
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

// NOTE -- should init on creation as this is just spinning up excessive goroutines
func (j *Jam) SetClient(cli *websocket.Client) {
	if j.cli == nil {
		j.cli = cli
	}
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

func (b *jamBroker) Delete(id uuid.UUID) {
	b.m.Delete(id)
}

func (b *jamBroker) LoadAndDelete(id uuid.UUID) (value *Jam, loaded bool) {
	actual, loaded := b.m.LoadAndDelete(id)
	return actual.(*Jam), loaded
}

func (b *jamBroker) Load(id uuid.UUID) (value *Jam, ok bool) {
	v, ok := b.m.Load(id)
	return v.(*Jam), ok
}

func (b *jamBroker) Store(id uuid.UUID, jam *Jam) {
	b.m.Store(id, jam)
}

func (b *jamBroker) LoadOrStore(id uuid.UUID, j *Jam) (*Jam, bool) {
	actual, loaded := b.m.LoadOrStore(id, j)
	if !loaded {
		actual = j
	}

	return actual.(*Jam), loaded
}
