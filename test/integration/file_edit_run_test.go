package integration_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/agent/githubmodels"
	"github.com/ericwooley/fastAI/internal/commandexec"
	"github.com/ericwooley/fastAI/internal/workspace"
)

func TestWorkspaceEditingToolOrchestration(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	runner := agent.NewLocalRunner(workspace.NewEditor(repo), commandexec.NewExecutor(repo), githubmodels.NewValidator(nil, "", "", githubmodels.WithOfflineValidation()))
	result, err := runner.Run(context.Background(), agent.Request{Prompt: "create file notes.txt with content hello", Model: "github:gpt-4.1", SessionID: "s", AccessToken: "token"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.FileChanges) != 1 || result.FileChanges[0].Path != "notes.txt" {
		t.Fatalf("unexpected file changes: %+v", result.FileChanges)
	}
	data, err := os.ReadFile(filepath.Join(repo, "notes.txt"))
	if err != nil {
		t.Fatalf("read notes: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("content = %q", data)
	}

	_, err = runner.Run(context.Background(), agent.Request{Prompt: "create file ../blocked.txt with content no", Model: "github:gpt-4.1", SessionID: "s", AccessToken: "token"})
	if !errors.Is(err, workspace.ErrUnsafePath) {
		t.Fatalf("expected unsafe path, got %v", err)
	}
}
