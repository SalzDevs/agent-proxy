package groxy_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SalzDevs/groxy"
)

func TestPublicAPI_UseMiddleware(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Groxy-Request"); got != "true" {
			t.Fatalf("upstream X-Groxy-Request = %q, want %q", got, "true")
		}

		w.Header().Set("X-Upstream", "true")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	defer upstream.Close()

	proxy, err := groxy.New(groxy.Config{Addr: "127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("groxy.New() error = %v", err)
	}

	if err := proxy.Use(
		groxy.AccessLog(nil),
		groxy.ProxyBasicAuth("user", "pass"),
		groxy.AddRequestHeader("X-Groxy-Request", "true"),
		groxy.AddResponseHeader("X-Groxy-Response", "true"),
		groxy.TransformResponseBody(func(body []byte) ([]byte, error) {
			return bytes.ToUpper(body), nil
		}),
	); err != nil {
		t.Fatalf("proxy.Use() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, upstream.URL, nil)
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("X-Upstream"); got != "true" {
		t.Fatalf("response X-Upstream = %q, want %q", got, "true")
	}
	if got := rec.Header().Get("X-Groxy-Response"); got != "true" {
		t.Fatalf("response X-Groxy-Response = %q, want %q", got, "true")
	}
	if got := rec.Body.String(); got != "HELLO" {
		t.Fatalf("body = %q, want %q", got, "HELLO")
	}
}

func TestPublicAPI_ConnectContextIncludesRequest(t *testing.T) {
	proxy, err := groxy.New(groxy.Config{Addr: "127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("groxy.New() error = %v", err)
	}

	var sawRequest bool
	if err := proxy.OnConnect(func(ctx *groxy.ConnectContext) error {
		sawRequest = ctx.Request != nil && ctx.Request.Header.Get("X-Test") == "true"
		return groxy.Block(http.StatusForbidden, "stop")
	}); err != nil {
		t.Fatalf("proxy.OnConnect() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodConnect, "//example.com:443", nil)
	req.Host = "example.com:443"
	req.Header.Set("X-Test", "true")
	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if !sawRequest {
		t.Fatal("ConnectContext.Request was not available to hook")
	}
}

func TestPublicAPI_ProxyBasicAuthFunc(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	proxy, err := groxy.New(groxy.Config{Addr: "127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("groxy.New() error = %v", err)
	}

	if err := proxy.Use(groxy.ProxyBasicAuthFunc(func(username, password string) bool {
		return username == "user" && password == "pass"
	})); err != nil {
		t.Fatalf("proxy.Use() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, upstream.URL, nil)
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestPublicAPI_Block(t *testing.T) {
	proxy, err := groxy.New(groxy.Config{Addr: "127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("groxy.New() error = %v", err)
	}

	if err := proxy.Use(groxy.BlockHost("blocked.example", http.StatusForbidden, "blocked")); err != nil {
		t.Fatalf("proxy.Use() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://blocked.example/", nil)
	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}
