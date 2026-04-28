package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/agent/githubmodels"
	"github.com/ericwooley/fastAI/internal/commandexec"
	"github.com/ericwooley/fastAI/internal/workspace"
	"net/http"
	"net/http/httptest"
)

func TestWorkspaceEditingToolOrchestration(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	seed := filepath.Join(repo, "main.go")
	if err := os.WriteFile(seed, []byte("package main\n\nfunc main() {\n\tprintln(\"old\")\n}\n"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
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
				_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"read-1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"main.go\"}"}}]}}]}`))
				return
			}
			if callCount == 2 {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}
				var payload struct {
					Messages []struct {
						Role       string `json:"role"`
						Content    any    `json:"content"`
						ToolCallID string `json:"tool_call_id"`
					} `json:"messages"`
				}
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("decode payload: %v", err)
				}
				toolMsg := payload.Messages[len(payload.Messages)-1]
				content, ok := toolMsg.Content.(string)
				if toolMsg.Role != "tool" || toolMsg.ToolCallID != "read-1" || !ok || !strings.Contains(content, "println") {
					t.Fatalf("unexpected read tool message: %+v", toolMsg)
				}
				_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"patch-1","type":"function","function":{"name":"patch_file","arguments":"{\"path\":\"main.go\",\"old\":\"println(\\\"old\\\")\",\"new\":\"println(\\\"new\\\")\"}"}}]}}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"read and patched main.go"}}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := githubmodels.NewValidator(server.Client(), server.URL, "fastAI/test")
	runner := agent.NewLocalRunnerWithPromptRunner(workspace.NewEditor(repo), commandexec.NewExecutor(repo), adapter, adapter)
	result, err := runner.Run(context.Background(), agent.Request{Prompt: "Read main.go and change old to new", Model: "github:gpt-4.1", SessionID: "s", AccessToken: "token"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.FileChanges) != 1 || result.FileChanges[0].Path != "main.go" || result.FileChanges[0].Operation != "patch" {
		t.Fatalf("unexpected file changes: %+v", result.FileChanges)
	}
	data, err := os.ReadFile(seed)
	if err != nil {
		t.Fatalf("read main: %v", err)
	}
	if !strings.Contains(string(data), `println("new")`) {
		t.Fatalf("content = %q", data)
	}
	if result.Summary != "read and patched main.go" {
		t.Fatalf("summary = %q", result.Summary)
	}

	blockedCalls := 0
	blockedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			t.Fatalf("RunPrompt should not preflight model validation before completion")
		case "/chat/completions":
			blockedCalls++
			w.Header().Set("Content-Type", "application/json")
			if blockedCalls == 1 {
				_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"write-2","type":"function","function":{"name":"write_file","arguments":"{\"path\":\"../blocked.txt\",\"content\":\"no\"}"}}]}}]}`))
				return
			}
			_, _ = w.Write([]byte(fmt.Sprintf(`{"choices":[{"message":{"role":"assistant","content":"blocked"}}]}`)))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer blockedServer.Close()
	blockedAdapter := githubmodels.NewValidator(blockedServer.Client(), blockedServer.URL, "fastAI/test")
	blockedRunner := agent.NewLocalRunnerWithPromptRunner(workspace.NewEditor(repo), commandexec.NewExecutor(repo), blockedAdapter, blockedAdapter)
	_, err = blockedRunner.Run(context.Background(), agent.Request{Prompt: "Try to write outside the repo", Model: "github:gpt-4.1", SessionID: "s", AccessToken: "token"})
	if !errors.Is(err, workspace.ErrUnsafePath) {
		t.Fatalf("expected unsafe path, got %v", err)
	}
}
