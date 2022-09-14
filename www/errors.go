package www

import (
	"errors"
)

var (
	ErrNoCookie        = errors.New("www: cookie not found")
	ErrSessionNotFound = errors.New("www: session not found")
	ErrSessionExists   = errors.New("www: session already exists")
)
