package auth

import (
	"fmt"

	"github.com/google/uuid"
)

const (
	natsSubj           = "rmx.auth"
	natsSessionSufx    = ".session"
	natsIntrospectSufx = ".introspect"
)

type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
}

type Tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type Session struct {
	ClientID string `json:"clientId"`
	Issuer   string `json:"issuer"`
	Tokens   Tokens `json:"tokens"`
}

type AuthError struct {
	StatusCode int   `json:"status"`
	Err        error `json:"err"`
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("status %d: err %v", e.StatusCode, e.Err)
}
