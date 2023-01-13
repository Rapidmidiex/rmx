package jam

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/hyphengolang/prelude/types/suid"
)

type User struct {
	ID       suid.UUID
	Username string `json:"username"`
}

func NewUser(username string) *User {
	sid := suid.NewUUID()

	if strings.TrimSpace(username) == "" {
		username = sid.ShortUUID().String()
	}

	u := &User{
		ID:       suid.NewUUID(),
		Username: username,
	}

	return u
}

type Jam struct {
	ID       uuid.UUID
	Owner    *User
	Name     string `json:"name,omitempty"`
	Capacity uint   `json:"capacity,omitempty"`
	BPM      uint   `json:"bpm,omitempty"`
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
		j.Capacity = 10
	}

	if aux.BPM == 0 {
		j.BPM = 80
	}

	return nil
}

// SetDefaults set default values for BPM, Name, and Capacity.
func (j *Jam) SetDefaults() {
	// We probably want to declare these defaults somewhere else
	if j.BPM == 0 {
		j.BPM = 120
	}
	if j.Name == "" {
		j.Name = fmt.Sprintf("%s  %s", gofakeit.AdjectiveDescriptive(), gofakeit.NounAbstract())
	}
	if j.Capacity == 0 {
		j.Capacity = 5
	}
}
