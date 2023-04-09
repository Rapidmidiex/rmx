package google

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

var (
	key = []byte("test1234test1234")

	issuer      = "https://accounts.google.com"
	authURI     = "/google"
	callbackURI = "/google/callback"
	scopes      = []string{"email", "profile", "openid"}
)

func NewGoogle(router chi.Router, clientID, clientSecret string) (*auth.Provider, error) {
	ah, ch, err := initProvider(clientID, clientSecret)
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
	clientID,
	clientSecret string,
) (ah http.HandlerFunc, ch http.HandlerFunc, err error) {
	cookieHandler := httphelper.NewCookieHandler(key, key, httphelper.WithUnsecure(), httphelper.WithSameSite(http.SameSiteLaxMode))
	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(5 * time.Second)),
		rp.WithPKCE(cookieHandler),
	}

	redirectURI := fmt.Sprintf("http://localhost:9999/auth%v", callbackURI)

	// static port number just for testing
	provider, err := rp.NewRelyingPartyOIDC(
		issuer,
		clientID,
		clientSecret,
		redirectURI,
		scopes,
		options...,
	)

	if err != nil {
		return
	}

	state := func() string {
		return uuid.New().String()
	}

	marshalUserinfo := func(
		w http.ResponseWriter,
		r *http.Request,
		tokens *oidc.Tokens[*oidc.IDTokenClaims],
		state string,
		rp rp.RelyingParty,
		info *oidc.UserInfo,
	) {
		data, err := json.Marshal(info)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(data)
	}

	ah = rp.AuthURLHandler(state, provider, rp.WithPromptURLParam("Welcome back!"))
	ch = rp.CodeExchangeHandler(rp.UserinfoCallback(marshalUserinfo), provider)

	return
}
