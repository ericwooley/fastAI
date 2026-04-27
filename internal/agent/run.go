package agent

import (
	"context"
	"fmt"
	"iter"
	"regexp"
	"strings"

	adkagent "google.golang.org/adk/agent"
	adksession "google.golang.org/adk/session"

	"github.com/ericwooley/fastAI/internal/commandexec"
	"github.com/ericwooley/fastAI/internal/workspace"
)

type WorkspaceEditor interface {
	Apply(context.Context, workspace.Operation) (workspace.Change, error)
}

type CommandExecutor interface {
	Execute(context.Context, commandexec.Request) (commandexec.Result, error)
}

type LocalRunner struct {
	editor       WorkspaceEditor
	executor     CommandExecutor
	validator    ModelValidator
	promptRunner PromptRunner
}

func NewLocalRunner(editor WorkspaceEditor, executor CommandExecutor, validator ModelValidator) *LocalRunner {
	return &LocalRunner{editor: editor, executor: executor, validator: validator}
}

func NewLocalRunnerWithPromptRunner(editor WorkspaceEditor, executor CommandExecutor, validator ModelValidator, promptRunner PromptRunner) *LocalRunner {
	return &LocalRunner{editor: editor, executor: executor, validator: validator, promptRunner: promptRunner}
}

func (r *LocalRunner) Run(ctx context.Context, req Request) (Result, error) {
	if r.validator != nil {
		if err := r.validator.ValidateModel(ctx, req.AccessToken, req.Model); err != nil {
			return Result{SessionID: req.SessionID, Model: req.Model}, fmt.Errorf("%w: %v", ErrExecution, err)
		}
	}
	if err := touchADK(ctx, req.SessionID); err != nil {
		return Result{SessionID: req.SessionID, Model: req.Model}, fmt.Errorf("%w: %v", ErrExecution, err)
	}

	result := Result{SessionID: req.SessionID, Model: req.Model}
	for _, op := range parseFileOperations(req.Prompt) {
		change, err := r.editor.Apply(ctx, op)
		result.FileChanges = append(result.FileChanges, FileChange{
			Path:         change.Path,
			Operation:    change.Operation,
			Status:       change.Status,
			Reason:       change.Reason,
			BytesChanged: change.BytesChanged,
		})
		if err != nil {
			return result, err
		}
	}
	for _, commandLine := range parseCommands(req.Prompt) {
		commandResult, err := r.executor.Execute(ctx, commandexec.Request{CommandLine: commandLine})
		result.CommandResults = append(result.CommandResults, CommandResult{
			CommandLine: commandResult.CommandLine,
			WorkingDir:  commandResult.WorkingDir,
			ExitCode:    commandResult.ExitCode,
			Stdout:      commandResult.StdoutSummary,
			Stderr:      commandResult.StderrSummary,
			Status:      commandResult.Status,
		})
		if err != nil {
			return result, fmt.Errorf("%w: command %q exited with status %d", ErrExecution, commandLine, commandResult.ExitCode)
		}
	}
	if len(result.FileChanges) == 0 && len(result.CommandResults) == 0 && r.promptRunner != nil {
		text, err := r.promptRunner.RunPrompt(ctx, req.AccessToken, req.Model, req.Prompt)
		if err != nil {
			return result, fmt.Errorf("%w: %v", ErrExecution, err)
		}
		result.Summary = text
		return result, nil
	}
	return result, nil
}

func touchADK(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		sessionID = "session"
	}
	customAgent, err := adkagent.New(adkagent.Config{
		Name:        "fastAI",
		Description: "Repository-safe Copilot CLI agent",
		Run: func(adkagent.InvocationContext) iter.Seq2[*adksession.Event, error] {
			return func(yield func(*adksession.Event, error) bool) {}
		},
	})
	if err != nil {
		return err
	}
	if customAgent.Name() == "" {
		return fmt.Errorf("ADK agent was not initialized")
	}
	service := adksession.InMemoryService()
	_, err = service.Create(ctx, &adksession.CreateRequest{AppName: "fastAI", UserID: "local", SessionID: sessionID})
	return err
}

var (
	writeFilePattern  = regexp.MustCompile(`(?is)\b(?:create|write|update)\s+file\s+([^\s]+)\s+with\s+content\s+(.+?)(?:\s+and\s+|$)`)
	deleteFilePattern = regexp.MustCompile(`(?is)\bdelete\s+file\s+([^\s]+)`)
	runCommandPattern = regexp.MustCompile(`(?is)\b(?:run|execute)\s+command\s+(.+?)(?:\s+and\s+then\s+|$)`)
)

func parseFileOperations(prompt string) []workspace.Operation {
	var ops []workspace.Operation
	for _, match := range writeFilePattern.FindAllStringSubmatch(prompt, -1) {
		ops = append(ops, workspace.Operation{Type: workspace.OperationUpdate, Path: cleanToken(match[1]), Content: strings.TrimSpace(match[2])})
	}
	for _, match := range deleteFilePattern.FindAllStringSubmatch(prompt, -1) {
		ops = append(ops, workspace.Operation{Type: workspace.OperationDelete, Path: cleanToken(match[1])})
	}
	return ops
}

func parseCommands(prompt string) []string {
	matches := runCommandPattern.FindAllStringSubmatch(prompt, -1)
	commands := make([]string, 0, len(matches))
	for _, match := range matches {
		commands = append(commands, strings.Trim(strings.TrimSpace(match[1]), "`"))
	}
	return commands
}

func cleanToken(s string) string {
	return strings.Trim(strings.TrimSpace(s), "`'\"")
}
