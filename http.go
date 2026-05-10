package groxy

import (
	"io"
	"net/http"
)

// ServeHTTP handles incoming proxy requests.
//
// ServeHTTP allows Proxy to satisfy http.Handler, so it can be mounted on a
// custom http.Server instead of being started with Start.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.logger.Printf("Received request: %s %s", r.Method, r.URL.String())
	if r.Method == http.MethodConnect {
		p.handleCONNECT(w, r)
		return
	}

	p.handleForwardHTTP(w, r)
}

func (p *Proxy) runRequestHooks(r *http.Request) error {
	ctx := &RequestContext{Request: r, maxBodySize: p.config.MaxBodySize}
	for _, hook := range p.requestHooks {
		if err := hook(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (p *Proxy) runResponseHooks(req *http.Request, resp *http.Response) error {
	ctx := &ResponseContext{Request: req, Response: resp, maxBodySize: p.config.MaxBodySize}
	for _, hook := range p.responseHooks {
		if err := hook(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (p *Proxy) handleForwardHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL == nil || r.URL.Scheme == "" || r.URL.Host == "" {
		http.Error(w, "proxy request must contain an absolute URL", http.StatusBadRequest)
		return
	}

	resp, err := p.forwardRequest(r, r.URL.Scheme, r.URL.Host)
	if err != nil {
		p.writeForwardError(w, err)
		return
	}
	defer resp.Body.Close()

	writeForwardResponse(w, resp)
}

func (p *Proxy) forwardRequest(r *http.Request, scheme, host string) (*http.Response, error) {
	outReq, err := newForwardRequest(r, scheme, host)
	if err != nil {
		return nil, forwardError{status: http.StatusInternalServerError, message: "failed to reach upstream request", err: err}
	}

	if err := p.runRequestHooks(outReq); err != nil {
		if _, ok := blockError(err); ok {
			return nil, err
		}
		return nil, forwardError{status: http.StatusInternalServerError, message: "request hook failed", err: err}
	}

	resp, err := p.client.Do(outReq)
	if err != nil {
		return nil, forwardError{status: http.StatusBadGateway, message: "failed to reach upstream server", err: err}
	}

	if err := p.runResponseHooks(outReq, resp); err != nil {
		resp.Body.Close()
		if _, ok := blockError(err); ok {
			return nil, err
		}
		return nil, forwardError{status: http.StatusBadGateway, message: "response hook failed", err: err}
	}

	removeHopByHopHeaders(resp.Header)
	return resp, nil
}

func newForwardRequest(r *http.Request, scheme, host string) (*http.Request, error) {
	upstreamURL := *r.URL
	upstreamURL.Scheme = scheme
	upstreamURL.Host = host
	upstreamURL.User = nil

	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL.String(), r.Body)
	if err != nil {
		return nil, err
	}

	outReq.Header = r.Header.Clone()
	removeHopByHopHeaders(outReq.Header)
	outReq.Host = host

	return outReq, nil
}

func (p *Proxy) writeForwardError(w http.ResponseWriter, err error) {
	if block, ok := blockError(err); ok {
		writeBlock(w, block)
		return
	}

	if forward, ok := err.(forwardError); ok {
		http.Error(w, forward.message, forward.status)
		return
	}

	http.Error(w, "failed to reach upstream server", http.StatusBadGateway)
}

type forwardError struct {
	status  int
	message string
	err     error
}

func (e forwardError) Error() string {
	return e.err.Error()
}

func writeForwardResponse(w http.ResponseWriter, resp *http.Response) {
	for k, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
