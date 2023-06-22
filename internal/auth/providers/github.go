package providers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	githubClient "github.com/google/go-github/v53/github"
	"github.com/google/uuid"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	"github.com/zitadel/oidc/v2/pkg/client/rs"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
	"golang.org/x/oauth2"
	githubOAuth "golang.org/x/oauth2/github"
)

var (
	githubAuthURI     = "/github"
	githubCallbackURI = "/github/callback"
	githubScopes      = []string{string(githubClient.ScopeUserEmail)}
	githubIATOffset   = time.Second * 5
	githubUrlParams   = []rp.URLParamOpt{
		rp.WithURLParam("grant_type", "refresh_token"),
	}
)

type GithubProvider struct {
	issuer                 string
	authType               auth.AuthType
	clientID, clientSecret string
	hashKey                []byte
	encKey                 []byte
	rp                     rp.RelyingParty
	rs                     rs.ResourceServer
}

func NewGithub(clientID, clientSecret string, hashKey, encKey []byte) (Provider, error) {
	parsed, err := url.Parse(githubOAuth.Endpoint.AuthURL)
	if err != nil {
		return nil, err
	}
	return &GithubProvider{fmt.Sprint("https://", parsed.Hostname()), auth.OAuth, clientID, clientSecret, hashKey, encKey, nil, nil}, nil
}

func (p *GithubProvider) GetIssuer() string {
	return p.issuer
}

func (p *GithubProvider) GetAuthType() auth.AuthType {
	return p.authType
}

func (p *GithubProvider) GetHandlers(baseURI string, callback rp.CodeExchangeCallback[*oidc.IDTokenClaims]) (*Handlers, error) {
	cookieHandler := httphelper.NewCookieHandler(
		p.hashKey,
		p.encKey,
		httphelper.WithUnsecure(),
		httphelper.WithSameSite(http.SameSiteLaxMode),
	)

	redirectURI := fmt.Sprintf("%s%s", baseURI, githubCallbackURI)

	rpConfig := &oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		RedirectURL:  redirectURI,
		Scopes:       githubScopes,
		Endpoint:     githubOAuth.Endpoint,
	}

	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(githubIATOffset)),
	}

	orp, err := rp.NewRelyingPartyOAuth(rpConfig, options...)
	if err != nil {
		return nil, err
	}

	p.rp = orp

	ah, ch, err := p.getHandlers(callback)
	if err != nil {
		return nil, err
	}

	return &Handlers{
		AuthHandler:     ah,
		CallbackHandler: ch,
		AuthURI:         githubAuthURI,
		CallbackURI:     githubCallbackURI,
	}, nil
}

func (p *GithubProvider) getHandlers(callback rp.CodeExchangeCallback[*oidc.IDTokenClaims]) (http.HandlerFunc, http.HandlerFunc, error) {
	state := func() string {
		return uuid.New().String()
	}

	return rp.AuthURLHandler(state, p.rp, githubUrlParams...),
		rp.CodeExchangeHandler(rp.CodeExchangeCallback[*oidc.IDTokenClaims](callback), p.rp), nil
}

func (p *GithubProvider) Introspect(ctx context.Context, session *auth.Session) (*oidc.IntrospectionResponse, error) {
	return rs.Introspect(ctx, p.rs, "token")
}

func (p *GithubProvider) UserInfo(ctx context.Context, token string) (*auth.OAuthUserInfo, error) {
	client := githubClient.NewTokenClient(ctx, token)
	emails, _, err := client.Users.ListEmails(ctx, nil)
	if err != nil {
		return nil, err
	}

	var email string
	for _, e := range emails {
		if e.Email != nil && *e.Primary {
			email = *e.Email
		}
	}

	// empty string to get authorized user info
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return nil, err
	}

	if user.Login != nil && email != "" {
		return &auth.OAuthUserInfo{Username: *user.Login, Email: email}, nil
	}

	return nil, errors.New("rmx: github UserInfo invalid user.Login value")
}
