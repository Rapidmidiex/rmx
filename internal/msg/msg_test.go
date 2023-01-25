package msg_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/rapidmidiex/rmx/internal/msg"
	"github.com/stretchr/testify/require"
)

func TestEnvelope(t *testing.T) {
	t.Run("wraps and unwraps Text messages", func(t *testing.T) {
		envelope := msg.Envelope{
			Typ:    msg.TEXT,
			UserID: uuid.New(),
		}
		payload := msg.TextMsg{Body: "Howdy"}
		err := envelope.SetPayload(payload)
		require.NoError(t, err)

		var got msg.TextMsg
		err = envelope.Unwrap(&got)
		require.NoError(t, err)
		require.Equal(t, payload, got)
	})

	t.Run("wraps and unwraps MIDI messages", func(t *testing.T) {
		envelope := msg.Envelope{
			Typ:    msg.MIDI,
			UserID: uuid.New(),
		}
		payload := msg.MIDIMsg{
			State:    msg.NOTE_ON,
			Number:   67,
			Velocity: 127,
		}

		err := envelope.SetPayload(payload)
		require.NoError(t, err)

		var got msg.MIDIMsg
		err = envelope.Unwrap(&got)
		require.NoError(t, err)
		require.Equal(t, payload, got)
	})
}
