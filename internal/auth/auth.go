package auth

import (
	"fmt"
	"net/http"
)

type Provider struct {
	AuthHandler, CallbackHandler http.HandlerFunc
	AuthURI, CallbackURI         string
}

type ProviderCfg struct {
	ClientID, ClientSecret string
}

type AuthError struct {
	StatusCode int   `json:"status"`
	Err        error `json:"err"`
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("status %d: err %v", e.StatusCode, e.Err)
}
