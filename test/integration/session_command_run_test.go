package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/agent/githubmodels"
	"github.com/ericwooley/fastAI/internal/commandexec"
	appsession "github.com/ericwooley/fastAI/internal/session"
	"github.com/ericwooley/fastAI/internal/workspace"
)

func TestResumedSessionCommandExecutionFlow(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	store := appsession.NewFileStore(filepath.Join(t.TempDir(), "sessions"))
	service := appsession.NewService(store, time.Now)
	hashedSessionID := appsession.HashSessionID("follow")
	record, resumed, err := service.Start(context.Background(), appsession.StartOptions{RepoRoot: repo, SessionID: hashedSessionID, Model: "github:gpt-4.1", Prompt: "first"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if resumed || record.SessionID != hashedSessionID {
		t.Fatalf("expected explicit session to auto-create, record=%+v resumed=%v", record, resumed)
	}
	resumedRecord, resumed, err := service.Start(context.Background(), appsession.StartOptions{RepoRoot: repo, SessionID: hashedSessionID, Model: "github:gpt-4.1", Prompt: "run command printf ok"})
	if err != nil || !resumed {
		t.Fatalf("resume: record=%+v resumed=%v err=%v", resumedRecord, resumed, err)
	}
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			t.Fatalf("RunPrompt should not preflight model validation before completion")
		case "/chat/completions":
			callCount++
			w.Header().Set("Content-Type", "application/json")
			if callCount == 1 {
				_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"cmd-1","type":"function","function":{"name":"run_command","arguments":"{\"command\":\"printf ok\"}"}}]}}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"command finished"}}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	adapter := githubmodels.NewValidator(server.Client(), server.URL, "fastAI/test")
	runner := agent.NewLocalRunnerWithPromptRunner(workspace.NewEditor(repo), commandexec.NewExecutor(repo), adapter, adapter)
	result, err := runner.Run(context.Background(), agent.Request{Prompt: resumedRecord.LastPrompt, Model: resumedRecord.Model, SessionID: resumedRecord.SessionID, AccessToken: "token"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.CommandResults) != 1 || result.CommandResults[0].ExitCode != 0 {
		t.Fatalf("unexpected command results: %+v", result.CommandResults)
	}
	if result.Summary != "command finished" {
		t.Fatalf("summary = %q", result.Summary)
	}
}

func TestFailedCommandToolResultContinuesPromptRun(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			t.Fatalf("RunPrompt should not preflight model validation before completion")
		case "/chat/completions":
			callCount++
			w.Header().Set("Content-Type", "application/json")
			if callCount == 1 {
				_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"cmd-1","type":"function","function":{"name":"run_command","arguments":"{\"command\":\"exit 2\"}"}}]}}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"saw the command failure and recovered"}}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	adapter := githubmodels.NewValidator(server.Client(), server.URL, "fastAI/test")
	runner := agent.NewLocalRunnerWithPromptRunner(workspace.NewEditor(repo), commandexec.NewExecutor(repo), adapter, adapter)

	result, err := runner.Run(context.Background(), agent.Request{Prompt: "try a command and keep going", Model: "github:gpt-4.1", SessionID: "s", AccessToken: "token"})
	if err != nil {
		t.Fatalf("run should continue after failed command tool result: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("completion calls = %d, want 2", callCount)
	}
	if len(result.CommandResults) != 1 || result.CommandResults[0].Status != "failed" || result.CommandResults[0].ExitCode != 2 {
		t.Fatalf("unexpected command results: %+v", result.CommandResults)
	}
	if result.Summary != "saw the command failure and recovered" {
		t.Fatalf("summary = %q", result.Summary)
	}
}
