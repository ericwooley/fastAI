package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/agent/githubmodels"
	"github.com/ericwooley/fastAI/internal/commandexec"
	"github.com/ericwooley/fastAI/internal/workspace"
)

func TestCLIFileEditingAndBlockedPath(t *testing.T) {
	t.Parallel()
	repo := tempRepo(t)
	deps := deps(t, repo, agent.NewLocalRunner(workspace.NewEditor(repo), commandexec.NewExecutor(repo), githubmodels.NewValidator(nil, "", "", githubmodels.WithOfflineValidation())))
	code, out, errOut := execute(t, []string{"login", "copilot"}, deps)
	if code != 0 {
		t.Fatalf("login code=%d out=%q err=%q", code, out, errOut)
	}
	code, out, errOut = execute(t, []string{"--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all", "create file note.txt with content hi"}, deps)
	if code != 0 || out != "" || !strings.Contains(errOut, "file: applied update note.txt") {
		t.Fatalf("edit code=%d out=%q err=%q", code, out, errOut)
	}
	if strings.Contains(errOut, "thinking: ") {
		t.Fatalf("stderr unexpectedly contains thinking prefix: %q", errOut)
	}
	data, err := os.ReadFile(filepath.Join(repo, "note.txt"))
	if err != nil || string(data) != "hi" {
		t.Fatalf("file content=%q err=%v", data, err)
	}
	code, _, errOut = execute(t, []string{"--provider", "github-copilot", "--model", "github:gpt-4.1", "--permissions", "all", "create file ../bad.txt with content no"}, deps)
	if code != 4 || !strings.Contains(errOut, "unsafe repository operation blocked") {
		t.Fatalf("blocked code=%d err=%q", code, errOut)
	}
}
