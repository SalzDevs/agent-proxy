package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type capturedRequest struct {
	Method          string
	Path            string
	Body            string
	XTest           string
	ContentType     string
	Connection      string
	ProxyConnection string
}

func newTestProxy(t *testing.T) *Proxy {
	t.Helper()

	p, err := NewProxy(Config{Addr: "127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("NewProxy() error = %v", err)
	}

	return p
}

func TestNewProxy_RejectsEmptyAddr(t *testing.T) {
	if _, err := NewProxy(Config{}); err == nil {
		t.Fatal("expected error for empty address, got nil")
	}
}

func TestNewProxy_RejectsInvalidAddr(t *testing.T) {
	cases := []Config{
		{Addr: "127.0.0.1"},
		{Addr: "127.0.0.1:99999"},
		{Addr: "127.0.0.1:abc"},
	}

	for _, tc := range cases {
		if _, err := NewProxy(tc); err == nil {
			t.Fatalf("expected error for addr %q, got nil", tc.Addr)
		}
	}
}

func TestNewProxy_InitializesInternalFields(t *testing.T) {
	cfg := Config{Addr: "127.0.0.1:8080"}

	p, err := NewProxy(cfg)
	if err != nil {
		t.Fatalf("NewProxy() error = %v", err)
	}

	if p == nil {
		t.Fatal("expected proxy, got nil")
	}
	if p.config != cfg {
		t.Fatalf("proxy config = %+v, want %+v", p.config, cfg)
	}
	if p.server == nil {
		t.Fatal("expected server to be initialized")
	}
	if p.client == nil {
		t.Fatal("expected client to be initialized")
	}
	if p.transport == nil {
		t.Fatal("expected transport to be initialized")
	}
	if p.server.Addr != cfg.Addr {
		t.Fatalf("server addr = %q, want %q", p.server.Addr, cfg.Addr)
	}
	if p.client.Transport != p.transport {
		t.Fatalf("client transport = %#v, want %#v", p.client.Transport, p.transport)
	}
	if p.State {
		t.Fatal("expected proxy to start stopped")
	}
}

func TestServeHTTP_ReturnsBadRequestForRelativeURL(t *testing.T) {
	p := newTestProxy(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/relative", nil)

	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestServeHTTP_ForwardsGETRequestAndResponseHeaders(t *testing.T) {
	seen := make(chan capturedRequest, 1)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen <- capturedRequest{
			Method:          r.Method,
			Path:            r.URL.RequestURI(),
			Body:            string(body),
			XTest:           r.Header.Get("X-Test"),
			Connection:      r.Header.Get("Connection"),
			ProxyConnection: r.Header.Get("Proxy-Connection"),
			ContentType:     r.Header.Get("Content-Type"),
		}

		w.Header().Set("X-Upstream", "yes")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("upstream ok"))
	}))
	defer upstream.Close()

	p := newTestProxy(t)

	req, err := http.NewRequest(http.MethodGet, upstream.URL+"/hello?x=1", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	req.Header.Set("X-Test", "abc")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Proxy-Connection", "close")

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	got := <-seen
	if got.Method != http.MethodGet {
		t.Fatalf("upstream method = %q, want %q", got.Method, http.MethodGet)
	}
	if got.Path != "/hello?x=1" {
		t.Fatalf("upstream path = %q, want %q", got.Path, "/hello?x=1")
	}
	if got.XTest != "abc" {
		t.Fatalf("upstream header X-Test = %q, want %q", got.XTest, "abc")
	}
	if got.Connection != "" {
		t.Fatalf("upstream Connection header = %q, want empty", got.Connection)
	}
	if got.ProxyConnection != "" {
		t.Fatalf("upstream Proxy-Connection header = %q, want empty", got.ProxyConnection)
	}
	if got.Body != "" {
		t.Fatalf("upstream body = %q, want empty", got.Body)
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("response status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if rec.Header().Get("X-Upstream") != "yes" {
		t.Fatalf("response header X-Upstream = %q, want %q", rec.Header().Get("X-Upstream"), "yes")
	}
	if rec.Body.String() != "upstream ok" {
		t.Fatalf("response body = %q, want %q", rec.Body.String(), "upstream ok")
	}
}

func TestServeHTTP_ForwardsPOSTBody(t *testing.T) {
	seen := make(chan capturedRequest, 1)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen <- capturedRequest{
			Method:      r.Method,
			Path:        r.URL.RequestURI(),
			Body:        string(body),
			XTest:       r.Header.Get("X-Test"),
			ContentType: r.Header.Get("Content-Type"),
		}

		w.Header().Set("X-Upstream", "post-ok")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("post received"))
	}))
	defer upstream.Close()

	p := newTestProxy(t)

	payload := `{"hello":"world"}`
	req, err := http.NewRequest(http.MethodPost, upstream.URL+"/submit", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test", "post-header")

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	got := <-seen
	if got.Method != http.MethodPost {
		t.Fatalf("upstream method = %q, want %q", got.Method, http.MethodPost)
	}
	if got.Path != "/submit" {
		t.Fatalf("upstream path = %q, want %q", got.Path, "/submit")
	}
	if got.Body != payload {
		t.Fatalf("upstream body = %q, want %q", got.Body, payload)
	}
	if got.XTest != "post-header" {
		t.Fatalf("upstream header X-Test = %q, want %q", got.XTest, "post-header")
	}
	if got.ContentType != "application/json" {
		t.Fatalf("upstream Content-Type = %q, want %q", got.ContentType, "application/json")
	}

	if rec.Code != http.StatusAccepted {
		t.Fatalf("response status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	if rec.Header().Get("X-Upstream") != "post-ok" {
		t.Fatalf("response header X-Upstream = %q, want %q", rec.Header().Get("X-Upstream"), "post-ok")
	}
	if rec.Body.String() != "post received" {
		t.Fatalf("response body = %q, want %q", rec.Body.String(), "post received")
	}
}

func TestRemoveHopByHopHeaders_RemovesHeaders(t *testing.T) {
	h := http.Header{}
	h.Set("Connection", "keep-alive")
	h.Set("Proxy-Connection", "close")
	h.Set("Keep-Alive", "timeout=5")
	h.Set("Upgrade", "websocket")
	h.Set("X-Keep", "yes")

	removeHopByHopHeaders(h)

	for _, header := range []string{
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Upgrade",
	} {
		if got := h.Get(header); got != "" {
			t.Fatalf("header %q = %q, want empty", header, got)
		}
	}

	if got := h.Get("X-Keep"); got != "yes" {
		t.Fatalf("header X-Keep = %q, want %q", got, "yes")
	}
}
