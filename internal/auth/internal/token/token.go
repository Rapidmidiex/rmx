package token

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hyphengolang/prelude/types/suid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

type Claims struct {
	Issuer     string
	Audience   []string
	Email      string
	ClientID   string
	Expiration time.Time
}

type ParsedClaims struct {
	JwtID      string    `json:"jti"`
	Subject    string    `json:"sub"`
	Issuer     string    `json:"iss"`
	Audience   []string  `json:"aud"`
	IssuedAt   time.Time `json:"iat"`
	NotBefore  time.Time `json:"nbf"`
	Expiration time.Time `json:"exp"`
	Email      string    `json:"email"`
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

	bearer := fmt.Sprint(oidc.PrefixBearer, string(signed))

	return bearer, nil
}

func TrimPrefix(token string) string {
	return strings.TrimPrefix(token, oidc.PrefixBearer)
}
