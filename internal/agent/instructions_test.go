package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentInstructionPathsAreOrderedFromRepoRootToWorkingDirectory(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Join(string(filepath.Separator), "repo")
	workingDir := filepath.Join(repoRoot, "services", "api")

	got, err := agentInstructionPaths(repoRoot, workingDir)
	if err != nil {
		t.Fatalf("instruction paths: %v", err)
	}

	want := []string{
		filepath.Join(repoRoot, "AGENTS.md"),
		filepath.Join(repoRoot, "services", "AGENTS.md"),
		filepath.Join(repoRoot, "services", "api", "AGENTS.md"),
	}
	if len(got) != len(want) {
		t.Fatalf("paths = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("paths[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestAgentInstructionPathsRejectWorkingDirectoryOutsideRepo(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Join(string(filepath.Separator), "repo")
	_, err := agentInstructionPaths(repoRoot, filepath.Join(string(filepath.Separator), "other"))
	if err == nil {
		t.Fatal("expected working directory outside repository to fail")
	}
}

func TestLoadAgentInstructionsSkipsMissingFilesAndPreservesOrder(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Join(string(filepath.Separator), "repo")
	workingDir := filepath.Join(repoRoot, "services", "api")
	contents := map[string]string{
		filepath.Join(repoRoot, "AGENTS.md"):                    "root guidance",
		filepath.Join(repoRoot, "services", "api", "AGENTS.md"): "api guidance",
	}

	got, err := loadAgentInstructions(repoRoot, workingDir, func(path string) ([]byte, error) {
		content, ok := contents[path]
		if !ok {
			return nil, os.ErrNotExist
		}
		return []byte(content), nil
	})
	if err != nil {
		t.Fatalf("load instructions: %v", err)
	}

	want := []agentInstruction{
		{Path: "AGENTS.md", Content: "root guidance"},
		{Path: "services/api/AGENTS.md", Content: "api guidance"},
	}
	if len(got) != len(want) {
		t.Fatalf("instructions = %#v, want %#v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("instructions[%d] = %#v, want %#v", index, got[index], want[index])
		}
	}
}

func TestLoadAgentInstructionsReturnsNonMissingReadErrors(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Join(string(filepath.Separator), "repo")
	readErr := errors.New("permission denied")
	_, err := loadAgentInstructions(repoRoot, repoRoot, func(string) ([]byte, error) {
		return nil, readErr
	})
	if !errors.Is(err, readErr) {
		t.Fatalf("error = %v, want %v", err, readErr)
	}
}

func TestLoadAgentInstructionsSkipsWhitespaceOnlyFiles(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Join(string(filepath.Separator), "repo")
	got, err := loadAgentInstructions(repoRoot, repoRoot, func(string) ([]byte, error) {
		return []byte(" \n\t "), nil
	})
	if err != nil {
		t.Fatalf("load instructions: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("instructions = %#v, want none", got)
	}
}

func TestAppendAgentInstructionsExplainsLeafPrecedence(t *testing.T) {
	t.Parallel()

	got := appendAgentInstructions("base instruction", []agentInstruction{
		{Path: "AGENTS.md", Content: "root guidance"},
		{Path: "services/api/AGENTS.md", Content: "api guidance"},
	})

	for _, want := range []string{
		"base instruction",
		"later file takes priority",
		"closer to the current working directory",
		"## AGENTS.md",
		"root guidance",
		"## services/api/AGENTS.md",
		"api guidance",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("instruction missing %q:\n%s", want, got)
		}
	}
	if strings.Index(got, "root guidance") > strings.Index(got, "api guidance") {
		t.Fatalf("root guidance must precede api guidance:\n%s", got)
	}
}

type instructionCaptureRunner struct {
	instruction string
}

func (r *instructionCaptureRunner) RunPrompt(_ context.Context, req PromptRunRequest) (PromptRunResult, error) {
	r.instruction = req.Instruction
	return PromptRunResult{Text: "ok"}, nil
}

func TestLocalRunnerAppendsAgentInstructionsFromRepoRootToWorkingDirectory(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	workingDir := filepath.Join(repoRoot, "services", "api")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir working directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "AGENTS.md"), []byte("root guidance"), 0o644); err != nil {
		t.Fatalf("write root instructions: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "services", "AGENTS.md"), []byte("service guidance"), 0o644); err != nil {
		t.Fatalf("write service instructions: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workingDir, "AGENTS.md"), []byte("api guidance"), 0o644); err != nil {
		t.Fatalf("write api instructions: %v", err)
	}

	promptRunner := &instructionCaptureRunner{}
	runner := NewLocalRunnerWithPromptRunner(nil, nil, nil, promptRunner)
	_, err := runner.Run(context.Background(), Request{
		Prompt:           "do work",
		Model:            "model",
		RepoRoot:         repoRoot,
		WorkingDirectory: workingDir,
		Permissions:      Permissions{Set: true},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	rootIndex := strings.Index(promptRunner.instruction, "root guidance")
	serviceIndex := strings.Index(promptRunner.instruction, "service guidance")
	apiIndex := strings.Index(promptRunner.instruction, "api guidance")
	if rootIndex < 0 || serviceIndex < 0 || apiIndex < 0 {
		t.Fatalf("missing AGENTS.md content:\n%s", promptRunner.instruction)
	}
	if !(rootIndex < serviceIndex && serviceIndex < apiIndex) {
		t.Fatalf("AGENTS.md content is not ordered root-to-leaf:\n%s", promptRunner.instruction)
	}
}

func TestLocalRunnerRejectsAgentInstructionSymlinkOutsideRepo(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	outsideDirectory := t.TempDir()
	outsideInstructions := filepath.Join(outsideDirectory, "AGENTS.md")
	if err := os.WriteFile(outsideInstructions, []byte("outside guidance"), 0o644); err != nil {
		t.Fatalf("write outside instructions: %v", err)
	}
	if err := os.Symlink(outsideInstructions, filepath.Join(repoRoot, "AGENTS.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	promptRunner := &instructionCaptureRunner{}
	runner := NewLocalRunnerWithPromptRunner(nil, nil, nil, promptRunner)
	_, err := runner.Run(context.Background(), Request{
		Prompt:           "do work",
		Model:            "model",
		RepoRoot:         repoRoot,
		WorkingDirectory: repoRoot,
		Permissions:      Permissions{Set: true},
	})
	if err == nil {
		t.Fatal("expected AGENTS.md symlink outside repository to fail")
	}
	if promptRunner.instruction != "" {
		t.Fatalf("prompt runner should not receive outside instructions:\n%s", promptRunner.instruction)
	}
}
