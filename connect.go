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
		if block, ok := blockError(err); ok {
			writeBlock(w, block)
			return
		}

		http.Error(w, "connect hook failed", http.StatusForbidden)
		return
	}

	if p.shouldInspectCONNECT(target) {
		p.inspectCONNECT(w, r, target)
		return
	}

	p.tunnelCONNECT(w, r, target)
}

func (p *Proxy) shouldInspectCONNECT(host string) bool {
	inspection := p.config.HTTPSInspection
	return inspection != nil && inspection.Intercept != nil && inspection.Intercept(host)
}

func (p *Proxy) tunnelCONNECT(w http.ResponseWriter, r *http.Request, target string) {
	upstreamConn, err := p.dialCONNECTTarget(r, target)
	if err != nil {
		http.Error(w, "failed to connect to target", http.StatusBadGateway)
		return
	}
	defer upstreamConn.Close()

	clientConn, err := hijackCONNECT(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	if err := writeCONNECTEstablished(clientConn); err != nil {
		return
	}

	copyCONNECTTunnel(clientConn, upstreamConn)
}

func (p *Proxy) inspectCONNECT(w http.ResponseWriter, r *http.Request, target string) {
	if p.config.HTTPSInspection != nil && p.config.HTTPSInspection.PassthroughOnError {
		p.logger.Printf("HTTPS inspection for %s is not implemented yet; falling back to tunnel", target)
		p.tunnelCONNECT(w, r, target)
		return
	}

	http.Error(w, "HTTPS inspection is not implemented yet", http.StatusNotImplemented)
}

func (p *Proxy) dialCONNECTTarget(r *http.Request, target string) (net.Conn, error) {
	dialer := net.Dialer{Timeout: p.config.Timeouts.Dial}
	return dialer.DialContext(r.Context(), "tcp", target)
}

func hijackCONNECT(w http.ResponseWriter) (net.Conn, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return nil, errHijackingNotSupported
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		return nil, err
	}

	return clientConn, nil
}

func writeCONNECTEstablished(conn net.Conn) error {
	_, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	return err
}

func copyCONNECTTunnel(clientConn, upstreamConn net.Conn) {
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
