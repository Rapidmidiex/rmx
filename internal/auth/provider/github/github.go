package github

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/go-github/v51/github"
	"github.com/google/uuid"
	"github.com/rapidmidiex/rmx/internal/auth"
	"golang.org/x/oauth2"

	ghOAuth "golang.org/x/oauth2/github"

	"github.com/zitadel/oidc/v2/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

var (
	key = []byte("test1234test1234")

	authURI     = "/github"
	callbackURI = "/github/callback"
)

func NewGithub(cfg *auth.ProviderCfg) (*auth.Provider, error) {
	ah, ch, err := initProvider(cfg.ClientID, cfg.ClientSecret)
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

func initProvider(clientID, clientSecret string) (ah http.HandlerFunc, ch http.HandlerFunc, err error) {
	rpConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  fmt.Sprintf("http://localhost:8000/v0/auth%v", callbackURI),
		Scopes:       []string{string(github.ScopeNone)},
		Endpoint:     ghOAuth.Endpoint,
	}

	cookieHandler := httphelper.NewCookieHandler(key, key, httphelper.WithUnsecure())
	provider, err := rp.NewRelyingPartyOAuth(rpConfig, rp.WithCookieHandler(cookieHandler))
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
		data, err := json.Marshal(tokens)
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
