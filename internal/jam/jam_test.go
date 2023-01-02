package jam_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rapidmidiex/rmx/internal/jam"

	"github.com/hyphengolang/prelude/testing/is"
)

func TestUnmarshalJam(t *testing.T) {
	is := is.New(t)

	t.Run("unmarshal jam session", func(t *testing.T) {
		payload := `
		{
			"name":     "John Doe",
			"capacity": 5,
			"bpm":      100
		}`

		var j jam.Jam
		err := json.NewDecoder(strings.NewReader(payload)).Decode(&j)
		is.NoErr(err)                 // decoding passed
		is.Equal(j.Capacity, uint(5)) // capacity is 5
		is.Equal(j.BPM, uint(100))    // bpm is 100
	})

	t.Run("unmarshal with defaults", func(t *testing.T) {
		payload := `
		{
			"name": "John Doe"
		}`

		var j jam.Jam
		err := json.NewDecoder(strings.NewReader(payload)).Decode(&j)
		is.NoErr(err)                  // decoding passed
		is.Equal(j.BPM, uint(80))      // default BPM
		is.Equal(j.Capacity, uint(10)) // default capacity
	})
}

func TestMarshalJam(t *testing.T) {}
