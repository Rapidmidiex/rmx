package jam

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/hyphengolang/prelude/types/suid"
)

const (
	defaultCapacity = 5
	defaultBPM      = 120
)

type Capacity uint

// Implements the UnmarshalJSON interface to set default values
func (c *Capacity) UnmarshalJSON(data []byte) error {
	type Alias Capacity
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Alias == nil {
		*c = Capacity(defaultCapacity)
	}

	return nil
}

type BPM uint

// Implements the UnmarshalJSON interface to set default values
func (b *BPM) UnmarshalJSON(data []byte) error {
	type Alias BPM
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(b),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Alias == nil {
		*b = BPM(defaultBPM)
	}

	return nil
}

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
