package groxy

import "time"

const (
	defaultDialTimeout           = 10 * time.Second
	defaultTLSHandshakeTimeout   = 10 * time.Second
	defaultResponseHeaderTimeout = 30 * time.Second
	defaultIdleConnTimeout       = 90 * time.Second
	defaultReadHeaderTimeout     = 5 * time.Second
	defaultIdleTimeout           = 60 * time.Second
)

// Config contains settings used to create a Proxy.
type Config struct {
	// Addr is the TCP address the proxy listens on, such as "127.0.0.1:8080".
	Addr string

	// Timeouts controls network timeout behavior for the proxy.
	//
	// If nil, Groxy uses DefaultTimeouts. If provided, zero-valued fields are
	// filled with their default values.
	Timeouts *Timeouts
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

// DefaultTimeouts returns Groxy's default timeout values.
func DefaultTimeouts() Timeouts {
	return Timeouts{
		Dial:           defaultDialTimeout,
		TLSHandshake:   defaultTLSHandshakeTimeout,
		ResponseHeader: defaultResponseHeaderTimeout,
		IdleConn:       defaultIdleConnTimeout,
		ReadHeader:     defaultReadHeaderTimeout,
		Idle:           defaultIdleTimeout,
	}
}

func resolveTimeouts(custom *Timeouts) Timeouts {
	defaults := DefaultTimeouts()
	if custom == nil {
		return defaults
	}

	return Timeouts{
		Dial:           durationOrDefault(custom.Dial, defaults.Dial),
		TLSHandshake:   durationOrDefault(custom.TLSHandshake, defaults.TLSHandshake),
		ResponseHeader: durationOrDefault(custom.ResponseHeader, defaults.ResponseHeader),
		IdleConn:       durationOrDefault(custom.IdleConn, defaults.IdleConn),
		ReadHeader:     durationOrDefault(custom.ReadHeader, defaults.ReadHeader),
		Idle:           durationOrDefault(custom.Idle, defaults.Idle),
	}
}

func durationOrDefault(value, fallback time.Duration) time.Duration {
	if value == 0 {
		return fallback
	}

	return value
}
