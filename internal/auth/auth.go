package auth

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	natsSubj           = "rmx.auth"
	natsSessionSufx    = ".session"
	natsIntrospectSufx = ".introspect"
)

var (
	RefreshTokenExp = time.Hour * 24 * 30
)

type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
}

type Session struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type AuthError struct {
	StatusCode int   `json:"status"`
	Err        error `json:"err"`
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("status %d: err %v", e.StatusCode, e.Err)
}
