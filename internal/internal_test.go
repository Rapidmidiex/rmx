package internal

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rog-golang-buddies/rmx/internal/is"
)

func TestPassword(t *testing.T) {
	t.Parallel()

	is := is.New(t)

	t.Run("(en/de)coding password type", func(t *testing.T) {
		payload := `"thispasswordiscomplex"`

		var pws Password
		err := json.NewDecoder(strings.NewReader(payload)).Decode(&pws)
		is.NoErr(err)

		pw, err := pws.Hash()
		is.NoErr(err) // hash password

		err = pw.Compare(payload[1 : len(payload)-1])
		is.NoErr(err) // valid password
	})
}
