package groxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
)

// Proxy represents a forward proxy server.
type Proxy struct {
	config    Config
	server    *http.Server
	client    *http.Client
	transport *http.Transport
	running   bool
}

// New creates a proxy from the given config.
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

// IsRunning reports whether the proxy server is currently running.
func (p *Proxy) IsRunning() bool {
	return p.running
}

// Start starts the proxy server and blocks until it stops.
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
func (p *Proxy) Shutdown(ctx context.Context) error {
	if !p.IsRunning() {
		return fmt.Errorf("proxy is not running")
	}

	defer func() { p.running = false }()
	log.Printf("Stopping proxy server on %s", p.config.Addr)
	return p.server.Shutdown(ctx)
}
