package agent

import (
	"context"
	"errors"
)

var ErrExecution = errors.New("agent execution failed")

type Request struct {
	Prompt      string
	Model       string
	SessionID   string
	RepoRoot    string
	AccessToken string
}

type Result struct {
	Summary        string
	SessionID      string
	Model          string
	FileChanges    []FileChange
	CommandResults []CommandResult
}

type FileChange struct {
	Path         string
	Operation    string
	Status       string
	Reason       string
	BytesChanged int64
}

type CommandResult struct {
	CommandLine string
	WorkingDir  string
	ExitCode    int
	Stdout      string
	Stderr      string
	Status      string
}

type Runner interface {
	Run(context.Context, Request) (Result, error)
}

type ModelValidator interface {
	ValidateModel(context.Context, string, string) error
}

type PromptRunner interface {
	RunPrompt(context.Context, string, string, string) (string, error)
}
