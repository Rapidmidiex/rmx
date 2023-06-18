package auth

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	AccessTokenExp         = time.Minute * 30
	RefreshTokenExp        = time.Hour * 24 * 30
	RefreshTokenCookieName = "RMX_AUTH_RT"
)

type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
}

type Session struct {
	TokenType    string    `json:"tokenType"`
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	Expiry       time.Time `json:"expiry"`
}

type AuthError struct {
	StatusCode int   `json:"status"`
	Err        error `json:"err"`
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("status %d: err %v", e.StatusCode, e.Err)
}
