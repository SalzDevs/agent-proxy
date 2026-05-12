package groxy

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
)

// ProxyBasicAuth returns middleware that requires HTTP Basic proxy
// authentication with a static username and password.
//
// The credentials are read from the Proxy-Authorization header. Static
// credentials are compared in constant time. Basic authentication is not
// encrypted by itself, so only use it when the client-to-proxy connection is
// otherwise protected or trusted.
func ProxyBasicAuth(username, password string) Middleware {
	wantUsername := sha256.Sum256([]byte(username))
	wantPassword := sha256.Sum256([]byte(password))

	return proxyBasicAuthMiddleware("ProxyBasicAuth", defaultProxyAuthRealm, func(gotUsername, gotPassword string) bool {
		gotUsernameHash := sha256.Sum256([]byte(gotUsername))
		gotPasswordHash := sha256.Sum256([]byte(gotPassword))

		usernameOK := subtle.ConstantTimeCompare(gotUsernameHash[:], wantUsername[:]) == 1
		passwordOK := subtle.ConstantTimeCompare(gotPasswordHash[:], wantPassword[:]) == 1
		return usernameOK && passwordOK
	})
}

func proxyBasicAuthMiddleware(name, realm string, validate func(username, password string) bool) Middleware {
	auth := &proxyBasicAuthenticator{realm: realm, validate: validate}
	return Middleware{name: name, requestHook: auth.onRequest}
}

type proxyBasicAuthenticator struct {
	realm    string
	validate func(username, password string) bool
}

func (auth *proxyBasicAuthenticator) onRequest(ctx *RequestContext) error {
	username, password, ok := parseProxyBasicAuth(ctx.Request.Header.Get("Proxy-Authorization"))
	if !ok || auth.validate == nil || !auth.validate(username, password) {
		return &proxyAuthRequiredError{realm: auth.realm}
	}

	return nil
}

type proxyAuthRequiredError struct {
	realm string
}

func (e proxyAuthRequiredError) Error() string {
	return "proxy authentication required"
}

func proxyAuthRequired(err error) (*proxyAuthRequiredError, bool) {
	var auth *proxyAuthRequiredError
	if errors.As(err, &auth) {
		return auth, true
	}

	return nil, false
}
