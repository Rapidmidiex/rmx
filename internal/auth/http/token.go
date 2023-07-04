package http

import (
	"encoding/json"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func (s *Service) newToken(issuer, sid, email string, exp time.Duration) (string, error) {
	// TODO: check if it's ok to set sid as JwtID
	token, err := jwt.NewBuilder().
		JwtID(sid).
		Issuer(issuer).
		Audience([]string{"web"}).
		Subject(email).
		IssuedAt(time.Now().UTC()).
		NotBefore(time.Now().UTC()).
		Expiration(time.Now().UTC().Add(exp)).
		Build()
	if err != nil {
		return "", err
	}

	bs, err := json.Marshal(token)
	if err != nil {
		return "", err
	}

	signed, err := jws.Sign(bs, jws.WithKey(jwa.ES256, s.keyPair.PrivateKey))
	if err != nil {
		return "", err
	}

	return string(signed), nil
}

func (s *Service) parseToken(token string) (jwt.Token, error) {
	return jwt.Parse([]byte(token), jwt.WithKey(jwa.ES256, s.keyPair.PublicKey))
}
