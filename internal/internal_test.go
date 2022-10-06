package internal

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rog-golang-buddies/rmx/internal/is"
)

func TestCustomTypes(t *testing.T) {
	t.Parallel()

	is := is.New(t)

	t.Run(`using "encoding/json" package with password`, func(t *testing.T) {
		payload := `"this_password_is_complex"`

		var p Password
		err := json.NewDecoder(strings.NewReader(payload)).Decode(&p)
		is.NoErr(err) // parse password

		h, err := p.Hash()
		is.NoErr(err) // hash password

		err = h.Compare(payload[1 : len(payload)-1])
		is.NoErr(err) // valid password
	})

	t.Run(`using "encoding/json" package with email`, func(t *testing.T) {
		payload := `"fizz@mail.com"`

		var e Email
		err := json.NewDecoder(strings.NewReader(payload)).Decode(&e)
		is.NoErr(err) // parse email
	})
}
