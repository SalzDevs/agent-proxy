package groxy

import "time"

// Config contains settings used to create a Proxy.
type Config struct {
	// Addr is the TCP address the proxy listens on, such as "127.0.0.1:8080".
	Addr string

	// Timeouts controls network timeout behavior for the proxy.
	//
	// If left as the zero value, Groxy will use its default timeout values.
	Timeouts Timeouts
}

// Timeouts contains timeout settings for client, upstream, and idle proxy connections.
type Timeouts struct {
	// Dial is the maximum time allowed to connect to an upstream server.
	Dial time.Duration

	// TLSHandshake is the maximum time allowed for TLS handshakes made by the proxy HTTP client.
	TLSHandshake time.Duration

	// ResponseHeader is the maximum time allowed to wait for upstream response headers.
	ResponseHeader time.Duration

	// IdleConn is the maximum time an unused upstream keep-alive connection stays open.
	IdleConn time.Duration

	// ReadHeader is the maximum time allowed for a client to send request headers to the proxy.
	ReadHeader time.Duration

	// Idle is the maximum time an idle client connection to the proxy stays open.
	Idle time.Duration
}
