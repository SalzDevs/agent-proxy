package groxy

import (
	"io"
	"net"
	"net/http"
)

func (p *Proxy) runConnectHooks(host string) error {
	ctx := &ConnectContext{Host: host}
	for _, hook := range p.connectHooks {
		if err := hook(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (p *Proxy) handleCONNECT(w http.ResponseWriter, r *http.Request) {
	target := r.Host
	if target == "" {
		http.Error(w, "CONNECT request missing target host", http.StatusBadRequest)
		return
	}

	if err := p.runConnectHooks(target); err != nil {
		http.Error(w, "connect hook failed", http.StatusForbidden)
		return
	}

	dialer := net.Dialer{Timeout: p.config.Timeouts.Dial}
	upstreamConn, err := dialer.DialContext(r.Context(), "tcp", target)
	if err != nil {
		http.Error(w, "failed to connect to target", http.StatusBadGateway)
		return
	}
	defer upstreamConn.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "failed to hijack connection", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		return
	}

	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(upstreamConn, clientConn)
		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(clientConn, upstreamConn)
		done <- struct{}{}
	}()

	<-done
}
