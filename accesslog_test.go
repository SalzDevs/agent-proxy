package groxy_test

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SalzDevs/groxy"
)

func TestAccessLog_HTTP(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	proxy, err := groxy.New(groxy.Config{Addr: "127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("groxy.New() error = %v", err)
	}
	if err := proxy.Use(groxy.AccessLog(logger)); err != nil {
		t.Fatalf("proxy.Use() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, upstream.URL, nil)
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	got := buf.String()
	if !strings.Contains(got, "GET") {
		t.Errorf("access log missing method; got %q", got)
	}
	if !strings.Contains(got, "200") {
		t.Errorf("access log missing status; got %q", got)
	}
}

func TestAccessLog_CONNECT(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	proxy, err := groxy.New(groxy.Config{Addr: "127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("groxy.New() error = %v", err)
	}
	if err := proxy.Use(groxy.AccessLog(logger)); err != nil {
		t.Fatalf("proxy.Use() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodConnect, "/", nil)
	req.Host = "example.com:443"
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	got := buf.String()
	if !strings.Contains(got, "CONNECT") {
		t.Errorf("access log missing CONNECT; got %q", got)
	}
	if !strings.Contains(got, "example.com:443") {
		t.Errorf("access log missing target host; got %q", got)
	}
}

func TestAccessLog_BlockedRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	proxy, err := groxy.New(groxy.Config{Addr: "127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("groxy.New() error = %v", err)
	}
	if err := proxy.Use(
		groxy.AccessLog(logger),
		groxy.BlockHost("blocked.example", http.StatusForbidden, "blocked"),
	); err != nil {
		t.Fatalf("proxy.Use() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://blocked.example/", nil)
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	got := buf.String()
	if !strings.Contains(got, ">") {
		t.Errorf("access log missing outgoing indicator for blocked request; got %q", got)
	}
	if !strings.Contains(got, "GET") {
		t.Errorf("access log missing method for blocked request; got %q", got)
	}
	if !strings.Contains(got, "blocked.example") {
		t.Errorf("access log missing host for blocked request; got %q", got)
	}
}
