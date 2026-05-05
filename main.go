package main

import (
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

func main(){
	log.Println("Starting proxy server...")
}

