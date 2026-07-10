package e2e_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/auth"
	appsession "github.com/ericwooley/fastAI/internal/session"
)

func TestCLILoginAndSuccessfulRun(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	deps := deps(t, repo, runner)
	code, out, errOut := execute(t, []string{"login", "copilot"}, deps)
	if code != 0 || !strings.Contains(out, "Login succeeded") || errOut != "" {
		t.Fatalf("login code=%d out=%q err=%q", code, out, errOut)
	}
	code, out, errOut = execute(t, []string{"--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all", "do work"}, deps)
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

func TestCLILoginRequiresProvider(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	code, _, errOut := execute(t, []string{"login"}, deps(t, repo, &fakeRunner{}))
	if code != 2 || !strings.Contains(errOut, "login provider is required") || !strings.Contains(errOut, "fastAI login copilot") {
		t.Fatalf("login code=%d err=%q", code, errOut)
	}
}

func TestCLIPassesPermissionsToRunner(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	deps := deps(t, repo, runner)
	if err := deps.AuthStore.Save(context.Background(), auth.Account{Provider: auth.ProviderGitHubCopilot, AccessToken: "token", OAuthClientID: auth.CopilotClientID}); err != nil {
		t.Fatalf("save auth: %v", err)
	}
	code, _, errOut := execute(t, []string{"--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "read,write", "do work"}, deps)
	if code != 0 {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	if len(runner.seen) != 1 || !runner.seen[0].Permissions.Read || !runner.seen[0].Permissions.Write || runner.seen[0].Permissions.Execute {
		t.Fatalf("runner saw permissions: %+v", runner.seen)
	}
}

func TestCLIRejectsInvalidPermissions(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	code, _, errOut := execute(t, []string{"--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "read,admin", "do work"}, deps(t, repo, runner))
	if code != 2 || !strings.Contains(errOut, "--permissions must be a comma-separated list") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	if len(runner.seen) != 0 {
		t.Fatalf("runner should not be called: %+v", runner.seen)
	}
}

func TestCLIUsesEnvironmentDefaults(t *testing.T) {
	t.Setenv("FASTAI_DEFAULT_PROVIDER", "github-copilot")
	t.Setenv("FASTAI_DEFAULT_MODEL", "github:gpt-5-mini")
	t.Setenv("FASTAI_DEFAULT_PERMISSIONS", "execute")
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	deps := deps(t, repo, runner)
	if err := deps.AuthStore.Save(context.Background(), auth.Account{Provider: auth.ProviderGitHubCopilot, AccessToken: "token", OAuthClientID: auth.CopilotClientID}); err != nil {
		t.Fatalf("save auth: %v", err)
	}
	code, _, errOut := execute(t, []string{"do work"}, deps)
	if code != 0 {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	if len(runner.seen) != 1 {
		t.Fatalf("runner calls = %+v", runner.seen)
	}
	seen := runner.seen[0]
	if seen.Provider != "github-copilot" || seen.Model != "github:gpt-5-mini" || !seen.Permissions.Execute || seen.Permissions.Read || seen.Permissions.Write {
		t.Fatalf("runner saw unexpected request: %+v", seen)
	}
}

func TestCLIEmptyPromptReadsFromEditor(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	deps := deps(t, repo, runner)
	deps.Editor = func(context.Context) (string, error) { return "prompt from editor", nil }
	if err := deps.AuthStore.Save(context.Background(), auth.Account{Provider: auth.ProviderGitHubCopilot, AccessToken: "token", OAuthClientID: auth.CopilotClientID}); err != nil {
		t.Fatalf("save auth: %v", err)
	}

	code, out, errOut := execute(t, []string{"--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all"}, deps)
	if code != 0 || strings.TrimSpace(out) != "ok" {
		t.Fatalf("run code=%d out=%q err=%q", code, out, errOut)
	}
	if len(runner.seen) != 1 || runner.seen[0].Prompt != "prompt from editor" {
		t.Fatalf("runner saw unexpected request: %+v", runner.seen)
	}
}

func TestCLIEmptyEditorPromptFailsValidation(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	deps := deps(t, repo, runner)
	deps.Editor = func(context.Context) (string, error) { return " \n\t ", nil }

	code, _, errOut := execute(t, []string{"--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all"}, deps)
	if code != 2 || !strings.Contains(errOut, "prompt is required") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	if len(runner.seen) != 0 {
		t.Fatalf("runner should not be called: %+v", runner.seen)
	}
}

func TestCLIEmptyPromptValidatesFlagsBeforeEditor(t *testing.T) {
	clearFastAIDefaults(t)
	repo := tempRepo(t)
	calledEditor := false
	deps := deps(t, repo, &fakeRunner{})
	deps.Editor = func(context.Context) (string, error) {
		calledEditor = true
		return "prompt from editor", nil
	}

	code, _, errOut := execute(t, []string{"--provider", "github-copilot", "--permissions", "all"}, deps)
	if code != 2 || !strings.Contains(errOut, "--model is required") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	if calledEditor {
		t.Fatalf("editor should not open before non-prompt validation succeeds")
	}
}

func TestCLIFlagsOverrideEnvironmentDefaults(t *testing.T) {
	t.Setenv("FASTAI_DEFAULT_PROVIDER", "openai")
	t.Setenv("FASTAI_DEFAULT_MODEL", "env-model")
	t.Setenv("FASTAI_DEFAULT_PERMISSIONS", "all")
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	deps := deps(t, repo, runner)
	if err := deps.AuthStore.Save(context.Background(), auth.Account{Provider: auth.ProviderGitHubCopilot, AccessToken: "token", OAuthClientID: auth.CopilotClientID}); err != nil {
		t.Fatalf("save auth: %v", err)
	}
	code, _, errOut := execute(t, []string{"--provider", "github-copilot", "--model", "github:gpt-5-mini", "--permissions", "execute", "do work"}, deps)
	if code != 0 {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	if len(runner.seen) != 1 {
		t.Fatalf("runner calls = %+v", runner.seen)
	}
	seen := runner.seen[0]
	if seen.Provider != "github-copilot" || seen.Model != "github:gpt-5-mini" || !seen.Permissions.Execute || seen.Permissions.Read || seen.Permissions.Write {
		t.Fatalf("runner saw unexpected request: %+v", seen)
	}
}

func TestCLINoSessionRunDoesNotPersistHistory(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	config := t.TempDir()
	deps := deps(t, repo, runner)
	deps.AuthStore = auth.NewFileStore(filepath.Join(config, "auth.json"))
	deps.SessionService = appsession.NewService(appsession.NewFileStore(filepath.Join(config, "sessions")), deps.Now)
	if err := deps.AuthStore.Save(context.Background(), auth.Account{Provider: auth.ProviderGitHubCopilot, AccessToken: "token", OAuthClientID: auth.CopilotClientID}); err != nil {
		t.Fatalf("save auth: %v", err)
	}

	code, out, errOut := execute(t, []string{"--no-session", "--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all", "small task"}, deps)
	if code != 0 || strings.TrimSpace(out) != "ok" || strings.Contains(errOut, "session:") {
		t.Fatalf("run code=%d out=%q err=%q", code, out, errOut)
	}
	if len(runner.seen) != 1 || runner.seen[0].SessionID == "" {
		t.Fatalf("runner saw unexpected request: %+v", runner.seen)
	}
	sessionsDir := filepath.Join(config, "sessions")
	if entries, err := os.ReadDir(sessionsDir); err == nil && len(entries) != 0 {
		t.Fatalf("expected no persisted sessions, got %d entries in %s", len(entries), sessionsDir)
	}
}

func TestCLINoSessionRejectsExplicitSession(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	code, _, errOut := execute(t, []string{"--no-session", "--session", "follow-up", "--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all", "small task"}, deps(t, repo, runner))
	if code != 2 || !strings.Contains(errOut, "--no-session cannot be used with --session") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	if len(runner.seen) != 0 {
		t.Fatalf("runner should not be called: %+v", runner.seen)
	}
}

func TestCLIGlobalSessionAddsPriorHistoryToPrompt(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "first summary"}}
	deps := deps(t, repo, runner)
	if err := deps.AuthStore.Save(context.Background(), auth.Account{Provider: auth.ProviderGitHubCopilot, AccessToken: "token", OAuthClientID: auth.CopilotClientID}); err != nil {
		t.Fatalf("save auth: %v", err)
	}

	code, _, errOut := execute(t, []string{"--globalSession", "--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all", "first global task"}, deps)
	if code != 0 {
		t.Fatalf("first run code=%d err=%q", code, errOut)
	}
	runner.result.Summary = "second summary"
	code, _, errOut = execute(t, []string{"--globalSession", "--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all", "second global task"}, deps)
	if code != 0 {
		t.Fatalf("second run code=%d err=%q", code, errOut)
	}
	if len(runner.seen) != 2 {
		t.Fatalf("runner calls = %+v", runner.seen)
	}
	secondPrompt := runner.seen[1].Prompt
	for _, want := range []string{"Persisted session context", "first global task", "first summary", "Current request:\nsecond global task"} {
		if !strings.Contains(secondPrompt, want) {
			t.Fatalf("global prompt missing %q:\n%s", want, secondPrompt)
		}
	}
	if runner.seen[1].SessionID != appsession.GlobalSessionID {
		t.Fatalf("expected global session id, got %+v", runner.seen[1])
	}
}

func TestCLIHistoryShowsGlobalSessionByDefault(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{}
	deps := deps(t, repo, runner)
	ctx := context.Background()
	for i := 1; i <= 6; i++ {
		record, _, err := deps.SessionService.Start(ctx, appsession.StartOptions{RepoRoot: repo, SessionID: appsession.GlobalSessionID, Model: "github:gpt-4.1", Prompt: fmt.Sprintf("input %d", i)})
		if err != nil {
			t.Fatalf("start global session %d: %v", i, err)
		}
		if err := deps.SessionService.Complete(ctx, record, fmt.Sprintf("output %d", i)); err != nil {
			t.Fatalf("complete global session %d: %v", i, err)
		}
	}

	code, out, errOut := execute(t, []string{"--history"}, deps)
	if code != 0 || errOut != "" {
		t.Fatalf("history code=%d out=%q err=%q", code, out, errOut)
	}
	for _, want := range []string{
		"Session: global",
		"Showing 5 of 6 conversations.",
		"Input:\ninput 2",
		"Output:\noutput 6",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("history missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "input 1") || strings.Contains(out, "output 1") {
		t.Fatalf("history should default to the last 5 conversations:\n%s", out)
	}
	if len(runner.seen) != 0 {
		t.Fatalf("history should not run the agent: %+v", runner.seen)
	}
}

func TestCLIHistoryAcceptsCountAndNamedSession(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{}
	deps := deps(t, repo, runner)
	ctx := context.Background()
	sessionID := appsession.HashSessionID("feature")
	for i := 1; i <= 3; i++ {
		record, _, err := deps.SessionService.Start(ctx, appsession.StartOptions{RepoRoot: repo, SessionID: sessionID, Model: "github:gpt-4.1", Prompt: fmt.Sprintf("named input %d", i)})
		if err != nil {
			t.Fatalf("start named session %d: %v", i, err)
		}
		if err := deps.SessionService.Complete(ctx, record, fmt.Sprintf("named output %d", i)); err != nil {
			t.Fatalf("complete named session %d: %v", i, err)
		}
	}

	code, out, errOut := execute(t, []string{"--session", "feature", "--history", "2"}, deps)
	if code != 0 || errOut != "" {
		t.Fatalf("history code=%d out=%q err=%q", code, out, errOut)
	}
	for _, want := range []string{
		"Session: " + sessionID,
		"Showing 2 of 3 conversations.",
		"Input:\nnamed input 2",
		"Output:\nnamed output 3",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("history missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "named input 1") || strings.Contains(out, "named output 1") {
		t.Fatalf("history should honor the requested count:\n%s", out)
	}
	if len(runner.seen) != 0 {
		t.Fatalf("history should not run the agent: %+v", runner.seen)
	}
}

func TestCLINewGlobalSessionClearsPriorHistory(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "first summary"}}
	deps := deps(t, repo, runner)
	if err := deps.AuthStore.Save(context.Background(), auth.Account{Provider: auth.ProviderGitHubCopilot, AccessToken: "token", OAuthClientID: auth.CopilotClientID}); err != nil {
		t.Fatalf("save auth: %v", err)
	}

	code, _, errOut := execute(t, []string{"--globalSession", "--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all", "first global task"}, deps)
	if code != 0 {
		t.Fatalf("first run code=%d err=%q", code, errOut)
	}
	code, _, errOut = execute(t, []string{"--newGlobalSession", "--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all", "fresh global task"}, deps)
	if code != 0 {
		t.Fatalf("new global run code=%d err=%q", code, errOut)
	}
	if len(runner.seen) != 2 {
		t.Fatalf("runner calls = %+v", runner.seen)
	}
	if got := runner.seen[1].Prompt; got != "fresh global task" {
		t.Fatalf("expected reset prompt without history, got:\n%s", got)
	}
}

func TestCLIGlobalSessionRejectsOtherSessionModes(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	deps := deps(t, repo, runner)

	code, _, errOut := execute(t, []string{"--globalSession", "--session", "follow-up", "--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all", "small task"}, deps)
	if code != 2 || !strings.Contains(errOut, "--session cannot be used with --globalSession or --newGlobalSession") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	code, _, errOut = execute(t, []string{"--newGlobalSession", "--no-session", "--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all", "small task"}, deps)
	if code != 2 || !strings.Contains(errOut, "--no-session cannot be used with --globalSession or --newGlobalSession") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	if len(runner.seen) != 0 {
		t.Fatalf("runner should not be called: %+v", runner.seen)
	}
}

func TestCLIExplicitSessionAddsPriorHistoryToPrompt(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "named summary"}}
	deps := deps(t, repo, runner)
	if err := deps.AuthStore.Save(context.Background(), auth.Account{Provider: auth.ProviderGitHubCopilot, AccessToken: "token", OAuthClientID: auth.CopilotClientID}); err != nil {
		t.Fatalf("save auth: %v", err)
	}

	code, _, errOut := execute(t, []string{"--session", "feature", "--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all", "first named task"}, deps)
	if code != 0 {
		t.Fatalf("first run code=%d err=%q", code, errOut)
	}
	code, _, errOut = execute(t, []string{"--session", "feature", "--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all", "second named task"}, deps)
	if code != 0 {
		t.Fatalf("second run code=%d err=%q", code, errOut)
	}
	if len(runner.seen) != 2 {
		t.Fatalf("runner calls = %+v", runner.seen)
	}
	secondPrompt := runner.seen[1].Prompt
	for _, want := range []string{"Persisted session context", "first named task", "named summary", "Current request:\nsecond named task"} {
		if !strings.Contains(secondPrompt, want) {
			t.Fatalf("session prompt missing %q:\n%s", want, secondPrompt)
		}
	}
}

func TestCLIMissingModelFailsValidation(t *testing.T) {
	clearFastAIDefaults(t)
	repo := tempRepo(t)
	code, _, errOut := execute(t, []string{"--provider", "github-copilot", "--permissions", "all", "do work"}, deps(t, repo, &fakeRunner{}))
	if code != 2 || !strings.Contains(errOut, "--model is required") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestCLIMissingProviderFailsValidation(t *testing.T) {
	clearFastAIDefaults(t)
	repo := tempRepo(t)
	code, _, errOut := execute(t, []string{"--model", "gpt-4.1", "--permissions", "all", "do work"}, deps(t, repo, &fakeRunner{}))
	if code != 2 || !strings.Contains(errOut, "--provider is required") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestCLIExplicitProviderRun(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "openrouter-token")
	repo := tempRepo(t)
	runner := &fakeRunner{result: agent.Result{Summary: "ok"}}
	code, out, errOut := execute(t, []string{"--provider", "openrouter", "--model", "deepseek/deepseek-chat", "--permissions", "all", "do work"}, deps(t, repo, runner))
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

func clearFastAIDefaults(t *testing.T) {
	t.Helper()
	t.Setenv("FASTAI_DEFAULT_MODEL", "")
	t.Setenv("FASTAI_DEFAULT_PROVIDER", "")
	t.Setenv("FASTAI_DEFAULT_PERMISSIONS", "")
}
