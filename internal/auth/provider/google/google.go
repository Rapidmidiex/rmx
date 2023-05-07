package google

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rapidmidiex/rmx/internal/auth/provider"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

var (
	issuer      = "https://accounts.google.com"
	authURI     = "/google"
	callbackURI = "/google/callback"
	scopes      = []string{"email", "profile", "openid"}
	iatOffset   = time.Second * 5
)

type Provider struct {
	clientID, clientSecret string
	hashKey                []byte
	encKey                 []byte
}

func New(clientID, clientSecret string, hashKey, encKey []byte) provider.Provider {
	return &Provider{clientID, clientSecret, hashKey, encKey}
}

func (p *Provider) Init(baseURI string, callback rp.CodeExchangeUserinfoCallback[*oidc.IDTokenClaims]) (*provider.Handlers, error) {
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

	// static port number just for testing
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

	ah, ch, err := initHandlers(orp, callback)
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

func initHandlers(
	provider rp.RelyingParty,
	callback rp.CodeExchangeUserinfoCallback[*oidc.IDTokenClaims],
) (http.HandlerFunc, http.HandlerFunc, error) {
	state := func() string {
		return uuid.New().String()
	}

	return rp.AuthURLHandler(state, provider),
		rp.CodeExchangeHandler(rp.UserinfoCallback(callback), provider), nil
}
