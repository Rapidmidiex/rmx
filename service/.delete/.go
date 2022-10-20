package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/pkg/errors"
	"github.com/rog-golang-buddies/rmx/internal/fp"
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
	email, ok := tc.PrivateClaims()["email"].(string)
	if !ok {
		return ErrRTValidate
	}

	cid := tc.Subject()
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
	if err != nil {
		return err
	}
	return nil
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

/*
// would like to find an alternative to using `os` package
func LoadPEM(path string) (private, public jwk.Key, err error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return
	}

	return GenerateKeys(string(buf))
}
*/

func GenerateKeys() (jwk.Key, jwk.Key, error) {
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	key, err := jwk.FromRaw(private)
	if err != nil {
		return nil, nil, err
	}

	_, ok := key.(jwk.ECDSAPrivateKey)
	if !ok {
		return nil, nil, ErrGenerateKey
	}

	pub, err := key.PublicKey()
	if err != nil {
		return nil, nil, err
	}

	return key, pub, nil
}

func SignToken(key *jwk.Key, opt *TokenOption) ([]byte, error) {
	var t time.Time
	if opt.IssuedAt.IsZero() {
		t = time.Now().UTC()
	} else {
		t = opt.IssuedAt
	}

	token, err := jwt.NewBuilder().
		Issuer(opt.Issuer).
		Audience(opt.Audience).
		Subject(opt.Subject).
		IssuedAt(t).
		Expiration(t.Add(opt.Expiration)).
		Build()
	if err != nil {
		return nil, ErrSignTokens
	}

	for _, c := range opt.Claims {
		if !c.HasValue() {
			return nil, fp.ErrTuple
		}

		err := token.Set(c[0], c[1])
		if err != nil {
			return nil, ErrSignTokens
		}
	}

	return jwt.Sign(token, jwt.WithKey(jwa.ES256, key))
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
	Issuer     string
	Audience   []string
	Subject    string
	Claims     []fp.Tuple
	IssuedAt   time.Time
	Expiration time.Duration
}

type authCtxKey string

const (
	RefreshTokenCookieName = "RMX_REFRESH_TOKEN"
	RefreshTokenExpiry     = time.Hour * 24 * 7
	AccessTokenExpiry      = time.Minute * 5
	EmailKey               = authCtxKey("rmx-email")
)
