package groxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type capturedRequest struct {
	Method             string
	Path               string
	Body               string
	XTest              string
	ContentType        string
	Connection         string
	ProxyConnection    string
	ProxyAuthenticate  string
	ProxyAuthorization string
}

func newTestProxy(t *testing.T) *Proxy {
	t.Helper()

	p, err := New(Config{Addr: "127.0.0.1:8080"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return p
}

func newTestProxyAtAddr(t *testing.T, addr string) *Proxy {
	t.Helper()

	p, err := New(Config{Addr: addr})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return p
}

func freeAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer l.Close()

	return l.Addr().String()
}

func waitForTCP(t *testing.T, addr string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s to accept connections", addr)
}

func testCA(t *testing.T) *CA {
	t.Helper()

	ca, err := NewCA(CAConfig{CommonName: "Test Groxy CA"})
	if err != nil {
		t.Fatalf("NewCA() error = %v", err)
	}

	return ca
}

func mustUse(t *testing.T, p *Proxy, middleware ...Middleware) {
	t.Helper()

	if err := p.Use(middleware...); err != nil {
		t.Fatalf("Use() error = %v", err)
	}
}

func mustOnRequest(t *testing.T, p *Proxy, fn RequestHook) {
	t.Helper()

	if err := p.OnRequest(fn); err != nil {
		t.Fatalf("OnRequest() error = %v", err)
	}
}

func mustOnResponse(t *testing.T, p *Proxy, fn ResponseHook) {
	t.Helper()

	if err := p.OnResponse(fn); err != nil {
		t.Fatalf("OnResponse() error = %v", err)
	}
}

func mustOnConnect(t *testing.T, p *Proxy, fn ConnectHook) {
	t.Helper()

	if err := p.OnConnect(fn); err != nil {
		t.Fatalf("OnConnect() error = %v", err)
	}
}

func TestNew_RejectsEmptyAddr(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected error for empty address, got nil")
	}
}

func TestNew_RejectsInvalidAddr(t *testing.T) {
	cases := []Config{
		{Addr: "127.0.0.1"},
		{Addr: "127.0.0.1:99999"},
		{Addr: "127.0.0.1:abc"},
	}

	for _, tc := range cases {
		if _, err := New(tc); err == nil {
			t.Fatalf("expected error for addr %q, got nil", tc.Addr)
		}
	}
}

func TestDefaultTimeouts_AppliesMissingValues(t *testing.T) {
	customDial := 2 * time.Second

	p, err := New(Config{
		Addr: "127.0.0.1:8080",
		Timeouts: &Timeouts{
			Dial: customDial,
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	want := DefaultTimeouts()
	want.Dial = customDial

	if *p.config.Timeouts != want {
		t.Fatalf("proxy timeouts = %+v, want %+v", *p.config.Timeouts, want)
	}
}

func TestNew_RejectsNegativeTimeout(t *testing.T) {
	if _, err := New(Config{
		Addr: "127.0.0.1:8080",
		Timeouts: &Timeouts{
			Dial: -1 * time.Second,
		},
	}); err == nil {
		t.Fatal("expected error for negative timeout, got nil")
	}
}

func TestNew_RejectsNegativeMaxBodySize(t *testing.T) {
	if _, err := New(Config{
		Addr:        "127.0.0.1:8080",
		MaxBodySize: -1,
	}); err == nil {
		t.Fatal("expected error for negative max body size, got nil")
	}
}

func TestNew_RejectsHTTPSInspectionWithoutCA(t *testing.T) {
	_, err := New(Config{
		Addr: "127.0.0.1:8080",
		HTTPSInspection: &HTTPSInspectionConfig{
			Intercept: func(host string) bool { return true },
		},
	})
	if err == nil {
		t.Fatal("expected error for HTTPS inspection without CA, got nil")
	}
	if !strings.Contains(err.Error(), "CA is required") {
		t.Fatalf("error = %q, want CA requirement", err.Error())
	}
}

func TestNew_RejectsUninitializedHTTPSInspectionCA(t *testing.T) {
	_, err := New(Config{
		Addr: "127.0.0.1:8080",
		HTTPSInspection: &HTTPSInspectionConfig{
			CA:        &CA{},
			Intercept: func(host string) bool { return true },
		},
	})
	if err == nil {
		t.Fatal("expected error for uninitialized HTTPS inspection CA, got nil")
	}
	if !strings.Contains(err.Error(), "CA is not initialized") {
		t.Fatalf("error = %q, want initialized CA requirement", err.Error())
	}
}

func TestNew_RejectsHTTPSInspectionWithoutInterceptMatcher(t *testing.T) {
	_, err := New(Config{
		Addr: "127.0.0.1:8080",
		HTTPSInspection: &HTTPSInspectionConfig{
			CA: testCA(t),
		},
	})
	if err == nil {
		t.Fatal("expected error for HTTPS inspection without intercept matcher, got nil")
	}
	if !strings.Contains(err.Error(), "intercept matcher is required") {
		t.Fatalf("error = %q, want intercept matcher requirement", err.Error())
	}
}

func TestNew_RejectsNegativeHTTPSInspectionCertificateTTL(t *testing.T) {
	_, err := New(Config{
		Addr: "127.0.0.1:8080",
		HTTPSInspection: &HTTPSInspectionConfig{
			CA:             testCA(t),
			Intercept:      func(host string) bool { return true },
			CertificateTTL: -time.Second,
		},
	})
	if err == nil {
		t.Fatal("expected error for negative certificate TTL, got nil")
	}
	if !strings.Contains(err.Error(), "certificate TTL cannot be negative") {
		t.Fatalf("error = %q, want certificate TTL requirement", err.Error())
	}
}

func TestNew_AcceptsHTTPSInspectionConfig(t *testing.T) {
	p, err := New(Config{
		Addr: "127.0.0.1:8080",
		HTTPSInspection: &HTTPSInspectionConfig{
			CA:        testCA(t),
			Intercept: func(host string) bool { return host == "example.com:443" },
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if p.config.HTTPSInspection == nil {
		t.Fatal("expected HTTPS inspection config to be stored")
	}
}

func TestNew_InitializesInternalFields(t *testing.T) {
	cfg := Config{Addr: "127.0.0.1:8080"}

	p, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if p == nil {
		t.Fatal("expected proxy, got nil")
	}
	if p.config.Addr != cfg.Addr {
		t.Fatalf("proxy config addr = %q, want %q", p.config.Addr, cfg.Addr)
	}
	if p.config.Timeouts == nil {
		t.Fatal("expected proxy timeouts to be resolved")
	}
	if *p.config.Timeouts != DefaultTimeouts() {
		t.Fatalf("proxy timeouts = %+v, want %+v", *p.config.Timeouts, DefaultTimeouts())
	}
	if p.config.MaxBodySize != DefaultMaxBodySize {
		t.Fatalf("max body size = %d, want %d", p.config.MaxBodySize, DefaultMaxBodySize)
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
	if p.logger == nil {
		t.Fatal("expected logger to be initialized")
	}
	if p.server.Addr != cfg.Addr {
		t.Fatalf("server addr = %q, want %q", p.server.Addr, cfg.Addr)
	}
	if p.server.ReadHeaderTimeout != p.config.Timeouts.ReadHeader {
		t.Fatalf("server read header timeout = %s, want %s", p.server.ReadHeaderTimeout, p.config.Timeouts.ReadHeader)
	}
	if p.server.IdleTimeout != p.config.Timeouts.Idle {
		t.Fatalf("server idle timeout = %s, want %s", p.server.IdleTimeout, p.config.Timeouts.Idle)
	}
	if p.transport.TLSHandshakeTimeout != p.config.Timeouts.TLSHandshake {
		t.Fatalf("transport TLS handshake timeout = %s, want %s", p.transport.TLSHandshakeTimeout, p.config.Timeouts.TLSHandshake)
	}
	if p.transport.ResponseHeaderTimeout != p.config.Timeouts.ResponseHeader {
		t.Fatalf("transport response header timeout = %s, want %s", p.transport.ResponseHeaderTimeout, p.config.Timeouts.ResponseHeader)
	}
	if p.transport.IdleConnTimeout != p.config.Timeouts.IdleConn {
		t.Fatalf("transport idle conn timeout = %s, want %s", p.transport.IdleConnTimeout, p.config.Timeouts.IdleConn)
	}
	if !p.transport.DisableCompression {
		t.Fatal("expected transport compression to be disabled")
	}
	if p.client.Transport != p.transport {
		t.Fatalf("client transport = %#v, want %#v", p.client.Transport, p.transport)
	}
	if p.IsRunning() {
		t.Fatal("expected proxy to start stopped")
	}
}

func TestShutdown_ReturnsErrorWhenNotRunning(t *testing.T) {
	p := newTestProxy(t)

	if err := p.Shutdown(context.Background()); err == nil {
		t.Fatal("expected error when stopping a non-running proxy, got nil")
	}
}

func TestStartAndShutdown_Lifecycle(t *testing.T) {
	addr := freeAddr(t)
	p := newTestProxyAtAddr(t, addr)

	upstreamBody := "upstream alive"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(upstreamBody))
	}))
	defer upstream.Close()

	startErrCh := make(chan error, 1)
	go func() {
		startErrCh <- p.Start()
	}()

	waitForTCP(t, addr)
	if !p.IsRunning() {
		t.Fatal("expected proxy state to be true while running")
	}

	proxyURL, err := url.Parse("http://" + addr)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
	}

	resp, err := client.Get(upstream.URL)
	if err != nil {
		t.Fatalf("client.Get() error = %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("response status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if string(body) != upstreamBody {
		t.Fatalf("response body = %q, want %q", string(body), upstreamBody)
	}

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := p.Shutdown(stopCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	if err := <-startErrCh; err != nil {
		t.Fatalf("Start() returned error = %v", err)
	}
	if p.IsRunning() {
		t.Fatal("expected proxy state to be false after shutdown")
	}
}

func TestHandleConnect_ReturnsBadRequestForMissingHost(t *testing.T) {
	p := newTestProxy(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodConnect, "/", nil)
	req.Host = ""

	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestHandleConnect_TunnelsTCPData(t *testing.T) {
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
	p := newTestProxyAtAddr(t, proxyAddr)

	startErrCh := make(chan error, 1)
	go func() {
		startErrCh <- p.Start()
	}()

	waitForTCP(t, proxyAddr)

	clientConn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatalf("net.Dial() proxy error = %v", err)
	}
	defer clientConn.Close()
	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetDeadline() error = %v", err)
	}

	connectReq := "CONNECT " + upstreamListener.Addr().String() + " HTTP/1.1\r\nHost: " + upstreamListener.Addr().String() + "\r\n\r\n"
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

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := p.Shutdown(stopCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := <-startErrCh; err != nil {
		t.Fatalf("Start() returned error = %v", err)
	}
}

func TestShouldInspectCONNECT(t *testing.T) {
	p, err := New(Config{
		Addr: "127.0.0.1:8080",
		HTTPSInspection: &HTTPSInspectionConfig{
			CA:        testCA(t),
			Intercept: MatchHosts("example.com"),
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if !p.shouldInspectCONNECT("example.com:443") {
		t.Fatal("expected matching host to be inspected")
	}
	if p.shouldInspectCONNECT("other.example:443") {
		t.Fatal("expected non-matching host not to be inspected")
	}
}

func TestShouldInspectCONNECT_ReturnsFalseWhenDisabled(t *testing.T) {
	p := newTestProxy(t)

	if p.shouldInspectCONNECT("example.com:443") {
		t.Fatal("expected CONNECT inspection to be disabled")
	}
}

func TestCONNECT_UnmatchedInspectionHostUsesTunnel(t *testing.T) {
	upstreamListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
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
	p, err := New(Config{
		Addr: proxyAddr,
		HTTPSInspection: &HTTPSInspectionConfig{
			CA:        testCA(t),
			Intercept: MatchHosts("does-not-match.example"),
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	startErrCh := make(chan error, 1)
	go func() {
		startErrCh <- p.Start()
	}()

	waitForTCP(t, proxyAddr)

	clientConn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatalf("net.Dial() proxy error = %v", err)
	}
	defer clientConn.Close()
	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetDeadline() error = %v", err)
	}

	connectReq := "CONNECT " + upstreamListener.Addr().String() + " HTTP/1.1\r\nHost: " + upstreamListener.Addr().String() + "\r\n\r\n"
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

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := p.Shutdown(stopCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := <-startErrCh; err != nil {
		t.Fatalf("Start() returned error = %v", err)
	}
}

func newHTTPSInspectionTestProxy(t *testing.T, upstreamHost string) (*Proxy, *CA, string) {
	t.Helper()

	ca := testCA(t)
	proxyAddr := freeAddr(t)
	p, err := New(Config{
		Addr: proxyAddr,
		HTTPSInspection: &HTTPSInspectionConfig{
			CA:        ca,
			Intercept: MatchHosts(upstreamHost),
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test upstream servers use self-signed certificates. This keeps the test
	// focused on Groxy's client-side MITM certificate instead of upstream trust.
	p.transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return p, ca, proxyAddr
}

func startProxyForTest(t *testing.T, p *Proxy, proxyAddr string) chan error {
	t.Helper()

	startErrCh := make(chan error, 1)
	go func() {
		startErrCh <- p.Start()
	}()
	waitForTCP(t, proxyAddr)

	return startErrCh
}

func stopProxyForTest(t *testing.T, p *Proxy, startErrCh chan error) {
	t.Helper()

	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := p.Shutdown(stopCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := <-startErrCh; err != nil {
		t.Fatalf("Start() returned error = %v", err)
	}
}

func inspectedHTTPSRequest(t *testing.T, proxyAddr, upstreamHost, serverName, path string, ca *CA) *http.Response {
	t.Helper()

	clientConn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatalf("net.Dial() proxy error = %v", err)
	}
	if err := clientConn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetDeadline() error = %v", err)
	}

	connectReq := "CONNECT " + upstreamHost + " HTTP/1.1\r\nHost: " + upstreamHost + "\r\n\r\n"
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

	roots := x509.NewCertPool()
	roots.AddCert(ca.cert)
	tlsConn := tls.Client(clientConn, &tls.Config{
		ServerName: serverName,
		RootCAs:    roots,
		NextProtos: []string{"http/1.1"},
	})
	if err := tlsConn.Handshake(); err != nil {
		t.Fatalf("TLS handshake error = %v", err)
	}

	if _, err := tlsConn.Write([]byte("GET " + path + " HTTP/1.1\r\nHost: " + upstreamHost + "\r\nConnection: close\r\n\r\n")); err != nil {
		t.Fatalf("write HTTPS request error = %v", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(tlsConn), nil)
	if err != nil {
		t.Fatalf("ReadResponse() error = %v", err)
	}

	return resp
}

func TestInspectCONNECT_TransformsHTTPSResponseBody(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	defer upstream.Close()

	upstreamURL, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	p, ca, proxyAddr := newHTTPSInspectionTestProxy(t, upstreamURL.Hostname())
	mustUse(t, p, TransformResponseBody(func(body []byte) ([]byte, error) {
		return bytes.ToUpper(body), nil
	}))

	startErrCh := startProxyForTest(t, p, proxyAddr)
	resp := inspectedHTTPSRequest(t, proxyAddr, upstreamURL.Host, upstreamURL.Hostname(), "/", ca)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(body) != "HELLO" {
		t.Fatalf("body = %q, want %q", string(body), "HELLO")
	}

	stopProxyForTest(t, p, startErrCh)
}

func TestInspectCONNECT_RunsHTTPSRequestHook(t *testing.T) {
	seen := make(chan string, 1)
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.Header.Get("X-Groxy-HTTPS")
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	upstreamURL, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	p, ca, proxyAddr := newHTTPSInspectionTestProxy(t, upstreamURL.Hostname())
	mustOnRequest(t, p, func(ctx *RequestContext) error {
		ctx.Request.Header.Set("X-Groxy-HTTPS", "true")
		return nil
	})

	startErrCh := startProxyForTest(t, p, proxyAddr)
	resp := inspectedHTTPSRequest(t, proxyAddr, upstreamURL.Host, upstreamURL.Hostname(), "/", ca)
	_ = resp.Body.Close()

	if got := <-seen; got != "true" {
		t.Fatalf("upstream X-Groxy-HTTPS = %q, want %q", got, "true")
	}
	stopProxyForTest(t, p, startErrCh)
}

func TestInspectCONNECT_RunsHTTPSResponseHook(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	upstreamURL, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	p, ca, proxyAddr := newHTTPSInspectionTestProxy(t, upstreamURL.Hostname())
	mustOnResponse(t, p, func(ctx *ResponseContext) error {
		ctx.Response.Header.Set("X-Groxy-Response", "true")
		return nil
	})

	startErrCh := startProxyForTest(t, p, proxyAddr)
	resp := inspectedHTTPSRequest(t, proxyAddr, upstreamURL.Host, upstreamURL.Hostname(), "/", ca)
	defer resp.Body.Close()

	if got := resp.Header.Get("X-Groxy-Response"); got != "true" {
		t.Fatalf("response X-Groxy-Response = %q, want %q", got, "true")
	}
	stopProxyForTest(t, p, startErrCh)
}

func TestInspectCONNECT_BlockHostCanBlockHTTPS(t *testing.T) {
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("upstream should not be called")
	}))
	defer upstream.Close()

	upstreamURL, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	p, ca, proxyAddr := newHTTPSInspectionTestProxy(t, upstreamURL.Hostname())
	mustUse(t, p, BlockHost(upstreamURL.Hostname(), http.StatusForbidden, "blocked https"))

	startErrCh := startProxyForTest(t, p, proxyAddr)
	resp := inspectedHTTPSRequest(t, proxyAddr, upstreamURL.Host, upstreamURL.Hostname(), "/", ca)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(body), "blocked https") {
		t.Fatalf("body = %q, want block message", string(body))
	}
	stopProxyForTest(t, p, startErrCh)
}

func TestOnRequest_ModifiesUpstreamRequest(t *testing.T) {
	seen := make(chan string, 1)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.Header.Get("X-Request-Hook")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p := newTestProxy(t)
	mustOnRequest(t, p, func(ctx *RequestContext) error {
		ctx.Request.Header.Set("X-Request-Hook", "applied")
		return nil
	})

	req, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if got := <-seen; got != "applied" {
		t.Fatalf("upstream X-Request-Hook = %q, want %q", got, "applied")
	}
}

func TestMiddlewareNames(t *testing.T) {
	cases := []struct {
		middleware Middleware
		want       string
	}{
		{OnRequest(func(ctx *RequestContext) error { return nil }), "OnRequest"},
		{OnResponse(func(ctx *ResponseContext) error { return nil }), "OnResponse"},
		{OnConnect(func(ctx *ConnectContext) error { return nil }), "OnConnect"},
		{AddRequestHeader("X-Test", "true"), "AddRequestHeader"},
		{AddResponseHeader("X-Test", "true"), "AddResponseHeader"},
		{RemoveRequestHeader("X-Test"), "RemoveRequestHeader"},
		{RemoveResponseHeader("X-Test"), "RemoveResponseHeader"},
		{BlockHost("example.com", http.StatusForbidden, "blocked"), "BlockHost"},
		{BlockConnectHost("example.com", http.StatusForbidden, "blocked"), "BlockConnectHost"},
		{TransformRequestBody(func(body []byte) ([]byte, error) { return body, nil }), "TransformRequestBody"},
		{TransformResponseBody(func(body []byte) ([]byte, error) { return body, nil }), "TransformResponseBody"},
	}

	for _, tc := range cases {
		if got := tc.middleware.Name(); got != tc.want {
			t.Fatalf("middleware name = %q, want %q", got, tc.want)
		}
	}
}

func TestUse_ReturnsErrorAfterProxyStarted(t *testing.T) {
	p := newTestProxy(t)
	p.setRunning(true)
	defer p.setRunning(false)

	err := p.Use(AddRequestHeader("X-Test", "true"))
	if err == nil {
		t.Fatal("expected error when adding middleware after proxy started, got nil")
	}
	if !strings.Contains(err.Error(), "AddRequestHeader") {
		t.Fatalf("error = %q, want middleware name", err.Error())
	}
}

func TestAddAndRemoveRequestHeaderMiddleware(t *testing.T) {
	seen := make(chan capturedRequest, 1)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- capturedRequest{
			XTest:       r.Header.Get("X-Test"),
			ContentType: r.Header.Get("Content-Type"),
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p := newTestProxy(t)
	mustUse(t, p,
		AddRequestHeader("X-Test", "added"),
		RemoveRequestHeader("Content-Type"),
	)

	req, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	got := <-seen
	if got.XTest != "added" {
		t.Fatalf("upstream X-Test = %q, want %q", got.XTest, "added")
	}
	if got.ContentType != "" {
		t.Fatalf("upstream Content-Type = %q, want empty", got.ContentType)
	}
}

func TestAddAndRemoveResponseHeaderMiddleware(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "upstream")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p := newTestProxy(t)
	mustUse(t, p,
		AddResponseHeader("X-Groxy", "true"),
		RemoveResponseHeader("Server"),
	)

	req, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Groxy"); got != "true" {
		t.Fatalf("response X-Groxy = %q, want %q", got, "true")
	}
	if got := rec.Header().Get("Server"); got != "" {
		t.Fatalf("response Server = %q, want empty", got)
	}
}

func TestBlockHostMiddleware(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer upstream.Close()

	p := newTestProxy(t)
	mustUse(t, p, BlockHost("127.0.0.1", http.StatusForbidden, "blocked host"))

	req, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if !strings.Contains(rec.Body.String(), "blocked host") {
		t.Fatalf("body = %q, want block message", rec.Body.String())
	}
	if called {
		t.Fatal("expected upstream not to be called")
	}
}

func TestBlockConnectHostMiddleware(t *testing.T) {
	p := newTestProxy(t)
	mustUse(t, p, BlockConnectHost("example.com", http.StatusForbidden, "connect blocked"))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodConnect, "//example.com:443", nil)
	req.Host = "example.com:443"

	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if !strings.Contains(rec.Body.String(), "connect blocked") {
		t.Fatalf("body = %q, want block message", rec.Body.String())
	}
}

func TestTransformRequestBodyMiddleware(t *testing.T) {
	seen := make(chan string, 1)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen <- string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p := newTestProxy(t)
	mustUse(t, p, TransformRequestBody(func(body []byte) ([]byte, error) {
		return bytes.ToUpper(body), nil
	}))

	req, err := http.NewRequest(http.MethodPost, upstream.URL, strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if got := <-seen; got != "HELLO" {
		t.Fatalf("upstream body = %q, want %q", got, "HELLO")
	}
}

func TestTransformResponseBodyMiddleware(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	defer upstream.Close()

	p := newTestProxy(t)
	mustUse(t, p, TransformResponseBody(func(body []byte) ([]byte, error) {
		return bytes.ToUpper(body), nil
	}))

	req, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Body.String() != "HELLO" {
		t.Fatalf("response body = %q, want %q", rec.Body.String(), "HELLO")
	}
}

func TestTransformRequestBodyMiddleware_ReturnsError(t *testing.T) {
	p := newTestProxy(t)
	mustUse(t, p, TransformRequestBody(func(body []byte) ([]byte, error) {
		return nil, errors.New("transform failed")
	}))

	req, err := http.NewRequest(http.MethodPost, "http://example.com", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestBodyBytes_ReturnsErrorWhenRequestBodyTooLarge(t *testing.T) {
	p, err := New(Config{Addr: "127.0.0.1:8080", MaxBodySize: 2})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	mustUse(t, p, TransformRequestBody(func(body []byte) ([]byte, error) {
		return body, nil
	}))

	req, err := http.NewRequest(http.MethodPost, "http://example.com", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestBodyBytes_ReturnsErrorWhenResponseBodyTooLarge(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	defer upstream.Close()

	p, err := New(Config{Addr: "127.0.0.1:8080", MaxBodySize: 2})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	mustUse(t, p, TransformResponseBody(func(body []byte) ([]byte, error) {
		return body, nil
	}))

	req, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}

func TestRequestContext_BodyBytesAndSetBody(t *testing.T) {
	seen := make(chan string, 1)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen <- string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p := newTestProxy(t)
	mustOnRequest(t, p, func(ctx *RequestContext) error {
		body, err := ctx.BodyBytes()
		if err != nil {
			return err
		}

		ctx.SetBody(bytes.ToUpper(body))
		return nil
	})

	req, err := http.NewRequest(http.MethodPost, upstream.URL, strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if got := <-seen; got != "HELLO" {
		t.Fatalf("upstream body = %q, want %q", got, "HELLO")
	}
}

func TestResponseContext_BodyBytesAndSetBody(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))
	defer upstream.Close()

	p := newTestProxy(t)
	mustOnResponse(t, p, func(ctx *ResponseContext) error {
		body, err := ctx.BodyBytes()
		if err != nil {
			return err
		}

		ctx.SetBody(bytes.ToUpper(body))
		return nil
	})

	req, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Body.String() != "HELLO" {
		t.Fatalf("response body = %q, want %q", rec.Body.String(), "HELLO")
	}
	if rec.Header().Get("Content-Length") != "5" {
		t.Fatalf("Content-Length = %q, want %q", rec.Header().Get("Content-Length"), "5")
	}
}

func TestOnResponse_ModifiesDownstreamResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p := newTestProxy(t)
	mustOnResponse(t, p, func(ctx *ResponseContext) error {
		ctx.Response.Header.Set("X-Response-Hook", "applied")
		return nil
	})

	req, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Response-Hook"); got != "applied" {
		t.Fatalf("response X-Response-Hook = %q, want %q", got, "applied")
	}
}

func TestUse_AppliesMiddleware(t *testing.T) {
	seen := make(chan string, 1)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.Header.Get("X-Use")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p := newTestProxy(t)
	mustUse(t, p, OnRequest(func(ctx *RequestContext) error {
		ctx.Request.Header.Set("X-Use", "applied")
		return nil
	}))

	req, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if got := <-seen; got != "applied" {
		t.Fatalf("upstream X-Use = %q, want %q", got, "applied")
	}
}

func TestBlock_CreatesBlockResponse(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer upstream.Close()

	p := newTestProxy(t)
	mustOnRequest(t, p, func(ctx *RequestContext) error {
		return Block(http.StatusForbidden, "blocked by policy")
	})

	req, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if !strings.Contains(rec.Body.String(), "blocked by policy") {
		t.Fatalf("body = %q, want block message", rec.Body.String())
	}
	if called {
		t.Fatal("expected upstream not to be called")
	}
}

func TestOnResponse_CanBlockResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream response"))
	}))
	defer upstream.Close()

	p := newTestProxy(t)
	mustOnResponse(t, p, func(ctx *ResponseContext) error {
		return Block(http.StatusUnavailableForLegalReasons, "response blocked")
	})

	req, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnavailableForLegalReasons {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnavailableForLegalReasons)
	}
	if !strings.Contains(rec.Body.String(), "response blocked") {
		t.Fatalf("body = %q, want block message", rec.Body.String())
	}
}

func TestOnConnect_CanBlockTunnel(t *testing.T) {
	p := newTestProxy(t)
	mustOnConnect(t, p, func(ctx *ConnectContext) error {
		return Block(http.StatusForbidden, "connect blocked")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodConnect, "//example.com:443", nil)
	req.Host = "example.com:443"

	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if !strings.Contains(rec.Body.String(), "connect blocked") {
		t.Fatalf("body = %q, want block message", rec.Body.String())
	}
}

func TestOnRequest_ReturnsErrorBlocksForwarding(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer upstream.Close()

	p := newTestProxy(t)
	mustOnRequest(t, p, func(ctx *RequestContext) error {
		return errors.New("blocked")
	})

	req, err := http.NewRequest(http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if called {
		t.Fatal("expected upstream not to be called")
	}
}

func TestOnConnect_ReturnsErrorBlocksTunnel(t *testing.T) {
	p := newTestProxy(t)
	mustOnConnect(t, p, func(ctx *ConnectContext) error {
		return errors.New("blocked")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodConnect, "//example.com:443", nil)
	req.Host = "example.com:443"

	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
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
			Method:             r.Method,
			Path:               r.URL.RequestURI(),
			Body:               string(body),
			XTest:              r.Header.Get("X-Test"),
			Connection:         r.Header.Get("Connection"),
			ProxyConnection:    r.Header.Get("Proxy-Connection"),
			ProxyAuthenticate:  r.Header.Get("Proxy-Authenticate"),
			ProxyAuthorization: r.Header.Get("Proxy-Authorization"),
			ContentType:        r.Header.Get("Content-Type"),
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
	req.Header.Set("Proxy-Authenticate", "Basic realm=\"Groxy\"")
	req.Header.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")

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
	if got.ProxyAuthenticate != "" {
		t.Fatalf("upstream Proxy-Authenticate header = %q, want empty", got.ProxyAuthenticate)
	}
	if got.ProxyAuthorization != "" {
		t.Fatalf("upstream Proxy-Authorization header = %q, want empty", got.ProxyAuthorization)
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
	h.Set("Proxy-Authenticate", "Basic realm=\"Groxy\"")
	h.Set("Proxy-Authorization", "Basic dXNlcjpwYXNz")
	h.Set("Upgrade", "websocket")
	h.Set("X-Keep", "yes")

	removeHopByHopHeaders(h)

	for _, header := range []string{
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
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
