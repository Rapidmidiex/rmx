package jam_test

import (
	"encoding/json"
	"log"
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
		is.Equal(j.BPM, uint(120))     // default BPM
		is.Equal(j.Capacity, uint(10)) // default capacity
	})
}

func TestCapacity(t *testing.T) {
	is := is.New(t)

	t.Run("capacity from json payload", func(t *testing.T) {
		type testcase struct {
			name     string
			input    string
			expected uint
			err      error
		}

		tt := []testcase{
			{
				name:     "capacity is 5",
				input:    `{ "capacity": 5 }`,
				expected: 5,
			},
			{
				name:     "capacity is 10",
				input:    `{ "capacity": 10 }`,
				expected: 10,
			},
			{
				name:     "default capacity",
				input:    `{ "capacity": 0 }`,
				expected: 10,
			},
			{
				name:     "implicit default capacity",
				input:    `{  }`,
				expected: 10,
			},
			{
				name:  "invalid capacity",
				input: `{ "capacity": 1 }`,
				// expected: 10, // should error
			},
		}

		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				type payload struct {
					Capacity jam.Capacity `json:"capacity"`
				}

				var p payload
				err := json.NewDecoder(strings.NewReader(tc.input)).Decode(&p)

				log.Println(p)

				if err != nil {
					is.Equal(err, tc.err) // decoding payload
					return
				}

				is.Equal(jam.Capacity(tc.expected), p.Capacity) // capacity
			})
		}

	})
}
