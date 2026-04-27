package e2e_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/auth"
	"github.com/ericwooley/fastAI/internal/cli"
	appsession "github.com/ericwooley/fastAI/internal/session"
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
	seen   []agent.Request
}

func (r *fakeRunner) Run(_ context.Context, req agent.Request) (agent.Result, error) {
	r.seen = append(r.seen, req)
	if r.result.SessionID == "" {
		r.result.SessionID = req.SessionID
	}
	if r.result.Model == "" {
		r.result.Model = req.Model
	}
	return r.result, r.err
}

func execute(t *testing.T, args []string, deps cli.Dependencies) (int, string, string) {
	t.Helper()
	var out bytes.Buffer
	var errOut bytes.Buffer
	deps.Out = &out
	deps.Err = &errOut
	code := cli.Execute(context.Background(), args, deps)
	return code, out.String(), errOut.String()
}

func deps(t *testing.T, repo string, runner agent.Runner) cli.Dependencies {
	t.Helper()
	config := t.TempDir()
	return cli.Dependencies{
		AuthStore:      auth.NewFileStore(filepath.Join(config, "auth.json")),
		Authenticator:  auth.StaticAuthenticator{Account: auth.Account{Provider: auth.ProviderGitHubCopilot, AccessToken: "token", Login: "octo"}},
		SessionService: appsession.NewService(appsession.NewFileStore(filepath.Join(config, "sessions")), func() time.Time { return time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC) }),
		Runner:         runner,
		RepoRoot:       repo,
		Now:            func() time.Time { return time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC) },
	}
}
