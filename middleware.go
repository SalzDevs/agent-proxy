package groxy

import (
	"fmt"
	"net/http"
)

// RequestHook is called before a normal HTTP request is sent upstream.
type RequestHook func(*RequestContext) error

// ResponseHook is called before an upstream HTTP response is sent back to the client.
type ResponseHook func(*ResponseContext) error

// ConnectHook is called before a CONNECT tunnel is opened.
type ConnectHook func(*ConnectContext) error

// RequestContext contains data available to request hooks.
type RequestContext struct {
	Request *http.Request
}

// ResponseContext contains data available to response hooks.
type ResponseContext struct {
	Request  *http.Request
	Response *http.Response
}

// ConnectContext contains data available to CONNECT hooks.
type ConnectContext struct {
	Host string
}

// Middleware configures proxy behavior.
type Middleware interface {
	Name() string
	apply(*Proxy)
}

type requestMiddleware struct {
	name string
	hook RequestHook
}

func (m requestMiddleware) Name() string {
	return m.name
}

func (m requestMiddleware) apply(p *Proxy) {
	p.requestHooks = append(p.requestHooks, m.hook)
}

type responseMiddleware struct {
	name string
	hook ResponseHook
}

func (m responseMiddleware) Name() string {
	return m.name
}

func (m responseMiddleware) apply(p *Proxy) {
	p.responseHooks = append(p.responseHooks, m.hook)
}

type connectMiddleware struct {
	name string
	hook ConnectHook
}

func (m connectMiddleware) Name() string {
	return m.name
}

func (m connectMiddleware) apply(p *Proxy) {
	p.connectHooks = append(p.connectHooks, m.hook)
}

func newRequestMiddleware(name string, fn RequestHook) Middleware {
	return requestMiddleware{name: name, hook: fn}
}

func newResponseMiddleware(name string, fn ResponseHook) Middleware {
	return responseMiddleware{name: name, hook: fn}
}

func newConnectMiddleware(name string, fn ConnectHook) Middleware {
	return connectMiddleware{name: name, hook: fn}
}

// OnRequest creates middleware that runs fn before HTTP requests are sent upstream.
func OnRequest(fn RequestHook) Middleware {
	return newRequestMiddleware("OnRequest", fn)
}

// OnResponse creates middleware that runs fn before HTTP responses are sent to the client.
func OnResponse(fn ResponseHook) Middleware {
	return newResponseMiddleware("OnResponse", fn)
}

// OnConnect creates middleware that runs fn before CONNECT tunnels are opened.
func OnConnect(fn ConnectHook) Middleware {
	return newConnectMiddleware("OnConnect", fn)
}

// Use adds middleware to the proxy.
//
// Middleware must be registered before Start is called. Use returns an error if
// middleware is added after the proxy has started.
func (p *Proxy) Use(middleware ...Middleware) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, m := range middleware {
		if m == nil {
			continue
		}

		if p.running {
			return fmt.Errorf("cannot add middleware %q after proxy has started", m.Name())
		}

		m.apply(p)
	}

	return nil
}

// OnRequest adds a request hook to the proxy.
func (p *Proxy) OnRequest(fn RequestHook) error {
	return p.Use(OnRequest(fn))
}

// OnResponse adds a response hook to the proxy.
func (p *Proxy) OnResponse(fn ResponseHook) error {
	return p.Use(OnResponse(fn))
}

// OnConnect adds a CONNECT hook to the proxy.
func (p *Proxy) OnConnect(fn ConnectHook) error {
	return p.Use(OnConnect(fn))
}
