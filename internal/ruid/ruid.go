package ruid

import (
	"crypto/rand"
	"io"
)

type RUID string

var table = [...]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}

func New(length int) (string, error) {
	b := make([]byte, length)
	n, err := io.ReadAtLeast(rand.Reader, b, length)
	if n != length {
		return "", err
	}
	for i := 0; i < len(b); i++ {
		b[i] = table[int(b[i])%len(table)]
	}
	return string(b), nil
}

func (r RUID) String() string { return string(r) }
