package commandexec

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ericwooley/fastAI/internal/workspace"
)

var ErrBlocked = errors.New("command blocked by repository boundary")

type Request struct {
	CommandLine string
	WorkingDir  string
	Timeout     time.Duration
}

type Result struct {
	CommandLine   string        `json:"command_line"`
	WorkingDir    string        `json:"working_directory"`
	ExitCode      int           `json:"exit_code"`
	StdoutSummary string        `json:"stdout_summary,omitempty"`
	StderrSummary string        `json:"stderr_summary,omitempty"`
	Duration      time.Duration `json:"duration_ms"`
	Status        string        `json:"status"`
}

type Executor struct {
	repoRoot string
}

func NewExecutor(repoRoot string) *Executor {
	return &Executor{repoRoot: repoRoot}
}

func (e *Executor) Execute(ctx context.Context, req Request) (Result, error) {
	commandLine := strings.TrimSpace(req.CommandLine)
	result := Result{CommandLine: commandLine, ExitCode: -1}
	if commandLine == "" {
		result.Status = "failed"
		return result, errors.New("command line is required")
	}
	workingDir := req.WorkingDir
	if workingDir == "" {
		workingDir = "."
	}
	rel, abs, err := workspace.NormalizeRepoPath(e.repoRoot, workingDir)
	if err != nil {
		result.WorkingDir = workingDir
		result.Status = "blocked"
		return result, errors.Join(ErrBlocked, err)
	}
	result.WorkingDir = rel
	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	name, args := shellCommand(commandLine)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = abs
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	started := time.Now()
	err = cmd.Run()
	result.Duration = time.Since(started)
	result.StdoutSummary = truncate(stdout.String(), 4096)
	result.StderrSummary = truncate(stderr.String(), 4096)
	if err == nil {
		result.ExitCode = 0
		result.Status = "succeeded"
		return result, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
	} else if ctx.Err() != nil {
		result.StderrSummary = ctx.Err().Error()
	}
	result.Status = "failed"
	return result, err
}

func shellCommand(commandLine string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", commandLine}
	}
	return "sh", []string{"-c", commandLine}
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
