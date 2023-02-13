package jam

import (
	"sync"

	rmx "github.com/rapidmidiex/rmx/internal"
)

type Broker rmx.Broker[string, *Jam]

type jamBroker struct {
	m sync.Map
}

func NewBroker() Broker {
	b := &jamBroker{}
	return b
}

func (b *jamBroker) Delete(id string) {
	b.m.Delete(id)
}

func (b *jamBroker) LoadAndDelete(id string) (value *Jam, loaded bool) {
	actual, loaded := b.m.LoadAndDelete(id)
	return actual.(*Jam), loaded
}

func (b *jamBroker) Load(id string) (value *Jam, ok bool) {
	v, ok := b.m.Load(id)
	return v.(*Jam), ok
}

func (b *jamBroker) Store(id string, jam *Jam) {
	b.m.Store(id, jam)
}

func (b *jamBroker) LoadOrStore(id string, j *Jam) (*Jam, bool) {
	actual, loaded := b.m.LoadOrStore(id, j)
	if !loaded {
		actual = j
	}

	return actual.(*Jam), loaded
}
