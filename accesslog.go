package groxy

import (
	"net/http"
	"sync"
	"time"
)

func AccessLog(logger Logger) Middleware {
	al := &accessLogger{logger: logger}
	return Middleware{
		name:         "AccessLog",
		requestHook:  al.onRequest,
		responseHook: al.onResponse,
		connectHook:  al.onConnect,
	}
}

type accessLogger struct {
	mu      sync.Mutex
	logger  Logger
	started map[*http.Request]time.Time
}

func (al *accessLogger) onRequest(ctx *RequestContext) error {
	al.mu.Lock()
	if al.started == nil {
		al.started = make(map[*http.Request]time.Time)
	}
	al.started[ctx.Request] = time.Now()
	al.mu.Unlock()

	al.logger.Printf("> %s %s", ctx.Request.Method, requestHost(ctx.Request))
	return nil
}

func (al *accessLogger) onResponse(ctx *ResponseContext) error {
	al.mu.Lock()
	start, recorded := al.started[ctx.Request]
	if recorded {
		delete(al.started, ctx.Request)
	}
	al.mu.Unlock()

	host := requestHost(ctx.Request)
	if recorded {
		al.logger.Printf("< %s %s %d %s",
			ctx.Request.Method, host, ctx.Response.StatusCode, time.Since(start))
	} else {
		al.logger.Printf("< %s %s %d",
			ctx.Request.Method, host, ctx.Response.StatusCode)
	}
	return nil
}

func (al *accessLogger) onConnect(ctx *ConnectContext) error {
	al.logger.Printf("CONNECT %s", ctx.Host)
	return nil
}

func requestHost(r *http.Request) string {
	if r.URL.Host != "" {
		return r.URL.Host
	}
	return r.Host
}
