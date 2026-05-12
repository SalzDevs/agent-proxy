package groxy

import (
	"net/http"
	"sync"
	"time"
)

// AccessLog returns middleware that writes one-line logs for proxied HTTP
// requests and CONNECT tunnels.
//
// HTTP requests are logged when they are sent upstream and again when they
// finish with a response, block, or forwarding error. CONNECT tunnels are logged
// when the CONNECT hook runs. Pass nil to disable logging.
func AccessLog(logger Logger) Middleware {
	al := &accessLogger{
		logger:  resolveLogger(logger),
		started: make(map[*http.Request]time.Time),
	}

	return Middleware{
		name:                "AccessLog",
		requestHook:         al.onRequest,
		connectHook:         al.onConnect,
		forwardCompleteHook: al.onForwardComplete,
	}
}

type accessLogger struct {
	mu      sync.Mutex
	logger  Logger
	started map[*http.Request]time.Time
}

func (al *accessLogger) onRequest(ctx *RequestContext) error {
	al.mu.Lock()
	al.started[ctx.Request] = time.Now()
	al.mu.Unlock()

	al.logger.Printf("> %s %s", ctx.Request.Method, accessLogHost(ctx.Request))
	return nil
}

func (al *accessLogger) onForwardComplete(ctx *forwardCompleteContext) {
	al.mu.Lock()
	start, recorded := al.started[ctx.Request]
	if recorded {
		delete(al.started, ctx.Request)
	}
	al.mu.Unlock()

	if !recorded {
		return
	}

	al.logger.Printf("< %s %s %d %s",
		ctx.Request.Method,
		accessLogHost(ctx.Request),
		accessLogStatus(ctx),
		time.Since(start),
	)
}

func (al *accessLogger) onConnect(ctx *ConnectContext) error {
	al.logger.Printf("CONNECT %s", ctx.Host)
	return nil
}

func accessLogHost(r *http.Request) string {
	if r.URL != nil && r.URL.Host != "" {
		return r.URL.Host
	}
	return r.Host
}

func accessLogStatus(ctx *forwardCompleteContext) int {
	if ctx.Response != nil {
		return ctx.Response.StatusCode
	}

	if block, ok := blockError(ctx.Err); ok {
		if block.StatusCode >= 100 && block.StatusCode <= 999 {
			return block.StatusCode
		}
		return http.StatusForbidden
	}

	if forward, ok := ctx.Err.(forwardError); ok {
		return forward.status
	}

	return http.StatusBadGateway
}
