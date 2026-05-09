package groxy

import "net/http"

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
	apply(*Proxy)
}

type requestMiddleware struct {
	hook RequestHook
}

func (m requestMiddleware) apply(p *Proxy) {
	p.requestHooks = append(p.requestHooks, m.hook)
}

type responseMiddleware struct {
	hook ResponseHook
}

func (m responseMiddleware) apply(p *Proxy) {
	p.responseHooks = append(p.responseHooks, m.hook)
}

type connectMiddleware struct {
	hook ConnectHook
}

func (m connectMiddleware) apply(p *Proxy) {
	p.connectHooks = append(p.connectHooks, m.hook)
}

// OnRequest creates middleware that runs fn before HTTP requests are sent upstream.
func OnRequest(fn RequestHook) Middleware {
	return requestMiddleware{hook: fn}
}

// OnResponse creates middleware that runs fn before HTTP responses are sent to the client.
func OnResponse(fn ResponseHook) Middleware {
	return responseMiddleware{hook: fn}
}

// OnConnect creates middleware that runs fn before CONNECT tunnels are opened.
func OnConnect(fn ConnectHook) Middleware {
	return connectMiddleware{hook: fn}
}

// Use adds middleware to the proxy.
func (p *Proxy) Use(middleware ...Middleware) {
	for _, m := range middleware {
		if m != nil {
			m.apply(p)
		}
	}
}

// OnRequest adds a request hook to the proxy.
func (p *Proxy) OnRequest(fn RequestHook) {
	p.Use(OnRequest(fn))
}

// OnResponse adds a response hook to the proxy.
func (p *Proxy) OnResponse(fn ResponseHook) {
	p.Use(OnResponse(fn))
}

// OnConnect adds a CONNECT hook to the proxy.
func (p *Proxy) OnConnect(fn ConnectHook) {
	p.Use(OnConnect(fn))
}
