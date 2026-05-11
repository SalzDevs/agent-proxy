package groxy

import "time"

// HostMatcher decides whether a host should be selected for HTTPS inspection.
//
// The host may include a port, such as "example.com:443". Matchers should
// normalize hosts as needed before comparing them.
type HostMatcher func(host string) bool

// HTTPSInspectionConfig configures opt-in HTTPS inspection for CONNECT traffic.
//
// HTTPS inspection uses local TLS interception: Groxy terminates TLS from the
// client with a certificate signed by CA, then opens its own TLS connection to
// the upstream server. This allows normal request/response middleware and body
// transforms to run on selected HTTPS traffic.
//
// If HTTPSInspectionConfig is nil, Groxy keeps the current safe default and
// tunnels HTTPS traffic without inspecting encrypted request or response bodies.
type HTTPSInspectionConfig struct {
	// CA signs generated per-host certificates used for inspected HTTPS traffic.
	CA *CA

	// Intercept decides which CONNECT hosts should be inspected.
	//
	// This field is required when HTTPS inspection is enabled. Use MatchAllHosts
	// only if you explicitly want to inspect every host.
	Intercept HostMatcher

	// CertificateTTL controls how long generated per-host certificates are valid.
	//
	// If zero, Groxy uses a safe default. Generated certificates are kept in
	// memory only and renewed before they expire.
	CertificateTTL time.Duration

	// PassthroughOnError controls whether Groxy falls back to a normal CONNECT
	// tunnel if HTTPS inspection setup fails.
	//
	// The default is false, so inspection failures fail closed instead of
	// silently bypassing inspection policy.
	PassthroughOnError bool
}
