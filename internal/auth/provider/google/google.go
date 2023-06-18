package google

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/rapidmidiex/rmx/internal/auth/provider"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	"github.com/zitadel/oidc/v2/pkg/client/rs"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

var (
	issuer      = "https://accounts.google.com"
	authURI     = "/google"
	callbackURI = "/google/callback"
	scopes      = []string{"email", "profile", "openid"}
	iatOffset   = time.Second * 5
	urlParams   = []rp.URLParamOpt{
		rp.WithURLParam("access_type", "offline"),

		// prompt=consent forces google API to send a new refresh token on each login
		// https://stackoverflow.com/questions/10827920/not-receiving-google-oauth-refresh-token
		rp.WithURLParam("prompt", "consent"),
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

func (p *Provider) GetHandlers(baseURI string, callback rp.CodeExchangeUserinfoCallback[*oidc.IDTokenClaims]) (*provider.Handlers, error) {
	cookieHandler := httphelper.NewCookieHandler(
		p.hashKey,
		p.encKey,
		httphelper.WithUnsecure(),
		httphelper.WithSameSite(http.SameSiteLaxMode),
	)

	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(iatOffset)),
	}

	redirectURI := fmt.Sprintf("%s%s", baseURI, callbackURI)

	orp, err := rp.NewRelyingPartyOIDC(
		issuer,
		p.clientID,
		p.clientSecret,
		redirectURI,
		scopes,
		options...,
	)
	if err != nil {
		return nil, err
	}

	p.rp = orp

	/*
		ors, err := rs.NewResourceServerClientCredentials(issuer, p.clientID, p.clientSecret)
		if err != nil {
			return nil, err
		}

		p.rs = ors
	*/
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

func (p *Provider) getHandlers(callback rp.CodeExchangeUserinfoCallback[*oidc.IDTokenClaims]) (http.HandlerFunc, http.HandlerFunc, error) {
	state := func() string {
		return uuid.New().String()
	}

	return rp.AuthURLHandler(state, p.rp, urlParams...),
		rp.CodeExchangeHandler(rp.UserinfoCallback(callback), p.rp), nil
}

func (p *Provider) Introspect(ctx context.Context, session *auth.Session) (*oidc.IntrospectionResponse, error) {
	token, err := p.checkToken(session.AccessToken)
	if err != nil {
		return nil, err
	}

	return rs.Introspect(ctx, p.rs, token)
}

func (*Provider) checkToken(token string) (string, error) {
	if !strings.HasPrefix(token, oidc.PrefixBearer) {
		return "", fmt.Errorf("invalid token header")
	}

	return strings.TrimPrefix(token, oidc.PrefixBearer), nil
}
