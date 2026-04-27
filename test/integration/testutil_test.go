package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/auth"
)

func tempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	return dir
}

type fakeRunner struct {
	result agent.Result
	err    error
	seen   agent.Request
}

func (r *fakeRunner) Run(_ context.Context, req agent.Request) (agent.Result, error) {
	r.seen = req
	if r.result.SessionID == "" {
		r.result.SessionID = req.SessionID
	}
	if r.result.Model == "" {
		r.result.Model = req.Model
	}
	return r.result, r.err
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func account() auth.Account {
	return auth.Account{Provider: auth.ProviderGitHubCopilot, AccessToken: "token", OAuthClientID: auth.CopilotClientID, Login: "octo"}
}

func modelServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" && r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("missing auth header: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}
