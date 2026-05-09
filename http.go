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
	ctx := &RequestContext{Request: r}
	for _, hook := range p.requestHooks {
		if err := hook(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (p *Proxy) runResponseHooks(req *http.Request, resp *http.Response) error {
	ctx := &ResponseContext{Request: req, Response: resp}
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

	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, "failed to reach upstream request", http.StatusInternalServerError)
		return
	}

	outReq.Header = r.Header.Clone()
	removeHopByHopHeaders(outReq.Header)
	outReq.Host = r.URL.Host

	if err := p.runRequestHooks(outReq); err != nil {
		http.Error(w, "request hook failed", http.StatusInternalServerError)
		return
	}

	resp, err := p.client.Do(outReq)
	if err != nil {
		http.Error(w, "failed to reach upstream server", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if err := p.runResponseHooks(outReq, resp); err != nil {
		http.Error(w, "response hook failed", http.StatusBadGateway)
		return
	}

	removeHopByHopHeaders(resp.Header)

	for k, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
