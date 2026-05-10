package groxy

import "testing"

func TestMatchAllHosts(t *testing.T) {
	matcher := MatchAllHosts()

	for _, host := range []string{"example.com", "example.com:443", ""} {
		if !matcher(host) {
			t.Fatalf("MatchAllHosts()(%q) = false, want true", host)
		}
	}
}

func TestMatchHosts_ExactMatch(t *testing.T) {
	matcher := MatchHosts("example.com")

	if !matcher("example.com") {
		t.Fatal("expected exact host to match")
	}
	if !matcher("EXAMPLE.COM") {
		t.Fatal("expected exact host match to be case-insensitive")
	}
	if matcher("api.example.com") {
		t.Fatal("expected different host not to match")
	}
}

func TestMatchHosts_StripsPort(t *testing.T) {
	matcher := MatchHosts("example.com")

	if !matcher("example.com:443") {
		t.Fatal("expected host with port to match")
	}
}

func TestMatchHosts_WildcardMatch(t *testing.T) {
	matcher := MatchHosts("*.example.com")

	if !matcher("api.example.com") {
		t.Fatal("expected wildcard subdomain to match")
	}
	if !matcher("v1.api.example.com") {
		t.Fatal("expected nested wildcard subdomain to match")
	}
	if matcher("example.com") {
		t.Fatal("expected wildcard not to match root domain")
	}
	if matcher("example.org") {
		t.Fatal("expected different domain not to match")
	}
}

func TestMatchHosts_IgnoresEmptyPatterns(t *testing.T) {
	matcher := MatchHosts("", "  ")

	if matcher("example.com") {
		t.Fatal("expected empty patterns not to match")
	}
}

func TestMatchHosts_IPv6WithPort(t *testing.T) {
	matcher := MatchHosts("2001:db8::1")

	if !matcher("[2001:db8::1]:443") {
		t.Fatal("expected IPv6 host with port to match")
	}
}
