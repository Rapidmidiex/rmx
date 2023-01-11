// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0

package db

import (
	"time"

	"github.com/google/uuid"
)

type Jam struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Bpm       int32     `json:"bpm"`
	Capacity  int32     `json:"capacity"`
	CreatedAt time.Time `json:"createdAt"`
}

type User struct {
	ID        uuid.UUID   `json:"id"`
	Username  string      `json:"username"`
	Email     interface{} `json:"email"`
	Password  interface{} `json:"password"`
	CreatedAt time.Time   `json:"createdAt"`
}
