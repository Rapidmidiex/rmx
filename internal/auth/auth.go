package auth

import (
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	AccessTokenExp         = time.Minute * 30
	RefreshTokenExp        = time.Hour * 24 * 30
	RefreshTokenCookieName = "RMX_AUTH_RT"
)

type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

type OAuthUserInfo struct {
	Username string
	Email    string
}

type Session struct {
	Provider    string `json:"provider"`
	SessionInfo string `json:"sessionInfo"`
}

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
