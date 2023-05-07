package auth

import (
	"fmt"

	"github.com/google/uuid"
)

type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
}

type AuthError struct {
	StatusCode int   `json:"status"`
	Err        error `json:"err"`
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("status %d: err %v", e.StatusCode, e.Err)
}
