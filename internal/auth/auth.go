package auth

import (
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	AccessTokenExp  = time.Minute * 30
	RefreshTokenExp = time.Hour * 24 * 30
)

type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

type OAuthUserInfo struct {
	Username string
	Email    string
}

type User struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
}

type Error struct {
	StatusCode int    `json:"status"`
	Err        error  `json:"err"`
	Text       string `json:"text"`
}

func (e Error) Error() string {
	return fmt.Sprintf("status %d: err %v", e.StatusCode, e.Err)
}
