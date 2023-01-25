// Package msg contains the RMX message types for communication between clients.
package msg

import (
	"encoding/json"

	"github.com/google/uuid"
)

type (
	MsgType   int
	NoteState int

	Envelope struct {
		// Message identifier
		ID uuid.UUID `json:"id"`
		// TextMsg | MIDIMsg | ConnectMsg
		Typ MsgType `json:"type"`
		// RMX client identifier
		UserID uuid.UUID `json:"userId"`
		// Actual message data.
		Payload json.RawMessage `json:"payload"`
	}

	TextMsg struct {
		DisplayName string `json:"displayName"`
		Body        string `json:"body"`
	}

	MIDIMsg struct {
		State NoteState `json:"state"`
		// MIDI Note # in "C3 Convention", C3 = 60. Available values: (0-127)
		Number int `json:"number"`
		// MIDI Velocity (0-127)
		Velocity int `json:"velocity"`
	}

	ConnectMsg struct {
		UserID   uuid.UUID `json:"userId"`
		UserName string    `json:"userName"`
	}
)

const (
	TEXT MsgType = iota
	MIDI
	CONNECT
)

const (
	NOTE_ON NoteState = iota
	NOTE_OFF
)

func (e *Envelope) SetPayload(payload any) error {
	p, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	e.Payload = p
	return nil
}

func (e *Envelope) Unwrap(msg any) error {
	return json.Unmarshal(e.Payload, msg)
}
