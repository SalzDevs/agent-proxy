package groxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
)

// Proxy represents a forward proxy server.
//
// A Proxy can be started as a standalone server with Start, gracefully stopped
// with Shutdown, or used directly as an http.Handler through ServeHTTP.
type Proxy struct {
	config    Config
	server    *http.Server
	client    *http.Client
	transport *http.Transport
	running   bool
}

// New creates a Proxy from config.
//
// New validates the config and prepares the internal server/client state, but it
// does not start listening. Call Start to begin accepting proxy requests.
func New(config Config) (*Proxy, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	transport := &http.Transport{}

	proxy := &Proxy{
		config:    config,
		server:    &http.Server{Addr: config.Addr},
		client:    &http.Client{Transport: transport},
		transport: transport,
		running:   false,
	}

	return proxy, nil
}

// Addr returns the configured TCP address the proxy listens on
func (p *Proxy) Addr() string {
	return p.config.Addr
}

// IsRunning reports whether the proxy server is currently running.
func (p *Proxy) IsRunning() bool {
	return p.running
}

// Start starts the proxy server and blocks until the server stops.
//
// Start returns nil when the server is stopped through Shutdown. It returns an
// error if the server fails to start or stops unexpectedly.
func (p *Proxy) Start() error {
	p.server.Handler = p
	p.running = true
	defer func() { p.running = false }()

	log.Printf("Starting proxy server on %s", p.config.Addr)
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

	defer func() { p.running = false }()
	log.Printf("Stopping proxy server on %s", p.config.Addr)
	return p.server.Shutdown(ctx)
}
