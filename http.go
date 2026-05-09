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

	resp, err := p.client.Do(outReq)
	if err != nil {
		http.Error(w, "failed to reach upstream server", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	removeHopByHopHeaders(resp.Header)

	for k, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
