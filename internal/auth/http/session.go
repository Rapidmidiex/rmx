package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/rapidmidiex/rmx/internal/auth"
)

func (s *Service) genTokens(issuer, sid, email string) (string, string, error) {
	accessToken, err := jwt.NewBuilder().
		JwtID(uuid.NewString()).
		Issuer(issuer).
		Audience([]string{"web"}).
		Subject(sid).
		IssuedAt(time.Now().UTC()).
		NotBefore(time.Now().UTC()).
		Expiration(time.Now().UTC().Add(auth.AccessTokenExp)).
		Claim("email", email).
		Build()
	if err != nil {
		return "", "", err
	}

	bs, err := json.Marshal(accessToken)
	if err != nil {
		return "", "", err
	}

	atSigned, err := jws.Sign(bs, jws.WithKey(jwa.ES256, s.keyPair.PrivateKey))
	if err != nil {
		return "", "", err
	}

	refreshToken, err := jwt.NewBuilder().
		JwtID(uuid.NewString()).
		Issuer(issuer).
		Audience([]string{"web"}).
		Subject(sid).
		IssuedAt(time.Now().UTC()).
		NotBefore(time.Now().UTC()).
		Expiration(time.Now().UTC().Add(auth.RefreshTokenExp)).
		Claim("email", email).
		Build()
	if err != nil {
		return "", "", err
	}

	bs, err = json.Marshal(refreshToken)
	if err != nil {
		return "", "", err
	}

	rtSigned, err := jws.Sign(bs, jws.WithKey(jwa.ES256, s.keyPair.PrivateKey))
	if err != nil {
		return "", "", err
	}

	return string(atSigned), string(rtSigned), nil
}

func (s *Service) parseSession(r *http.Request) (jwt.Token, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}

	return jwt.Parse([]byte(cookie.Value), jwt.WithKey(jwa.ES256, s.keyPair.PublicKey))
}
