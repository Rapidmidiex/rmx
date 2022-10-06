package auth

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

var rsaPrivateKey = `-----BEGIN PRIVATE KEY-----
MIICdgIBADANBgkqhkiG9w0BAQEFAASCAmAwggJcAgEAAoGBAML5MHFgqUlZcENS
hHZ83yXfoUpqaMfp5/UdgMIJ0S5DW5QEON6reAsDu6zP0BEVZhg65pEYWEraBrGK
Vcbx7dsVqK4Z0GMm0YRAvB+1K+pYlXwld90mwG1TqOKDPQXqC0Z/jZi6DSsAhfJU
WN0rkInZRtoVeRzbbh+nLN8nd14fAgMBAAECgYEAor+A2VL3XBvFIt0RZxpq5mFa
cBSMrDsqfSeIX+/z5SsimVZA5lW5GXCfSuwY4Pm8xAL+jSUGJk0CA1bWrP8rLByS
cQAy1q0odaAiWIG5zFUEQBg5Q5b3+jXmh2zwtO7yhPuXn1/vBGg+FvyR57gV/3F+
TuBfR6Bc3VWKuj7Gm5kCQQDuRgm8HTDbX7IQ0EFAVKB73Pj4Gx5u2NieD9U8+qXx
JsAdn1vRvQ3mNJDR5OcTr4rPkpLLCtzjA2iTDXp4yqmrAkEA0Xp91LCpImKAOtM3
4SGXdzKi9+7fWmcTtfkz996y9A1C9l27Cj92P7OFdwMB4Z/ZMizJd0eXYhXr4IxH
wBoxXQJAUBOXp/HDfqZdiIsEsuL+AEKWJYOvqZ8UxaIajuDJrg7Q1+O7jvRTXH9k
ADZGdnYzV2kyDiy7aUu29Fy+QSQS+wJAJyEsdBhz35pqvZJK8+DkfD2XN50FV8u9
YNamIH0XDIOVqJOlpqpoGkocejizl0PWvIqlL4TOAGJ75zwNAxNheQJABEA2/hfF
GMJsOrnD74rGP/Lfpg882AmeUoT5eH766sSobFfUYJZvyAmnQoK2Lzg2hrKwXXix
JvEGfrhihVLb7g==
-----END PRIVATE KEY-----
`

func TestJWT(t *testing.T) {
	// -- Parse, Serialize JSON Web Key --
	jwkPrivate, err := jwk.ParseKey([]byte(rsaPrivateKey), jwk.WithPEM(true))
	if err != nil {
		t.Fatalf("failed to parse JWK: %s\n", err)
	}

	jwtPublic, err := jwk.PublicKeyOf(jwkPrivate)
	if err != nil {
		t.Fatalf("failed to get public key: %s\n", err)
	}
	// -- Parse, Serialize --

	// -- Working with JSON Web Tokens --
	// build (server boot?)
	token, err := jwt.NewBuilder().Build()
	if err != nil {
		t.Fatalf("failed to get public key: %s\n", err)
	}

	// sign (server boot?)
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, jwkPrivate))
	if err != nil {
		t.Fatalf("failed to sign token: %s\n", err)
		return
	}

	// verify (every endpoint hit)
	// verified, err := jwt.Parse(signed, jwt.WithKey(jwa.RS256, jwtPublic))
	// if err != nil {
	// 	fmt.Printf("failed to verify JWS: %s\n", err)
	// 	return
	// }
	// _ = verified

	req, err := http.NewRequest(http.MethodGet, `http://localhost:8080`, nil)
	req.Header.Set(`Authorization`, fmt.Sprintf(`Bearer %s`, signed))
	if err != nil {
		t.Fatalf("failed to create HTTP request: %s\n", err)
		return
	}

	verifiedToken, err := jwt.ParseRequest(req, jwt.WithKey(jwa.RS256, jwtPublic))
	if err != nil {
		t.Fatalf("failed to verify token from HTTP request: %s\n", err)
		return
	}

	_ = verifiedToken
	// -- Working with htt.Request --
}

func TestCerts(t *testing.T) {
	t.Parallel()

	// c := `../../certs/cert.pem`
	k := `../../certs/key.pem`
	buf, err := os.ReadFile(k)
	if err != nil {
		t.Fatalf("failed to read file %s\n", err)
	}

	raw, _, err := jwk.DecodePEM(buf)
	if err != nil {
		t.Fatalf("failed to decode PEM key: %s\n", err)
	}

	private, err := jwk.FromRaw(raw)
	if err != nil {
		t.Fatalf("failed to create private key")
	}

	_ = private
}
