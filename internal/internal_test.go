package internal

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hyphengolang/prelude/testing/is"
	"github.com/hyphengolang/prelude/types/email"
	"github.com/hyphengolang/prelude/types/password"
)

func TestCustomTypes(t *testing.T) {
	t.Parallel()

	is := is.New(t)

	t.Run(`using "encoding/json" package with password`, func(t *testing.T) {
		payload := `"this_password_is_complex"`

		var p password.Password
		err := json.NewDecoder(strings.NewReader(payload)).Decode(&p)
		is.NoErr(err) // parse password

		h, err := p.Hash()
		is.NoErr(err) // hash password

		err = h.Compare(payload[1 : len(payload)-1])
		is.NoErr(err) // valid password
	})

	t.Run(`using "encoding/json" package with email`, func(t *testing.T) {
		payload := `"fizz@mail.com"`

		var e email.Email
		err := json.NewDecoder(strings.NewReader(payload)).Decode(&e)
		is.NoErr(err) // parse email
	})
}
