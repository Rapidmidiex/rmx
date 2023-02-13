package jam

import (
	"sync"

	"github.com/google/uuid"
	rmx "github.com/rapidmidiex/rmx/internal"
	"github.com/rapidmidiex/rmx/internal/http/websocket/v2"
)

type Thread struct {
	ID       uuid.UUID  `json:"id"`
	Messages []*Message `json:"messages"`

	cli *websocket.Client
}

// NOTE this should not be empty but panic if it is
func (thr *Thread) Client() *websocket.Client {
	if thr.cli == nil {
		thr.cli = websocket.NewClient()
	}

	return thr.cli
}

func (thr *Thread) Close() error {
	return thr.cli.Close()
}

// NOTE -- should init on creation as this is just spinning up excessive goroutines
func (thr *Thread) SetClient(cli *websocket.Client) {
	if thr.cli == nil {
		thr.cli = cli
	}
}

func NewThread() *Thread {
	thread := Thread{ID: uuid.New()}

	return &thread
}

func (c *Thread) String() string {
	return "chat no: " + c.ID.String()
}

type Message struct {
	ID      int     `json:"id,omitempty"`
	Content string  `json:"content"`
	Thread  *Thread `json:"-"`
}

type Broker rmx.Broker[string, *Thread]

type threadBroker struct {
	m sync.Map
}

func NewBroker() Broker {
	b := &threadBroker{}
	return b
}

func (b *threadBroker) Delete(id string) {
	b.m.Delete(id)
}

func (b *threadBroker) LoadAndDelete(id string) (value *Thread, loaded bool) {
	actual, loaded := b.m.LoadAndDelete(id)
	return actual.(*Thread), loaded
}

func (b *threadBroker) Load(id string) (value *Thread, ok bool) {
	v, ok := b.m.Load(id)
	return v.(*Thread), ok
}

func (b *threadBroker) Store(id string, thread *Thread) {
	b.m.Store(id, thread)
}

func (b *threadBroker) LoadOrStore(id string, chat *Thread) (*Thread, bool) {
	actual, loaded := b.m.LoadOrStore(id, chat)
	if !loaded {
		actual = chat
	}

	return actual.(*Thread), loaded
}
