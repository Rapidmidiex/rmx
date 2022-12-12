package auth

import (
	"context"
	"errors"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

var DefaultTokenClient = &client{make(map[string]bool), make(map[string]bool)}

func (c *client) ValidateRefreshToken(ctx context.Context, token string) error {
	return ErrNotImplemented
}

func (c *client) ValidateClientID(ctx context.Context, token string) error {
	return ErrNotImplemented
}

// BlackListClientID implements internal.TokenClient
func (c *client) BlackListClientID(ctx context.Context, cid string, email string) error {
	panic("unimplemented")
}

// BlackListRefreshToken implements internal.TokenClient
func (c *client) BlackListRefreshToken(ctx context.Context, token string) error {
	panic("unimplemented")
}

type client struct {
	mrt, mci map[string]bool
}

type Client struct {
	rtdb, cidb *redis.Client
}

// ValidateRefreshToken implements internal.TokenClient
func (*Client) ValidateRefreshToken(ctx context.Context, token string) error {
	panic("unimplemented")
}

// BlackListClientID implements internal.TokenClient
func (*Client) BlackListClientID(ctx context.Context, cid string, email string) error {
	panic("unimplemented")
}

// BlackListRefreshToken implements internal.TokenClient
func (*Client) BlackListRefreshToken(ctx context.Context, token string) error {
	panic("unimplemented")
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

// var DefaultClient = &Client{
// 	rtdb: redis.NewClient(&redis.Options{Addr: defaultAddr, Password: defaultPassword, DB: 0}),
// 	cidb: redis.NewClient(&redis.Options{Addr: defaultAddr, Password: defaultPassword, DB: 1}),
// }

func (c *Client) Validate(ctx context.Context, token string) error {
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

	err = c.RevokeClientID(ctx, cid, email)
	if err != nil {
		return err
	}

	return ErrRTValidate
}

func (c *Client) RevokeClientID(ctx context.Context, cid, email string) error {
	_, err := c.cidb.Set(ctx, cid, email, RefreshTokenExpiry).Result()
	return err
}

func (c *Client) RevokeRefreshToken(ctx context.Context, token string) error {
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

func ParseRefreshTokenClaims(token string) (jwt.Token, error) { return jwt.Parse([]byte(token)) }

const (
	RefreshTokenExpiry = time.Hour * 24 * 7
	AccessTokenExpiry  = time.Minute * 5
)
