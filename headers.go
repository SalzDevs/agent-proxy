package groxy

import "net/http"

func removeHopByHopHeaders(h http.Header) {
	headers := []string{
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	}

	for _, header := range headers {
		h.Del(header)
	}
}
