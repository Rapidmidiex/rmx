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
		Typ    MsgType   `json:"type"`
		UserID uuid.UUID `json:"userId"`
		// TextMsg | MIDIMsg | ConnectMsg
		Payload json.RawMessage `json:"payload"`
	}

	TextMsg struct {
		Body string `json:"body"`
	}

	MIDIMsg struct {
		State  NoteState `json:"state"`
		Number int       `json:"number"`
		UserID uuid.UUID `json:"userId"`
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
