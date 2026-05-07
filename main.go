package main

import (
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
	config Config
	server *http.Server
	client *http.Client
	transport *http.Transport
	// true for running, false for stopped
	State bool
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

func NewProxy(config Config) (*Proxy,error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	transport := &http.Transport{}

	proxy := &Proxy{
		config: config,
		server: &http.Server{Addr: config.Addr},
		client: &http.Client{Transport: transport},
		transport: transport,
		State: false,
	}

	return proxy, nil
}

func removeHopByHopHeaders(h http.Header){
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

	for _,header := range headers {
		h.Del(header)
	}
}

func (p* Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: %s %s", r.Method, r.URL.String())

	if r.URL == nil || r.URL.Scheme == "" || r.URL.Host == "" {
		http.Error(w, "proxy request must contain an absolute URL", http.StatusBadRequest)
		return 
	}

	outReq, err := http.NewRequestWithContext(r.Context(), r.Method,r.URL.String(), r.Body)
	if err!=nil {
		http.Error(w,"failed to reach upstream request", http.StatusInternalServerError)
		return
	}

	outReq.Header = r.Header.Clone()
	removeHopByHopHeaders(outReq.Header)
	outReq.Host = r.URL.Host

	resp,err := p.client.Do(outReq)
	if err!=nil {
		http.Error(w,"failed to reach upstream server",http.StatusBadGateway)
		return
	}

	defer resp.Body.Close()

	removeHopByHopHeaders(resp.Header)

	for k, values := range resp.Header {
		for _,v := range values {
			w.Header().Add(k,v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_,_ = io.Copy(w,resp.Body)
}


func (p *Proxy) StartProxy() error {
  p.server.Handler = p
	p.State = true
	defer func() { p.State = false}()

	log.Printf("Starting proxy server on %s", p.config.Addr)
	return p.server.ListenAndServe()
}

func main(){
	config := Config{Addr: "127.0.0.1:8080"}
	proxy, err := NewProxy(config)
	if err != nil {
		log.Fatalf("Failed to create proxy: %v", err)
	}

	if err := proxy.StartProxy(); err != nil {	
		log.Fatalf("Failed to start proxy: %v", err)
	}
}

