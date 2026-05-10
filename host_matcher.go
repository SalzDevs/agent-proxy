package groxy

import (
	"net"
	"strings"
)

// MatchAllHosts returns a HostMatcher that matches every host.
func MatchAllHosts() HostMatcher {
	return func(host string) bool {
		return true
	}
}

// MatchHosts returns a HostMatcher for exact and wildcard host patterns.
//
// Patterns are case-insensitive. Hosts may include ports. A pattern beginning
// with "*." matches subdomains only, so "*.example.com" matches
// "api.example.com" but not "example.com".
func MatchHosts(patterns ...string) HostMatcher {
	normalized := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = normalizeHostPattern(pattern)
		if pattern == "" {
			continue
		}

		normalized = append(normalized, pattern)
	}

	return func(host string) bool {
		host = normalizeHost(host)
		if host == "" {
			return false
		}

		for _, pattern := range normalized {
			if strings.HasPrefix(pattern, "*.") {
				base := strings.TrimPrefix(pattern, "*.")
				if host != base && strings.HasSuffix(host, "."+base) {
					return true
				}
				continue
			}

			if host == pattern {
				return true
			}
		}

		return false
	}
}

func normalizeHostPattern(pattern string) string {
	return normalizeHost(pattern)
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}

	if strings.HasPrefix(host, "[") {
		if withoutPort, _, err := net.SplitHostPort(host); err == nil {
			return strings.Trim(withoutPort, "[]")
		}
		return strings.Trim(host, "[]")
	}

	if withoutPort, _, err := net.SplitHostPort(host); err == nil {
		return withoutPort
	}

	if strings.Count(host, ":") == 1 {
		name, port, ok := strings.Cut(host, ":")
		if ok && port != "" {
			return name
		}
	}

	return strings.TrimSuffix(host, ".")
}
