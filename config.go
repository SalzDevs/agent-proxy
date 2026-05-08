package groxy

import "time"

// Config contains settings used to create a Proxy.
type Config struct {
	// Addr is the TCP address the proxy listens on, such as "127.0.0.1:8080".
	Addr string
	Timeouts Timeouts
}

type Timeouts struct {
	Dial time.Duration
	TLSHandshake time.Duration
	ResponseHeader time.Duration
	IdleConn time.Duration
	ReadHeader time.Duration
	Idle time.Duration
}

