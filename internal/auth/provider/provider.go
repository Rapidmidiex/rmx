package provider

import (
	"context"
	"net/http"

	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/zitadel/oidc/v2/pkg/client/rp"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

type Handlers struct {
	AuthHandler, CallbackHandler http.HandlerFunc
	AuthURI, CallbackURI         string
}

type Provider interface {
	Issuer() string
	GetHandlers(baseURI string, callback rp.CodeExchangeCallback[*oidc.IDTokenClaims]) (*Handlers, error)
	Introspect(ctx context.Context, token *auth.Session) (*oidc.IntrospectionResponse, error)
}
