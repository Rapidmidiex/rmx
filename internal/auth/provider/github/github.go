package github

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/rapidmidiex/rmx/internal/auth/provider"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	"github.com/zitadel/oidc/v2/pkg/client/rs"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
	"golang.org/x/oauth2"
	githubOAuth "golang.org/x/oauth2/github"
)

var (
	issuer      = githubOAuth.Endpoint.AuthURL
	authURI     = "/github"
	callbackURI = "/github/callback"
	scopes      = []string{"(no scope)"}
	iatOffset   = time.Second * 5
	urlParams   = []rp.URLParamOpt{
		rp.WithURLParam("grant_type", "refresh_token"),
	}
)

type Provider struct {
	issuer string
	rp     rp.RelyingParty
	rs     rs.ResourceServer

	clientID, clientSecret string
	hashKey                []byte
	encKey                 []byte
}

func New(clientID, clientSecret string, hashKey, encKey []byte) provider.Provider {
	return &Provider{issuer, nil, nil, clientID, clientSecret, hashKey, encKey}
}

func (p *Provider) Issuer() string {
	return p.issuer
}

func (p *Provider) GetHandlers(baseURI string, callback rp.CodeExchangeCallback[*oidc.IDTokenClaims]) (*provider.Handlers, error) {
	cookieHandler := httphelper.NewCookieHandler(
		p.hashKey,
		p.encKey,
		httphelper.WithUnsecure(),
		httphelper.WithSameSite(http.SameSiteLaxMode),
	)

	redirectURI := fmt.Sprintf("%s%s", baseURI, callbackURI)

	rpConfig := &oauth2.Config{
		ClientID:     p.clientID,
		ClientSecret: p.clientSecret,
		RedirectURL:  redirectURI,
		Scopes:       scopes,
		Endpoint:     githubOAuth.Endpoint,
	}

	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(iatOffset)),
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

	return &provider.Handlers{
		AuthHandler:     ah,
		CallbackHandler: ch,
		AuthURI:         authURI,
		CallbackURI:     callbackURI,
	}, nil
}

func (p *Provider) getHandlers(callback rp.CodeExchangeCallback[*oidc.IDTokenClaims]) (http.HandlerFunc, http.HandlerFunc, error) {
	state := func() string {
		return uuid.New().String()
	}

	return rp.AuthURLHandler(state, p.rp, urlParams...),
		rp.CodeExchangeHandler(rp.CodeExchangeCallback[*oidc.IDTokenClaims](callback2), p.rp), nil
}

func callback2(w http.ResponseWriter, r *http.Request, tokens *oidc.Tokens[*oidc.IDTokenClaims], state string, rp rp.RelyingParty) {
	fmt.Printf("%v", tokens.AccessToken)
}

func (p *Provider) Introspect(ctx context.Context, session *auth.Session) (*oidc.IntrospectionResponse, error) {
	return rs.Introspect(ctx, p.rs, "token")
}
