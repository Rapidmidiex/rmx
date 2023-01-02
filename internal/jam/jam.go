package jam

import (
	"encoding/json"
	"strings"

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
	// FIXME - ID ought to be public
	ID suid.UUID
	// FIXME - User ought to be public
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

func NewJam(bpm uint, cap uint) *Jam {
	sid := suid.NewUUID()

	if cap == 0 {
		cap = 10
	}

	if bpm == 0 {
		bpm = 80
	}

	j := &Jam{
		ID: sid,
		// Owner:    NewUser(""),
		Name:     sid.ShortUUID().String(),
		Capacity: cap,
		BPM:      bpm,
	}

	return j
}
