package groxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// Proxy represents a forward proxy server.
//
// A Proxy can be started as a standalone server with Start, gracefully stopped
// with Shutdown, or used directly as an http.Handler through ServeHTTP.
type Proxy struct {
	config        Config
	server        *http.Server
	client        *http.Client
	transport     *http.Transport
	logger        Logger
	requestHooks  []RequestHook
	responseHooks []ResponseHook
	connectHooks  []ConnectHook
	running       bool
	mu            sync.RWMutex
}

// New creates a Proxy from config.
//
// New validates the config and prepares the internal server/client state, but it
// does not start listening. Call Start to begin accepting proxy requests.
func New(config Config) (*Proxy, error) {
	timeouts := resolveTimeouts(config.Timeouts)
	config.Timeouts = &timeouts
	logger := resolveLogger(config.Logger)
	config.Logger = logger

	if err := validateConfig(config); err != nil {
		return nil, err
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: timeouts.Dial,
		}).DialContext,
		TLSHandshakeTimeout:   timeouts.TLSHandshake,
		ResponseHeaderTimeout: timeouts.ResponseHeader,
		IdleConnTimeout:       timeouts.IdleConn,
		DisableCompression:    true,
	}

	proxy := &Proxy{
		config: config,
		server: &http.Server{
			Addr:              config.Addr,
			ReadHeaderTimeout: timeouts.ReadHeader,
			IdleTimeout:       timeouts.Idle,
		},
		client:    &http.Client{Transport: transport},
		transport: transport,
		logger:    logger,
		running:   false,
	}

	return proxy, nil
}

// Addr returns the configured TCP address the proxy listens on.
func (p *Proxy) Addr() string {
	return p.config.Addr
}

// IsRunning reports whether the proxy server is currently running.
func (p *Proxy) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

func (p *Proxy) setRunning(flag bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.running = flag
}

// Start starts the proxy server and blocks until the server stops.
//
// Start returns nil when the server is stopped through Shutdown. It returns an
// error if the server fails to start or stops unexpectedly.
func (p *Proxy) Start() error {
	p.server.Handler = p
	p.setRunning(true)
	defer p.setRunning(false)

	p.logger.Printf("Starting proxy server on %s", p.config.Addr)
	if err := p.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Shutdown gracefully stops the proxy server.
//
// Shutdown stops accepting new requests and waits for active requests to finish
// until ctx is canceled or expires.
func (p *Proxy) Shutdown(ctx context.Context) error {
	if !p.IsRunning() {
		return fmt.Errorf("proxy is not running")
	}

	defer p.setRunning(false)
	p.logger.Printf("Stopping proxy server on %s", p.config.Addr)
	return p.server.Shutdown(ctx)
}
