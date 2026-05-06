package main

import (
	"fmt"
	"log"
	"net/http"
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

func NewProxy(config Config) (*Proxy,error) {
	if config.Addr == "" {
		return nil, fmt.Errorf("Address is required")
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

func (p* Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: %s", r.Method)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello from the proxy!"))
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

