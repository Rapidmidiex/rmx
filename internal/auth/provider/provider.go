package provider

import (
	"context"
	"net/http"

	"github.com/zitadel/oidc/v2/pkg/client/rp"
	"github.com/zitadel/oidc/v2/pkg/oidc"
)

type Handlers struct {
	AuthHandler, CallbackHandler http.HandlerFunc
	AuthURI, CallbackURI         string
}

type Provider interface {
	Init(baseURI string, callback rp.CodeExchangeUserinfoCallback[*oidc.IDTokenClaims]) (*Handlers, error)
	Introspect(ctx context.Context, token string) (*oidc.IntrospectionResponse, error)
}
