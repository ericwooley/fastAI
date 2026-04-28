package e2e_test

import (
	"strings"
	"testing"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/agent/githubmodels"
	"github.com/ericwooley/fastAI/internal/commandexec"
	appsession "github.com/ericwooley/fastAI/internal/session"
	"github.com/ericwooley/fastAI/internal/workspace"
)

func TestCLISessionContinuationAndCommandFailure(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	deps := deps(t, repo, agent.NewLocalRunner(workspace.NewEditor(repo), commandexec.NewExecutor(repo), githubmodels.NewValidator(nil, "", "", githubmodels.WithOfflineValidation())))
	if code, _, errOut := execute(t, []string{"login"}, deps); code != 0 {
		t.Fatalf("login code=%d err=%q", code, errOut)
	}
	code, out, errOut := execute(t, []string{"--model", "github:gpt-4.1", "--session", "follow", "run command printf ok"}, deps)
	if code != 0 || out != "" || !strings.Contains(errOut, "command: printf ok exit=0") {
		t.Fatalf("explicit session first run should succeed: code=%d out=%q err=%q", code, out, errOut)
	}
	wantSessionID := appsession.HashSessionID("follow")
	if !strings.Contains(errOut, "session: "+wantSessionID) {
		t.Fatalf("stderr missing hashed session id %q in %q", wantSessionID, errOut)
	}
	code, out, errOut = execute(t, []string{"--model", "github:gpt-4.1", "run command printf ok"}, deps)
	if code != 0 || out != "" || !strings.Contains(errOut, "command: printf ok exit=0") {
		t.Fatalf("new run code=%d out=%q err=%q", code, out, errOut)
	}
	if strings.Contains(errOut, "thinking: ") {
		t.Fatalf("stderr unexpectedly contains thinking prefix: %q", errOut)
	}
	code, out, errOut = execute(t, []string{"--model", "github:gpt-4.1", "--session", "follow", "run command exit 7"}, deps)
	if code != 1 || !strings.Contains(errOut, "agent execution failed") {
		t.Fatalf("failed command code=%d out=%q err=%q", code, out, errOut)
	}
}
