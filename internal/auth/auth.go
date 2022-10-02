package auth

import (
	"context"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/pkg/errors"
	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/fp"
	"github.com/rog-golang-buddies/rmx/internal/suid"
)

var ErrNotImplemented = errors.New("not implemented")

type Client struct {
	rtdb, cidb *redis.Client
}

func New(addr, password string) *Client {
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

func (c *Client) SetRefresh(ctx context.Context, token string, exp time.Duration) error {

	_, err := c.rtdb.Set(ctx, token, nil, exp).Result()
	return err
}

func (c *Client) HasTokenUsed(ctx context.Context, token string) bool {
	// check if token is available in redis database
	// if it's not then token is not reused
	_, err := c.cidb.Get(ctx, token).Result()
	return err == nil
}

func (c *Client) SetClientID(ctx context.Context, id suid.UUID, email internal.Email, exp time.Duration) error {
	_, err := c.cidb.Set(ctx, id.String(), email, exp).Result()
	return err
}

func (c *Client) HasClientID(ctx context.Context, id suid.UUID) bool {
	// check if a key with client id exists
	// if the key exists it means that the client id is revoked and token should be denied
	// we don't need the email value here
	_, err := c.cidb.Get(ctx, id.String()).Result()
	return err == nil
}

// isTokenUsed

// saveRefreshToken
func (c *Client) SaveRefreshToken() error {
	return ErrNotImplemented
}

func GenerateKeys(secret string) (private, public jwk.Key, err error) {
	if private, err = jwk.ParseKey([]byte(secret), jwk.WithPEM(true)); err != nil {
		return nil, nil, err
	}

	if public, err = private.PublicKey(); err != nil {
		return nil, nil, err
	}

	return private, public, nil
}

func SignToken(key jwk.Key, opt *TokenOption) ([]byte, error) {
	if !opt.Claim.HasValue() {
		return nil, fp.ErrTuple
	}

	var t time.Time
	if opt.IssuedAt.IsZero() {
		t = time.Now()
	} else {
		t = opt.IssuedAt
	}

	token, _ := jwt.NewBuilder().
		Issuer(opt.Issuer).
		Audience(opt.Audience).
		Subject(opt.Subject).
		Claim(opt.Claim[0], opt.Claim[1]).
		IssuedAt(t).
		Expiration(t.Add(opt.Expiration)).
		Build()

	return jwt.Sign(token, jwt.WithKey(jwa.RS256, key))
}

type TokenOption struct {
	Issuer     string
	Audience   []string
	Subject    string
	Claim      fp.Tuple
	IssuedAt   time.Time
	Expiration time.Duration
}
