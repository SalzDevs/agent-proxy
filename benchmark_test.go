package groxy

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func benchmarkProxy(b *testing.B) *Proxy {
	b.Helper()

	p, err := New(Config{Addr: "127.0.0.1:8080"})
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}

	return p
}

func benchmarkUse(b *testing.B, p *Proxy, middleware ...Middleware) {
	b.Helper()

	if err := p.Use(middleware...); err != nil {
		b.Fatalf("Use() error = %v", err)
	}
}

func BenchmarkForwardHTTP_GET(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	p := benchmarkProxy(b)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, upstream.URL, nil)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)
	}
}

func BenchmarkForwardHTTP_POSTSmallBody(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	p := benchmarkProxy(b)
	body := strings.Repeat("a", 1024)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, upstream.URL, strings.NewReader(body))
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)
	}
}

func BenchmarkForwardHTTP_POSTLargeBody(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	p := benchmarkProxy(b)
	body := strings.Repeat("a", 1024*1024)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, upstream.URL, strings.NewReader(body))
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)
	}
}

func BenchmarkForwardHTTP_MultipleHooks(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	p := benchmarkProxy(b)
	benchmarkUse(b, p,
		AddRequestHeader("X-One", "1"),
		AddRequestHeader("X-Two", "2"),
		RemoveRequestHeader("X-Remove"),
		AddResponseHeader("X-Three", "3"),
		RemoveResponseHeader("X-Remove-Response"),
	)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, upstream.URL, nil)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)
	}
}

func BenchmarkTransformRequestBody_Large(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	p := benchmarkProxy(b)
	benchmarkUse(b, p, TransformRequestBody(func(body []byte) ([]byte, error) {
		return bytes.ReplaceAll(body, []byte("a"), []byte("b")), nil
	}))
	body := strings.Repeat("a", 1024*1024)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, upstream.URL, strings.NewReader(body))
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)
	}
}

func BenchmarkTransformResponseBody_Large(b *testing.B) {
	body := strings.Repeat("a", 1024*1024)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer upstream.Close()

	p := benchmarkProxy(b)
	benchmarkUse(b, p, TransformResponseBody(func(body []byte) ([]byte, error) {
		return bytes.ReplaceAll(body, []byte("a"), []byte("b")), nil
	}))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, upstream.URL, nil)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)
	}
}

func BenchmarkBlockHost(b *testing.B) {
	p := benchmarkProxy(b)
	benchmarkUse(b, p, BlockHost("blocked.example", http.StatusForbidden, "blocked"))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "http://blocked.example/", nil)
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)
	}
}

func BenchmarkCONNECT_TunnelSmallPayload(b *testing.B) {
	benchmarkCONNECTTunnel(b, []byte("ping"))
}

func BenchmarkCONNECT_TunnelLargePayload(b *testing.B) {
	benchmarkCONNECTTunnel(b, bytes.Repeat([]byte("a"), 1024*1024))
}

func benchmarkCONNECTTunnel(b *testing.B, payload []byte) {
	upstreamListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("net.Listen() upstream error = %v", err)
	}
	defer upstreamListener.Close()

	go func() {
		for {
			conn, err := upstreamListener.Accept()
			if err != nil {
				return
			}

			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}(conn)
		}
	}()

	proxyAddr := freeAddrForBenchmark(b)
	p, err := New(Config{Addr: proxyAddr})
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}

	startErrCh := make(chan error, 1)
	go func() {
		startErrCh <- p.Start()
	}()
	waitForTCPBenchmark(b, proxyAddr)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = p.Shutdown(ctx)
		_ = <-startErrCh
	}()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := net.Dial("tcp", proxyAddr)
		if err != nil {
			b.Fatalf("net.Dial() proxy error = %v", err)
		}

		reader := bufio.NewReader(conn)
		connectReq := "CONNECT " + upstreamListener.Addr().String() + " HTTP/1.1\r\nHost: " + upstreamListener.Addr().String() + "\r\n\r\n"
		if _, err := conn.Write([]byte(connectReq)); err != nil {
			_ = conn.Close()
			b.Fatalf("write CONNECT error = %v", err)
		}

		statusLine, err := reader.ReadString('\n')
		if err != nil {
			_ = conn.Close()
			b.Fatalf("read CONNECT status error = %v", err)
		}
		if !strings.Contains(statusLine, "200 Connection Established") {
			_ = conn.Close()
			b.Fatalf("unexpected CONNECT status = %q", statusLine)
		}
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				_ = conn.Close()
				b.Fatalf("read CONNECT header error = %v", err)
			}
			if line == "\r\n" {
				break
			}
		}

		if _, err := conn.Write(payload); err != nil {
			_ = conn.Close()
			b.Fatalf("write payload error = %v", err)
		}

		buf := make([]byte, len(payload))
		if _, err := io.ReadFull(reader, buf); err != nil {
			_ = conn.Close()
			b.Fatalf("read payload error = %v", err)
		}

		_ = conn.Close()
	}
}

func freeAddrForBenchmark(b *testing.B) string {
	b.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("net.Listen() error = %v", err)
	}
	defer l.Close()

	return l.Addr().String()
}

func waitForTCPBenchmark(b *testing.B, addr string) {
	b.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	b.Fatalf("timed out waiting for %s to accept connections", addr)
}
