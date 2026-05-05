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

func main(){
	config := Config{Addr: "127.0.0.1:8080"}
	proxy, err := NewProxy(config)
	if err != nil {
		log.Fatalf("Failed to create proxy: %v", err)
	}

	log.Printf("Proxy data: %+v", proxy)
}

