package agent

import (
	"context"
	"errors"
	"time"

	adktool "google.golang.org/adk/tool"
)

var ErrExecution = errors.New("agent execution failed")

type Request struct {
	Prompt       string
	Model        string
	SessionID    string
	RepoRoot     string
	AccessToken  string
	Provider     string
	Permissions  Permissions
	PromptRunner PromptRunner
	Progress     func(ProviderRequestTelemetry)
}

type Permissions struct {
	Set     bool
	Read    bool
	Write   bool
	Execute bool
}

func AllPermissions() Permissions {
	return Permissions{Set: true, Read: true, Write: true, Execute: true}
}

type Result struct {
	Summary        string
	SessionID      string
	Model          string
	Provider       string
	FileChanges    []FileChange
	CommandResults []CommandResult
	Telemetry      ProviderTelemetry
}

type ProviderTelemetry struct {
	Requests []ProviderRequestTelemetry
}

type ProviderRequestTelemetry struct {
	Provider  string
	Model     string
	Endpoint  string
	Duration  time.Duration
	Usage     map[string]any
	ToolCalls []ToolCallTelemetry
}

type ToolCallTelemetry struct {
	ID        string
	Name      string
	Arguments string
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
	RunPrompt(context.Context, PromptRunRequest) (PromptRunResult, error)
}

type PromptRunResult struct {
	Text      string
	Telemetry ProviderTelemetry
}

type PromptRunRequest struct {
	AccessToken string
	Model       string
	Prompt      string
	SessionID   string
	Instruction string
	Tools       []adktool.Tool
	Progress    func(ProviderRequestTelemetry)
}
