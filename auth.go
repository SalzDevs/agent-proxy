package groxy

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
)

const defaultProxyAuthRealm = "Groxy"

type proxyAuthAlreadyCheckedContextKey struct{}

func withProxyAuthAlreadyChecked(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), proxyAuthAlreadyCheckedContextKey{}, true)
	return r.WithContext(ctx)
}

func proxyAuthAlreadyChecked(r *http.Request) bool {
	checked, _ := r.Context().Value(proxyAuthAlreadyCheckedContextKey{}).(bool)
	return checked
}

func parseProxyBasicAuth(header string) (username, password string, ok bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Basic") {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", false
	}

	username, password, ok = strings.Cut(string(decoded), ":")
	if !ok {
		return "", "", false
	}

	return username, password, true
}

func writeProxyAuthRequired(w http.ResponseWriter, realm string) {
	if realm == "" {
		realm = defaultProxyAuthRealm
	}

	w.Header().Set("Proxy-Authenticate", `Basic realm="`+escapeAuthRealm(realm)+`"`)
	http.Error(w, "proxy authentication required", http.StatusProxyAuthRequired)
}

func escapeAuthRealm(realm string) string {
	realm = strings.ReplaceAll(realm, `\`, `\\`)
	realm = strings.ReplaceAll(realm, `"`, `\"`)
	return realm
}
