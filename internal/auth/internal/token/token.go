package token

import (
	"crypto/ecdsa"
	"encoding/json"
	"time"

	"github.com/hyphengolang/prelude/types/suid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type Claims struct {
	Issuer     string
	Audience   []string
	Email      string
	ClientID   string
	Expiration time.Time
}

func New(claims *Claims, key *ecdsa.PrivateKey) (string, error) {
	token, err := jwt.NewBuilder().
		JwtID(suid.NewSUID().String()).
		Issuer(claims.Issuer).
		Audience(claims.Audience).
		Subject(claims.ClientID).
		IssuedAt(time.Now().UTC()).
		NotBefore(time.Now().UTC()).
		Expiration(claims.Expiration).
		Claim("email", claims.Email).
		Build()

	if err != nil {
		return "", err
	}

	bs, err := json.Marshal(token)
	if err != nil {
		return "", err
	}

	signed, err := jws.Sign(bs, jws.WithKey(jwa.ES256, key))
	if err != nil {
		return "", err
	}

	return string(signed), nil
}
