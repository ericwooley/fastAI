package commandexec

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExecutorClassifiesResults(t *testing.T) {
	t.Parallel()
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	executor := NewExecutor(repo)

	result, err := executor.Execute(context.Background(), Request{CommandLine: successCommand()})
	if err != nil {
		t.Fatalf("success command: %v", err)
	}
	if result.ExitCode != 0 || result.Status != "succeeded" || result.StdoutSummary == "" {
		t.Fatalf("unexpected success result: %+v", result)
	}

	result, err = executor.Execute(context.Background(), Request{CommandLine: failureCommand()})
	if err == nil {
		t.Fatalf("expected command failure")
	}
	if result.ExitCode == 0 || result.Status != "failed" {
		t.Fatalf("unexpected failure result: %+v", result)
	}

	result, err = executor.Execute(context.Background(), Request{CommandLine: successCommand(), WorkingDir: ".."})
	if !errors.Is(err, ErrBlocked) || result.Status != "blocked" {
		t.Fatalf("expected blocked command, got result=%+v err=%v", result, err)
	}
}

func successCommand() string {
	if runtime.GOOS == "windows" {
		return "echo ok"
	}
	return "printf ok"
}

func failureCommand() string {
	if runtime.GOOS == "windows" {
		return "exit 7"
	}
	return "exit 7"
}
