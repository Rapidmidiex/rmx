package internal

type MessageType int

const (
	Unknown = iota

	Create
	Delete

	Join
	Leave
	Message

	NoteOn
	NoteOff
)

func (t MessageType) String() string {
	switch t {
	case Create:
		return "CREATE"
	case Delete:
		return "DELETE"
	case Join:
		return "JOIN"
	case Leave:
		return "LEAVE"
	case Message:
		return "MESSAGE"
	case NoteOn:
		return "NOTE_ON"
	case NoteOff:
		return "NOTE_OFF"

	default:
		return "UNKNOWN"
	}
}

func (t *MessageType) UnmarshalJSON(b []byte) error {
	switch s := string(b[1 : len(b)-1]); s {
	case "CREATE":
		*t = Create
	case "DELETE":
		*t = Delete
	case "JOIN":
		*t = Join
	case "LEAVE":
		*t = Leave
	case "MESSAGE":
		*t = Message
	case "NOTE_ON":
		*t = NoteOn
	case "NOTE_OFF":
		*t = NoteOff
	default:
		*t = Unknown
	}

	return nil
}

func (t MessageType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

// ID with ability to encode to a URL friendly version
type ID string
