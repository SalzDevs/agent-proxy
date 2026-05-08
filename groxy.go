package groxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type Config struct {
	Addr string
}

type Proxy struct {
	config    Config
	server    *http.Server
	client    *http.Client
	transport *http.Transport
	// true for running, false for stopped
	running bool
}

func (p *Proxy) IsRunning() bool {
	return p.running
}

func parseAddr(addr string) (string, string, error) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return "", "", err
	}

	return host, port, nil
}

func validatePort(port string) error {
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("port must be numeric: %w", err)
	}

	if portInt < 1 || portInt > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

func validateAddr(addr string) error {
	if strings.TrimSpace(addr) == "" {
		return fmt.Errorf("address is required")
	}

	_, port, err := parseAddr(addr)
	if err != nil {
		return fmt.Errorf("invalid address %q: %w", addr, err)
	}

	if err := validatePort(port); err != nil {
		return fmt.Errorf("invalid address %q: %w", addr, err)
	}

	return nil
}

func validateConfig(config Config) error {
	if err := validateAddr(config.Addr); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return nil
}

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

func removeHopByHopHeaders(h http.Header) {
	headers := []string{
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	}

	for _, header := range headers {
		h.Del(header)
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: %s %s", r.Method, r.URL.String())

	if r.Method == http.MethodConnect {
		p.handleCONNECT(w, r)
		return
	}

	p.handleForwardHTTP(w, r)
}

func (p *Proxy) handleForwardHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL == nil || r.URL.Scheme == "" || r.URL.Host == "" {
		http.Error(w, "proxy request must contain an absolute URL", http.StatusBadRequest)
		return
	}

	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, "failed to reach upstream request", http.StatusInternalServerError)
		return
	}

	outReq.Header = r.Header.Clone()
	removeHopByHopHeaders(outReq.Header)
	outReq.Host = r.URL.Host

	resp, err := p.client.Do(outReq)
	if err != nil {
		http.Error(w, "failed to reach upstream server", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	removeHopByHopHeaders(resp.Header)

	for k, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (p *Proxy) handleCONNECT(w http.ResponseWriter, r *http.Request) {
	target := r.Host
	if target == "" {
		http.Error(w, "CONNECT request missing target host", http.StatusBadRequest)
		return
	}

	upstreamConn, err := net.Dial("tcp", target)
	if err != nil {
		http.Error(w, "failed to connect to target", http.StatusBadGateway)
		return
	}
	defer upstreamConn.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "failed to hijack connection", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		return
	}

	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(upstreamConn, clientConn)
		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(clientConn, upstreamConn)
		done <- struct{}{}
	}()

	<-done
}

func (p *Proxy) Shutdown(ctx context.Context) error {
	if !p.IsRunning() {
		return fmt.Errorf("proxy is not running")
	}

	defer func() { p.running = false }()
	log.Printf("Stopping proxy server on %s", p.config.Addr)
	return p.server.Shutdown(ctx)
}

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
