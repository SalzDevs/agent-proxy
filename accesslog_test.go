package groxy

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAccessLog_HTTP(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	}))
	defer upstream.Close()

	proxy := newTestProxy(t)
	mustUse(t, proxy, AccessLog(logger))

	req := httptest.NewRequest(http.MethodGet, upstream.URL, nil)
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	got := buf.String()
	for _, want := range []string{"> GET ", "< GET ", "201"} {
		if !strings.Contains(got, want) {
			t.Fatalf("access log = %q, want substring %q", got, want)
		}
	}
}

func TestAccessLog_BlockedRequestLogsAndCleansUp(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	accessLog := &accessLogger{
		logger:  resolveLogger(logger),
		started: make(map[*http.Request]time.Time),
	}

	proxy := newTestProxy(t)
	mustUse(t, proxy, Middleware{
		name:                "AccessLog",
		requestHook:         accessLog.onRequest,
		forwardCompleteHook: accessLog.onForwardComplete,
	}, BlockHost("blocked.example", http.StatusForbidden, "blocked"))

	req := httptest.NewRequest(http.MethodGet, "http://blocked.example/", nil)
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	got := buf.String()
	for _, want := range []string{"> GET blocked.example", "< GET blocked.example 403"} {
		if !strings.Contains(got, want) {
			t.Fatalf("access log = %q, want substring %q", got, want)
		}
	}

	accessLog.mu.Lock()
	defer accessLog.mu.Unlock()
	if len(accessLog.started) != 0 {
		t.Fatalf("started map has %d entries, want 0", len(accessLog.started))
	}
}

func TestAccessLog_DoesNotLogProxyAuthorization(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	proxy := newTestProxy(t)
	mustUse(t, proxy, AccessLog(logger), ProxyBasicAuth("user", "pass"))

	req := httptest.NewRequest(http.MethodGet, upstream.URL, nil)
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	got := buf.String()
	for _, forbidden := range []string{"Proxy-Authorization", "dXNlcjpwYXNz", "user", "pass"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("access log = %q, should not contain %q", got, forbidden)
		}
	}
}

func TestAccessLog_CONNECT(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	proxy := newTestProxy(t)
	mustUse(t, proxy, AccessLog(logger))

	req := httptest.NewRequest(http.MethodConnect, "/", nil)
	req.Host = "example.com:443"
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	got := buf.String()
	if !strings.Contains(got, "CONNECT example.com:443") {
		t.Fatalf("access log = %q, want CONNECT target", got)
	}
}

func TestAccessLog_NilLogger(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	proxy := newTestProxy(t)
	mustUse(t, proxy, AccessLog(nil))

	req := httptest.NewRequest(http.MethodGet, upstream.URL, nil)
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
