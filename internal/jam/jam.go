package jam

import (
	"encoding/json"
	"strings"

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
