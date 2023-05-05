package google

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rapidmidiex/rmx/internal/auth"
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

func New(
	cfg *provider.ProviderCfg,
	callback rp.CodeExchangeUserinfoCallback[*oidc.IDTokenClaims],
) (*auth.Provider, error) {
	cookieHandler := httphelper.NewCookieHandler(
		cfg.HashKey,
		cfg.EncKey,
		httphelper.WithUnsecure(),
		httphelper.WithSameSite(http.SameSiteLaxMode),
	)
	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(iatOffset)),
	}

	redirectURI := fmt.Sprintf("%s%s", cfg.BaseURI, callbackURI)

	// static port number just for testing
	provider, err := rp.NewRelyingPartyOIDC(
		issuer,
		cfg.ClientID,
		cfg.ClientSecret,
		redirectURI,
		scopes,
		options...,
	)
	if err != nil {
		return nil, err
	}

	ah, ch, err := initProvider(provider, callback)
	if err != nil {
		return nil, err
	}

	return &auth.Provider{
		AuthHandler:     ah,
		CallbackHandler: ch,
		AuthURI:         authURI,
		CallbackURI:     callbackURI,
	}, nil
}

func initProvider(
	provider rp.RelyingParty,
	callback rp.CodeExchangeUserinfoCallback[*oidc.IDTokenClaims],
) (http.HandlerFunc, http.HandlerFunc, error) {
	state := func() string {
		return uuid.New().String()
	}

	return rp.AuthURLHandler(state, provider),
		rp.CodeExchangeHandler(rp.UserinfoCallback(callback), provider), nil
}
