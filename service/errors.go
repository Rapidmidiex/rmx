package service

import (
	"errors"
)

var (
	ErrNoCookie        = errors.New("api: cookie not found")
	ErrSessionNotFound = errors.New("api: session not found")
	ErrSessionExists   = errors.New("api: session already exists")
)
