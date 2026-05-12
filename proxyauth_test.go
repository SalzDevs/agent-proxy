package groxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyBasicAuth_AllowsValidHTTP(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Proxy-Authorization"); got != "" {
			t.Fatalf("upstream Proxy-Authorization = %q, want empty", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	proxy := newTestProxy(t)
	mustUse(t, proxy, ProxyBasicAuth("user", "pass"))

	req := httptest.NewRequest(http.MethodGet, upstream.URL, nil)
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestProxyBasicAuth_Name(t *testing.T) {
	if got := ProxyBasicAuth("user", "pass").Name(); got != "ProxyBasicAuth" {
		t.Fatalf("Name() = %q, want %q", got, "ProxyBasicAuth")
	}
}

func TestProxyBasicAuth_HasConnectHook(t *testing.T) {
	if ProxyBasicAuth("user", "pass").connectHook == nil {
		t.Fatal("connectHook is nil, want CONNECT authentication")
	}
}

func TestProxyBasicAuth_HookAllowsValidCredentials(t *testing.T) {
	middleware := ProxyBasicAuth("user", "pass")
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")

	if err := middleware.requestHook(&RequestContext{Request: req}); err != nil {
		t.Fatalf("requestHook() error = %v", err)
	}
}

func TestProxyBasicAuth_ConnectHookAllowsValidCredentials(t *testing.T) {
	middleware := ProxyBasicAuth("user", "pass")
	req := httptest.NewRequest(http.MethodConnect, "//example.com:443", nil)
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")

	if err := middleware.connectHook(&ConnectContext{Host: "example.com:443", Request: req}); err != nil {
		t.Fatalf("connectHook() error = %v", err)
	}
}

func TestProxyBasicAuth_RejectsMissingHTTP(t *testing.T) {
	proxy := newTestProxy(t)
	mustUse(t, proxy, ProxyBasicAuth("user", "pass"))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusProxyAuthRequired {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusProxyAuthRequired)
	}
	if got := rec.Header().Get("Proxy-Authenticate"); got != `Basic realm="Groxy"` {
		t.Fatalf("Proxy-Authenticate = %q, want default challenge", got)
	}
}

func TestProxyBasicAuth_RejectsInvalidHTTP(t *testing.T) {
	proxy := newTestProxy(t)
	mustUse(t, proxy, ProxyBasicAuth("user", "pass"))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjp3cm9uZw==")
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusProxyAuthRequired {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusProxyAuthRequired)
	}
}

func TestProxyBasicAuth_RejectsMalformedHTTP(t *testing.T) {
	proxy := newTestProxy(t)
	mustUse(t, proxy, ProxyBasicAuth("user", "pass"))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("Proxy-Authorization", "Basic not-base64")
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusProxyAuthRequired {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusProxyAuthRequired)
	}
}

func TestProxyBasicAuth_RejectsMissingCONNECT(t *testing.T) {
	proxy := newTestProxy(t)
	mustUse(t, proxy, ProxyBasicAuth("user", "pass"))

	req := httptest.NewRequest(http.MethodConnect, "//example.com:443", nil)
	req.Host = "example.com:443"
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusProxyAuthRequired {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusProxyAuthRequired)
	}
	if got := rec.Header().Get("Proxy-Authenticate"); got != `Basic realm="Groxy"` {
		t.Fatalf("Proxy-Authenticate = %q, want default challenge", got)
	}
}

func TestProxyBasicAuth_RejectsInvalidCONNECT(t *testing.T) {
	proxy := newTestProxy(t)
	mustUse(t, proxy, ProxyBasicAuth("user", "pass"))

	req := httptest.NewRequest(http.MethodConnect, "//example.com:443", nil)
	req.Host = "example.com:443"
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjp3cm9uZw==")
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusProxyAuthRequired {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusProxyAuthRequired)
	}
}

func TestProxyBasicAuthFunc_AllowsValidHTTP(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	var gotUsername, gotPassword string
	proxy := newTestProxy(t)
	mustUse(t, proxy, ProxyBasicAuthFunc(func(username, password string) bool {
		gotUsername = username
		gotPassword = password
		return username == "user" && password == "pass"
	}))

	req := httptest.NewRequest(http.MethodGet, upstream.URL, nil)
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if gotUsername != "user" {
		t.Fatalf("validator username = %q, want %q", gotUsername, "user")
	}
	if gotPassword != "pass" {
		t.Fatalf("validator password = %q, want %q", gotPassword, "pass")
	}
}

func TestProxyBasicAuthFunc_RejectsInvalidHTTP(t *testing.T) {
	proxy := newTestProxy(t)
	mustUse(t, proxy, ProxyBasicAuthFunc(func(username, password string) bool {
		return false
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusProxyAuthRequired {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusProxyAuthRequired)
	}
}

func TestProxyBasicAuthFunc_NilValidatorRejectsHTTP(t *testing.T) {
	proxy := newTestProxy(t)
	mustUse(t, proxy, ProxyBasicAuthFunc(nil))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusProxyAuthRequired {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusProxyAuthRequired)
	}
}

func TestProxyBasicAuthFunc_AllowsValidCONNECT(t *testing.T) {
	var gotUsername, gotPassword string
	proxy := newTestProxy(t)
	mustUse(t, proxy, ProxyBasicAuthFunc(func(username, password string) bool {
		gotUsername = username
		gotPassword = password
		return username == "user" && password == "pass"
	}))

	req := httptest.NewRequest(http.MethodConnect, "//example.com:443", nil)
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")

	if err := proxy.runConnectHooks("example.com:443", req); err != nil {
		t.Fatalf("runConnectHooks() error = %v", err)
	}
	if gotUsername != "user" {
		t.Fatalf("validator username = %q, want %q", gotUsername, "user")
	}
	if gotPassword != "pass" {
		t.Fatalf("validator password = %q, want %q", gotPassword, "pass")
	}
}

func TestProxyBasicAuthFunc_NilValidatorRejectsCONNECT(t *testing.T) {
	proxy := newTestProxy(t)
	mustUse(t, proxy, ProxyBasicAuthFunc(nil))

	req := httptest.NewRequest(http.MethodConnect, "//example.com:443", nil)
	req.Host = "example.com:443"
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusProxyAuthRequired {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusProxyAuthRequired)
	}
}
