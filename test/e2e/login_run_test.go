package e2e_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/auth"
)

func TestCLILoginAndSuccessfulRun(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	deps := deps(t, repo, runner)
	code, out, errOut := execute(t, []string{"login"}, deps)
	if code != 0 || !strings.Contains(out, "Login succeeded") || errOut != "" {
		t.Fatalf("login code=%d out=%q err=%q", code, out, errOut)
	}
	code, out, errOut = execute(t, []string{"--model", "github:gpt-4.1", "do work"}, deps)
	if code != 0 || strings.TrimSpace(out) != "ok" || !strings.Contains(errOut, "session:") || !strings.Contains(errOut, "provider: github-copilot") || !strings.Contains(errOut, "model: github:gpt-4.1") {
		t.Fatalf("run code=%d out=%q err=%q", code, out, errOut)
	}
	if strings.Contains(errOut, "thinking: ") {
		t.Fatalf("stderr unexpectedly contains thinking prefix: %q", errOut)
	}
	for _, forbidden := range []string{"Run completed successfully", "Session:", "Model:", "Summary:"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("stdout contains %q: %q", forbidden, out)
		}
	}
	if len(runner.seen) != 1 || runner.seen[0].Model != "github:gpt-4.1" || runner.seen[0].AccessToken != "token" {
		t.Fatalf("runner seen = %+v", runner.seen)
	}
	if _, err := auth.NewFileStore(filepath.Join(filepath.Dir(repo), "missing", "auth.json")).Load(context.Background()); err == nil {
		t.Fatalf("sanity check expected missing auth")
	}
}

func TestCLIMissingModelFailsValidation(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	code, _, errOut := execute(t, []string{"do work"}, deps(t, repo, &fakeRunner{}))
	if code != 2 || !strings.Contains(errOut, "--model is required") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestCLIProviderPrefixedModelRun(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "openrouter-token")
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	code, out, errOut := execute(t, []string{"--model", "openrouter/deepseek/deepseek-chat", "do work"}, deps(t, repo, runner))
	if code != 0 || strings.TrimSpace(out) != "ok" || !strings.Contains(errOut, "provider: openrouter") || !strings.Contains(errOut, "model: deepseek/deepseek-chat") {
		t.Fatalf("run code=%d out=%q err=%q", code, out, errOut)
	}
	if len(runner.seen) != 1 {
		t.Fatalf("runner calls = %+v", runner.seen)
	}
	seen := runner.seen[0]
	if seen.Provider != "openrouter" || seen.Model != "deepseek/deepseek-chat" || seen.AccessToken != "openrouter-token" || seen.PromptRunner == nil {
		t.Fatalf("runner saw unexpected request: %+v", seen)
	}
}

func TestCLIConflictingProviderAndModelPrefixFailsValidation(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	code, _, errOut := execute(t, []string{"--provider", "openai", "--model", "openrouter/deepseek-chat", "do work"}, deps(t, repo, runner))
	if code != 2 || !strings.Contains(errOut, "--provider conflicts with --model prefix") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	if len(runner.seen) != 0 {
		t.Fatalf("runner should not be called: %+v", runner.seen)
	}
}
