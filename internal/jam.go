package internal

type MessageTyp int

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

func (t MessageTyp) String() string {
	switch t {
	case Create:
		return "Create"
	case Delete:
		return "Delete"
	case Join:
		return "Join"
	case Leave:
		return "Leave"
	case Message:
		return "Message"
	case NoteOn:
		return "NoteOn"
	case NoteOff:
		return "NoteOff"

	default:
		return "Unknown"
	}
}

func (t *MessageTyp) UnmarshalJSON(b []byte) error {
	switch s := string(b[1 : len(b)-1]); s {
	case "Create":
		*t = Create
	case "Delete":
		*t = Delete
	case "Join":
		*t = Join
	case "Leave":
		*t = Leave
	case "Message":
		*t = Message
	case "NoteOn":
		*t = NoteOn
	case "NoteOff":
		*t = NoteOff
	default:
		*t = Unknown
	}

	return nil
}

func (t MessageTyp) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

type ID string
