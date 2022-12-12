package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/pkg/errors"
)

type Client struct {
	rtdb, cidb *redis.Client
}

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrGenerateKey    = errors.New("failed to generate new ecdsa key pair")
	ErrSignTokens     = errors.New("failed to generate signed tokens")
	ErrRTValidate     = errors.New("failed to validate refresh token")
)

func NewRedis(addr, password string) *Client {
	rtdb := redis.Options{Addr: addr, Password: password, DB: 0}
	cidb := redis.Options{Addr: addr, Password: password, DB: 1}

	c := &Client{redis.NewClient(&rtdb), redis.NewClient(&cidb)}
	return c
}

const (
	defaultAddr     = "localhost:6379"
	defaultPassword = ""
)

var DefaultClient = &Client{
	rtdb: redis.NewClient(&redis.Options{Addr: defaultAddr, Password: defaultPassword, DB: 0}),
	cidb: redis.NewClient(&redis.Options{Addr: defaultAddr, Password: defaultPassword, DB: 1}),
}

func (c *Client) ValidateRefreshToken(ctx context.Context, token string) error {
	tc, err := ParseRefreshTokenClaims(token)
	if err != nil {
		return err
	}

	cid := tc.Subject()
	email, ok := tc.PrivateClaims()["email"].(string)
	if !ok {
		return ErrRTValidate
	}

	if err := c.ValidateClientID(ctx, cid); err != nil {
		return err
	}

	if _, err := c.rtdb.Get(ctx, token).Result(); err != nil {
		switch err {
		case redis.Nil:
			return nil
		default:
			return err
		}
	}

	err = c.BlackListClientID(ctx, cid, email)
	if err != nil {
		return err
	}

	return ErrRTValidate
}

func (c *Client) BlackListClientID(ctx context.Context, cid, email string) error {
	_, err := c.cidb.Set(ctx, cid, email, RefreshTokenExpiry).Result()
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) BlackListRefreshToken(ctx context.Context, token string) error {
	_, err := c.rtdb.Set(ctx, token, nil, RefreshTokenExpiry).Result()
	return err
}

func (c *Client) ValidateClientID(ctx context.Context, cid string) error {
	// check if a key with client id exists
	// if the key exists it means that the client id is revoked and token should be denied
	// we don't need the email value here
	_, err := c.cidb.Get(ctx, cid).Result()
	if err != nil {
		switch err {
		case redis.Nil:
			return nil
		default:
			return ErrRTValidate
		}
	}

	return ErrRTValidate
}

func ES256() (public, private jwk.Key) {
	raw, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	private, err = jwk.FromRaw(raw)
	if err != nil {
		panic(err)
	}

	public, err = private.PublicKey()
	if err != nil {
		panic(err)
	}

	return
}

func RS256() (public, private jwk.Key) {
	raw, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	if private, err = jwk.FromRaw(raw); err != nil {
		panic(err)
	}

	if public, err = private.PublicKey(); err != nil {
		panic(err)
	}

	return
}

func Sign(key jwk.Key, o *TokenOption) ([]byte, error) {
	var sep jwt.SignEncryptParseOption
	switch key := key.(type) {
	case jwk.RSAPrivateKey:
		sep = jwt.WithKey(jwa.RS256, key)
	case jwk.ECDSAPrivateKey:
		sep = jwt.WithKey(jwa.ES256, key)
	default:
		return nil, errors.New(`unsupported encryption`)
	}

	var iat time.Time
	if o.IssuedAt.IsZero() {
		iat = time.Now().UTC()
	} else {
		iat = o.IssuedAt
	}

	tk, err := jwt.NewBuilder().
		Issuer(o.Issuer).
		Audience(o.Audience).
		Subject(o.Subject).
		IssuedAt(iat).
		Expiration(iat.Add(o.Expiration)).
		Build()

	if err != nil {
		return nil, ErrSignTokens
	}

	for k, v := range o.Claims {
		if err := tk.Set(k, v); err != nil {
			return nil, err
		}
	}

	return jwt.Sign(tk, sep)
}

func ParseCookie(r *http.Request, key jwk.Key, cookieName string) (jwt.Token, error) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return nil, err
	}

	var sep jwt.SignEncryptParseOption
	switch key := key.(type) {
	case jwk.RSAPublicKey:
		sep = jwt.WithKey(jwa.RS256, key)
	case jwk.ECDSAPublicKey:
		sep = jwt.WithKey(jwa.ES256, key)
	default:
		return nil, errors.New(`unsupported encryption`)
	}

	return jwt.Parse([]byte(c.Value), sep)
}

/*
ParseRequest searches a http.Request object for a JWT token.

Specifying WithHeaderKey() will tell it to search under a specific
header key. Specifying WithFormKey() will tell it to search under
a specific form field.

By default, "Authorization" header will be searched.

If WithHeaderKey() is used, you must explicitly re-enable searching for "Authorization" header.

	# searches for "Authorization"
	jwt.ParseRequest(req)

	# searches for "x-my-token" ONLY.
	jwt.ParseRequest(req, jwt.WithHeaderKey("x-my-token"))

	# searches for "Authorization" AND "x-my-token"
	jwt.ParseRequest(req, jwt.WithHeaderKey("Authorization"), jwt.WithHeaderKey("x-my-token"))
*/
func ParseRequest(r *http.Request, key jwk.Key) (jwt.Token, error) {
	var sep jwt.SignEncryptParseOption
	switch key := key.(type) {
	case jwk.RSAPublicKey:
		sep = jwt.WithKey(jwa.RS256, key)
	case jwk.ECDSAPublicKey:
		sep = jwt.WithKey(jwa.ES256, key)
	default:
		return nil, errors.New(`unsupported encryption`)
	}
	return jwt.ParseRequest(r, sep)
}

func ParseRefreshTokenClaims(token string) (jwt.Token, error) { return jwt.Parse([]byte(token)) }

func ParseRefreshTokenWithValidate(key *jwk.Key, token string) (jwt.Token, error) {
	payload, err := jwt.Parse([]byte(token),
		jwt.WithKey(jwa.ES256, key),
		jwt.WithValidate(true))
	if err != nil {
		return nil, err
	}

	return payload, nil
}

type TokenOption struct {
	IssuedAt   time.Time
	Issuer     string
	Audience   []string
	Subject    string
	Expiration time.Duration
	Claims     map[string]any
}

type authCtxKey string

const (
	// RefreshTokenCookieName = "RMX_REFRESH_TOKEN"
	RefreshTokenExpiry = time.Hour * 24 * 7
	AccessTokenExpiry  = time.Minute * 5
	EmailKey           = authCtxKey("rmx-email")
)
