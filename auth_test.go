package groxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyAuthAlreadyChecked(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	if proxyAuthAlreadyChecked(req) {
		t.Fatal("new request is marked as already checked")
	}

	checkedReq := withProxyAuthAlreadyChecked(req)
	if !proxyAuthAlreadyChecked(checkedReq) {
		t.Fatal("request is not marked as already checked")
	}
	if proxyAuthAlreadyChecked(req) {
		t.Fatal("original request was mutated")
	}
}

func TestParseProxyBasicAuth(t *testing.T) {
	cases := []struct {
		name         string
		header       string
		wantUsername string
		wantPassword string
		wantOK       bool
	}{
		{
			name:         "valid",
			header:       "Basic dXNlcjpwYXNz",
			wantUsername: "user",
			wantPassword: "pass",
			wantOK:       true,
		},
		{
			name:         "case insensitive scheme",
			header:       "basic dXNlcjpwYXNz",
			wantUsername: "user",
			wantPassword: "pass",
			wantOK:       true,
		},
		{
			name:         "password may contain colon",
			header:       "Basic dXNlcjpwYTpzcw==",
			wantUsername: "user",
			wantPassword: "pa:ss",
			wantOK:       true,
		},
		{
			name:   "missing header",
			header: "",
		},
		{
			name:   "wrong scheme",
			header: "Bearer token",
		},
		{
			name:   "invalid base64",
			header: "Basic not-base64",
		},
		{
			name:   "missing colon",
			header: "Basic dXNlcg==",
		},
		{
			name:   "extra fields",
			header: "Basic dXNlcjpwYXNz extra",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotUsername, gotPassword, gotOK := parseProxyBasicAuth(tc.header)
			if gotOK != tc.wantOK {
				t.Fatalf("ok = %v, want %v", gotOK, tc.wantOK)
			}
			if gotUsername != tc.wantUsername {
				t.Fatalf("username = %q, want %q", gotUsername, tc.wantUsername)
			}
			if gotPassword != tc.wantPassword {
				t.Fatalf("password = %q, want %q", gotPassword, tc.wantPassword)
			}
		})
	}
}

func TestWriteProxyAuthRequired(t *testing.T) {
	rec := httptest.NewRecorder()

	writeProxyAuthRequired(rec, "")

	if rec.Code != http.StatusProxyAuthRequired {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusProxyAuthRequired)
	}
	if got := rec.Header().Get("Proxy-Authenticate"); got != `Basic realm="Groxy"` {
		t.Fatalf("Proxy-Authenticate = %q, want default realm", got)
	}
}

func TestWriteProxyAuthRequired_EscapesRealm(t *testing.T) {
	rec := httptest.NewRecorder()

	writeProxyAuthRequired(rec, `Groxy "Local" \ Proxy`)

	want := `Basic realm="Groxy \"Local\" \\ Proxy"`
	if got := rec.Header().Get("Proxy-Authenticate"); got != want {
		t.Fatalf("Proxy-Authenticate = %q, want %q", got, want)
	}
}
