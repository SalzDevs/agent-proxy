package groxy

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestProxyBasicAuth_AllowsValidCONNECTTunnel(t *testing.T) {
	upstreamListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() upstream error = %v", err)
	}
	defer upstreamListener.Close()

	upstreamErrCh := make(chan error, 1)
	go func() {
		conn, err := upstreamListener.Accept()
		if err != nil {
			upstreamErrCh <- err
			return
		}
		defer conn.Close()

		buf := make([]byte, len("ping"))
		if _, err := io.ReadFull(conn, buf); err != nil {
			upstreamErrCh <- err
			return
		}
		if string(buf) != "ping" {
			upstreamErrCh <- nil
			return
		}
		_, err = conn.Write([]byte("pong"))
		upstreamErrCh <- err
	}()

	proxyAddr := freeAddr(t)
	proxy := newTestProxyAtAddr(t, proxyAddr)
	mustUse(t, proxy, ProxyBasicAuth("user", "pass"))
	startErrCh := startProxyForTest(t, proxy, proxyAddr)

	clientConn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatalf("net.Dial() proxy error = %v", err)
	}
	defer clientConn.Close()
	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetDeadline() error = %v", err)
	}

	connectReq := "CONNECT " + upstreamListener.Addr().String() + " HTTP/1.1\r\n" +
		"Host: " + upstreamListener.Addr().String() + "\r\n" +
		"Proxy-Authorization: Basic dXNlcjpwYXNz\r\n" +
		"\r\n"
	if _, err := clientConn.Write([]byte(connectReq)); err != nil {
		t.Fatalf("write CONNECT request error = %v", err)
	}

	reader := bufio.NewReader(clientConn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read CONNECT status error = %v", err)
	}
	if !strings.Contains(statusLine, "200 Connection Established") {
		t.Fatalf("CONNECT status line = %q, want 200 Connection Established", statusLine)
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read CONNECT header error = %v", err)
		}
		if line == "\r\n" {
			break
		}
	}

	if _, err := clientConn.Write([]byte("ping")); err != nil {
		t.Fatalf("write tunnel payload error = %v", err)
	}
	buf := make([]byte, len("pong"))
	if _, err := io.ReadFull(reader, buf); err != nil {
		t.Fatalf("read tunnel payload error = %v", err)
	}
	if string(buf) != "pong" {
		t.Fatalf("tunnel response = %q, want %q", string(buf), "pong")
	}
	if err := <-upstreamErrCh; err != nil {
		t.Fatalf("upstream error = %v", err)
	}

	stopProxyForTest(t, proxy, startErrCh)
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
