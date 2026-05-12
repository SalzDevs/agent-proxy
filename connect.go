package groxy

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strings"
)

func (p *Proxy) runConnectHooks(host string, r *http.Request) error {
	ctx := &ConnectContext{Host: host, Request: r}
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

	if err := p.runConnectHooks(target, r); err != nil {
		if auth, ok := proxyAuthRequired(err); ok {
			writeProxyAuthRequired(w, auth.realm)
			return
		}

		if block, ok := blockError(err); ok {
			writeBlock(w, block)
			return
		}

		http.Error(w, "connect hook failed", http.StatusForbidden)
		return
	}

	if p.shouldInspectCONNECT(target) {
		p.logger.Printf("Inspecting HTTPS CONNECT tunnel for %s", target)
		p.inspectCONNECT(w, r, target)
		return
	}

	p.logger.Printf("Tunneling CONNECT request for %s", target)
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
	if p.certCache == nil {
		p.handleInspectionSetupError(w, r, target, "HTTPS inspection certificate cache is not initialized")
		return
	}

	cert, err := p.certCache.get(target)
	if err != nil {
		p.handleInspectionSetupError(w, r, target, err.Error())
		return
	}

	clientConn, err := hijackCONNECT(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	if err := writeCONNECTEstablished(clientConn); err != nil {
		return
	}

	tlsConn := tls.Server(clientConn, &tls.Config{
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"http/1.1"},
		Certificates: []tls.Certificate{*cert},
	})
	defer tlsConn.Close()

	if err := tlsConn.Handshake(); err != nil {
		p.logger.Printf("HTTPS inspection handshake failed for %s: %v", target, err)
		return
	}

	p.serveInspectedHTTPS(tlsConn, target)
}

func (p *Proxy) handleInspectionSetupError(w http.ResponseWriter, r *http.Request, target, message string) {
	if p.config.HTTPSInspection != nil && p.config.HTTPSInspection.PassthroughOnError {
		p.logger.Printf("HTTPS inspection setup for %s failed: %s; falling back to tunnel", target, message)
		p.tunnelCONNECT(w, r, target)
		return
	}

	p.logger.Printf("HTTPS inspection setup for %s failed closed: %s", target, message)
	http.Error(w, message, http.StatusBadGateway)
}

func (p *Proxy) serveInspectedHTTPS(conn net.Conn, target string) {
	reader := bufio.NewReader(conn)
	for {
		req, err := http.ReadRequest(reader)
		if err != nil {
			if err != io.EOF {
				p.logger.Printf("failed to read inspected HTTPS request for %s: %v", target, err)
			}
			return
		}

		resp, err := p.forwardRequest(req, "https", target)
		_ = req.Body.Close()
		if err != nil {
			if !p.writeInspectedForwardError(conn, err) {
				return
			}
			continue
		}

		if err := resp.Write(conn); err != nil {
			_ = resp.Body.Close()
			return
		}
		_ = resp.Body.Close()

		if !shouldKeepInspectedConnection(req, resp) {
			return
		}
	}
}

func (p *Proxy) writeInspectedForwardError(conn net.Conn, err error) bool {
	status := http.StatusBadGateway
	message := "failed to reach upstream server"

	if block, ok := blockError(err); ok {
		status = block.StatusCode
		message = block.Message
	} else if forward, ok := err.(forwardError); ok {
		status = forward.status
		message = forward.message
	}

	resp := &http.Response{
		StatusCode:    status,
		Status:        http.StatusText(status),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        make(http.Header),
		Body:          io.NopCloser(io.LimitReader(strings.NewReader(message+"\n"), int64(len(message)+1))),
		ContentLength: int64(len(message) + 1),
	}
	resp.Header.Set("Content-Type", "text/plain; charset=utf-8")

	return resp.Write(conn) == nil
}

func shouldKeepInspectedConnection(req *http.Request, resp *http.Response) bool {
	return !req.Close && !resp.Close
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
