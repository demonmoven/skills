package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"code.byted.org/lihuanyu.0w0/gloop/internal/version"
)

func TestDashboardURLUsesExplicitExternalHost(t *testing.T) {
	t.Setenv("GLOOP_DASHBOARD_EXTERNAL_HOST", "10.4.4.239")
	t.Setenv("WEB_EXTERNAL_HOST", "10.0.0.1")

	tok := &TokenStore{Token: "tok", Port: 37317, BindHost: "0.0.0.0"}
	got := tok.DashboardURL()
	want := "http://10.4.4.239:37317/dashboard?t=tok"
	if got != want {
		t.Fatalf("DashboardURL() = %q, want %q", got, want)
	}
}

func TestDashboardURLUsesWebExternalHostFallback(t *testing.T) {
	t.Setenv("WEB_EXTERNAL_HOST", "10.0.0.1")

	tok := &TokenStore{Token: "tok", Port: 37317, BindHost: "0.0.0.0"}
	got := tok.DashboardURL()
	want := "http://10.0.0.1:37317/dashboard?t=tok"
	if got != want {
		t.Fatalf("DashboardURL() = %q, want %q", got, want)
	}
}

func TestDashboardURLKeepsConcreteBindHost(t *testing.T) {
	tok := &TokenStore{Token: "tok", Port: 37317, BindHost: "10.4.4.239"}
	got := tok.DashboardURL()
	want := "http://10.4.4.239:37317/dashboard?t=tok"
	if got != want {
		t.Fatalf("DashboardURL() = %q, want %q", got, want)
	}
}

func TestDashboardURLDoesNotAdvertiseWildcardHost(t *testing.T) {
	tok := &TokenStore{Token: "tok", Port: 37317, BindHost: "0.0.0.0"}
	got := tok.DashboardURL()
	if strings.Contains(got, "0.0.0.0") {
		t.Fatalf("DashboardURL() advertised wildcard host: %q", got)
	}
}

func TestGenerateKeepsPersistedPortAndBindHost(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	tok, err := Generate(dir, "127.0.0.1", 37321, false)
	if err != nil {
		t.Fatal(err)
	}
	if tok.Port != 37321 || tok.BindHost != "127.0.0.1" {
		t.Fatalf("initial token = port %d host %q", tok.Port, tok.BindHost)
	}
	if info, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	} else if info.Mode().Perm() != 0o700 {
		t.Fatalf("token dir mode = %o, want 700", info.Mode().Perm())
	}

	loaded, err := Generate(dir, "0.0.0.0", 37317, false)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Token != tok.Token {
		t.Fatalf("token was regenerated")
	}
	if loaded.Port != 37321 || loaded.BindHost != "127.0.0.1" {
		t.Fatalf("persisted endpoint overwritten: port %d host %q", loaded.Port, loaded.BindHost)
	}
}

func TestTokenStoreSavePersistsEndpoint(t *testing.T) {
	dir := t.TempDir()
	tok := &TokenStore{Token: "tok", Port: 37322, BindHost: "127.0.0.1"}
	if err := tok.Save(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, version.TokenFileName)); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Port != tok.Port || loaded.BindHost != tok.BindHost || loaded.Token != tok.Token {
		t.Fatalf("loaded token = %#v, want %#v", loaded, tok)
	}
}

func TestMiddlewareMissingTokenReturnsServerError(t *testing.T) {
	called := false
	h := Middleware(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/quests", nil))

	if called {
		t.Fatal("next handler should not be called when token store is missing")
	}
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(rr.Body.String(), "local token not generated") {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}
