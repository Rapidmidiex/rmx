package auth

import (
	"fmt"
	"net/http"

	"encoding/json"

	"github.com/go-chi/chi/v5"

	"time"

	"github.com/google/uuid"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

var (
	key = []byte("test1234test1234")
)

type Service struct {
	Handlers    *authHandlers
	AuthURI     string
	CallbackURI string

	router chi.Router
}

type authHandlers struct {
	AuthHandler     http.HandlerFunc
	CallbackHandler http.HandlerFunc
}

type providerCfg struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	Scopes       []string
	CallbackURI  string
}

func NewWithProvider(
	issuer,
	clientID,
	clientSecret,
	callbackURI string,
	scopes []string,
) (*authHandlers, error) {
	cookieHandler := httphelper.NewCookieHandler(key, key, httphelper.WithUnsecure(), httphelper.WithSameSite(http.SameSiteLaxMode))
	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(5 * time.Second)),
	}

	if clientSecret == "" {
		options = append(options, rp.WithPKCE(cookieHandler))
	}

	// static port number just for testing
	provider, err := rp.NewRelyingPartyOIDC(
		issuer,
		clientID,
		clientSecret,
		fmt.Sprintf("http://localhost:9999/auth/%v", callbackURI),
		scopes,
		options...,
	)

	if err != nil {
		return nil, err
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

	return &authHandlers{
		AuthHandler:     rp.AuthURLHandler(state, provider, rp.WithPromptURLParam("Welcome back!")),
		CallbackHandler: rp.CodeExchangeHandler(rp.UserinfoCallback(marshalUserinfo), provider),
	}, nil
}

func (a *Service) CheckAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

type AuthError struct {
	StatusCode int   `json:"status"`
	Err        error `json:"err"`
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("status %d: err %v", e.StatusCode, e.Err)
}
