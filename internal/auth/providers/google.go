package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	"github.com/zitadel/oidc/v2/pkg/client/rs"
	httphelper "github.com/zitadel/oidc/v2/pkg/http"
	"github.com/zitadel/oidc/v2/pkg/oidc"
	googleOAuth "golang.org/x/oauth2/google"
)

var (
	googleAuthURI     = "/google"
	googleCallbackURI = "/google/callback"
	googleScopes      = []string{"email", "profile", "openid"}
	googleIATOffset   = time.Second * 5
	googleUrlParams   = []rp.URLParamOpt{
		rp.WithURLParam("access_type", "offline"),

		// prompt=consent forces google API to send a new refresh token on each login
		// https://stackoverflow.com/questions/10827920/not-receiving-google-oauth-refresh-token
		rp.WithURLParam("prompt", "consent"),
	}
)

type GoogleProvider struct {
	issuer                 string
	authType               auth.AuthType
	clientID, clientSecret string
	hashKey                []byte
	encKey                 []byte
	rp                     rp.RelyingParty
	rs                     rs.ResourceServer
}

func NewGoogle(clientID, clientSecret string, hashKey, encKey []byte) (Provider, error) {
	parsed, err := url.Parse(googleOAuth.Endpoint.AuthURL)
	if err != nil {
		return nil, err
	}
	return &GoogleProvider{fmt.Sprint("https://", parsed.Hostname()), auth.OIDC, clientID, clientSecret, hashKey, encKey, nil, nil}, nil
}

func (p *GoogleProvider) GetIssuer() string {
	return p.issuer
}

func (p *GoogleProvider) GetAuthType() auth.AuthType {
	return p.authType
}

func (p *GoogleProvider) GetHandlers(baseURI string, callback rp.CodeExchangeCallback[*oidc.IDTokenClaims]) (*Handlers, error) {
	cookieHandler := httphelper.NewCookieHandler(
		p.hashKey,
		p.encKey,
		httphelper.WithUnsecure(),
		httphelper.WithSameSite(http.SameSiteLaxMode),
	)

	options := []rp.Option{
		rp.WithCookieHandler(cookieHandler),
		rp.WithVerifierOpts(rp.WithIssuedAtOffset(googleIATOffset)),
	}

	redirectURI := fmt.Sprintf("%s%s", baseURI, googleCallbackURI)

	orp, err := rp.NewRelyingPartyOIDC(
		p.issuer,
		p.clientID,
		p.clientSecret,
		redirectURI,
		googleScopes,
		options...,
	)
	if err != nil {
		return nil, err
	}

	p.rp = orp

	// ors, err := rs.NewResourceServerClientCredentials(
	// 	p.issuer,
	// 	p.clientID,
	// 	p.clientSecret,
	// 	rs.WithStaticEndpoints("", ""),
	// )
	// if err != nil {
	// 	return nil, err
	// }

	// p.rs = ors

	ah, ch, err := p.getHandlers(callback)
	if err != nil {
		return nil, err
	}

	return &Handlers{
		AuthHandler:     ah,
		CallbackHandler: ch,
		AuthURI:         googleAuthURI,
		CallbackURI:     googleCallbackURI,
	}, nil
}

func (p *GoogleProvider) getHandlers(callback rp.CodeExchangeCallback[*oidc.IDTokenClaims]) (http.HandlerFunc, http.HandlerFunc, error) {
	state := func() string {
		return uuid.New().String()
	}

	return rp.AuthURLHandler(state, p.rp, googleUrlParams...),
		rp.CodeExchangeHandler(callback, p.rp), nil
}

func (p *GoogleProvider) Introspect(ctx context.Context, session *auth.Session) (*oidc.IntrospectionResponse, error) {
	token, err := checkToken(session.AccessToken)
	if err != nil {
		return nil, err
	}

	return rs.Introspect(ctx, p.rs, token)
}

func checkToken(token string) (string, error) {
	if !strings.HasPrefix(token, oidc.PrefixBearer) {
		return "", fmt.Errorf("invalid token header")
	}

	return strings.TrimPrefix(token, oidc.PrefixBearer), nil
}

func (p *GoogleProvider) UserInfo(ctx context.Context, token string) (*auth.OAuthUserInfo, error) {
	return nil, nil
}
