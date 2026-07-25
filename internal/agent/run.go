package agent

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	adktool "google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"

	"github.com/ericwooley/fastAI/internal/commandexec"
	"github.com/ericwooley/fastAI/internal/workspace"
)

type WorkspaceEditor interface {
	Apply(context.Context, workspace.Operation) (workspace.Change, error)
	Read(context.Context, string) (workspace.ReadResult, error)
	Patch(context.Context, workspace.PatchOperation) (workspace.Change, error)
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
	result := Result{SessionID: req.SessionID, Model: req.Model, Provider: req.Provider}
	permissions := effectivePermissions(req.Permissions)
	runPromptRunner := r.promptRunner
	if req.PromptRunner != nil {
		runPromptRunner = req.PromptRunner
	}
	if runPromptRunner != nil {
		tools, err := r.buildTools(ctx, &result, permissions)
		if err != nil {
			return result, fmt.Errorf("%w: %v", ErrExecution, err)
		}
		instruction := defaultInstruction(permissions)
		if strings.TrimSpace(req.RepoRoot) != "" {
			agentInstructions, err := loadAgentInstructions(req.RepoRoot, req.WorkingDirectory, repositoryInstructionReader(req.RepoRoot))
			if err != nil {
				return result, fmt.Errorf("%w: load AGENTS.md instructions: %v", ErrExecution, err)
			}
			instruction = appendAgentInstructions(instruction, agentInstructions)
		}
		promptResult, err := runPromptRunner.RunPrompt(ctx, PromptRunRequest{
			AccessToken: req.AccessToken,
			Model:       req.Model,
			Prompt:      req.Prompt,
			SessionID:   req.SessionID,
			Instruction: instruction,
			Tools:       tools,
			Progress:    req.Progress,
		})
		result.Telemetry = promptResult.Telemetry
		if strings.TrimSpace(promptResult.Text) != "" {
			result.Summary = strings.TrimSpace(promptResult.Text)
		}
		if err != nil {
			return result, fmt.Errorf("%w: %v", ErrExecution, err)
		}
		return result, nil
	}
	if r.validator != nil {
		if err := r.validator.ValidateModel(ctx, req.AccessToken, req.Model); err != nil {
			return Result{SessionID: req.SessionID, Model: req.Model, Provider: req.Provider}, fmt.Errorf("%w: %v", ErrExecution, err)
		}
	}
	if permissions.Write {
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
	}
	if permissions.Execute {
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
	}
	return result, nil
}

func repositoryInstructionReader(repoRoot string) readInstructionFile {
	return func(path string) ([]byte, error) {
		_, safePath, err := workspace.NormalizeRepoPath(repoRoot, path)
		if err != nil {
			return nil, err
		}
		return os.ReadFile(safePath)
	}
}

type writeFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type deleteFileArgs struct {
	Path string `json:"path"`
}

type readFileArgs struct {
	Path string `json:"path"`
}

type patchFileArgs struct {
	Path       string `json:"path"`
	Old        string `json:"old"`
	New        string `json:"new"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

type commandArgs struct {
	Command          string `json:"command"`
	WorkingDirectory string `json:"working_directory,omitempty"`
}

type fileToolResult struct {
	Path         string `json:"path"`
	Operation    string `json:"operation"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	BytesChanged int64  `json:"bytes_changed"`
}

type commandToolResult struct {
	CommandLine string `json:"command_line"`
	WorkingDir  string `json:"working_directory"`
	ExitCode    int    `json:"exit_code"`
	Stdout      string `json:"stdout,omitempty"`
	Stderr      string `json:"stderr,omitempty"`
	Status      string `json:"status"`
	Reason      string `json:"reason,omitempty"`
}

type readFileToolResult struct {
	Path    string `json:"path"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Content string `json:"content,omitempty"`
	Bytes   int64  `json:"bytes"`
}

func effectivePermissions(permissions Permissions) Permissions {
	if !permissions.Set {
		return AllPermissions()
	}
	return permissions
}

func (r *LocalRunner) buildTools(ctx context.Context, result *Result, permissions Permissions) ([]adktool.Tool, error) {
	writeTool, err := functiontool.New(functiontool.Config{
		Name:        "write_file",
		Description: "Create or update a file inside the current repository. Use this for all file edits.",
	}, func(_ adktool.Context, args writeFileArgs) (fileToolResult, error) {
		change, err := r.editor.Apply(ctx, workspace.Operation{Type: workspace.OperationUpdate, Path: args.Path, Content: args.Content})
		result.FileChanges = append(result.FileChanges, FileChange{Path: change.Path, Operation: change.Operation, Status: change.Status, Reason: change.Reason, BytesChanged: change.BytesChanged})
		if err != nil {
			return fileToolResult{Path: change.Path, Operation: change.Operation, Status: change.Status, Reason: change.Reason, BytesChanged: change.BytesChanged}, nil
		}
		return fileToolResult{Path: change.Path, Operation: change.Operation, Status: change.Status, Reason: change.Reason, BytesChanged: change.BytesChanged}, nil
	})
	if err != nil {
		return nil, err
	}

	deleteTool, err := functiontool.New(functiontool.Config{
		Name:        "delete_file",
		Description: "Delete a file inside the current repository.",
	}, func(_ adktool.Context, args deleteFileArgs) (fileToolResult, error) {
		change, err := r.editor.Apply(ctx, workspace.Operation{Type: workspace.OperationDelete, Path: args.Path})
		result.FileChanges = append(result.FileChanges, FileChange{Path: change.Path, Operation: change.Operation, Status: change.Status, Reason: change.Reason, BytesChanged: change.BytesChanged})
		if err != nil {
			return fileToolResult{Path: change.Path, Operation: change.Operation, Status: change.Status, Reason: change.Reason, BytesChanged: change.BytesChanged}, nil
		}
		return fileToolResult{Path: change.Path, Operation: change.Operation, Status: change.Status, Reason: change.Reason, BytesChanged: change.BytesChanged}, nil
	})
	if err != nil {
		return nil, err
	}

	readTool, err := functiontool.New(functiontool.Config{
		Name:        "read_file",
		Description: "Read a file inside the current repository. Use this before changing existing code so you can inspect current contents.",
	}, func(_ adktool.Context, args readFileArgs) (readFileToolResult, error) {
		result, err := r.editor.Read(ctx, args.Path)
		if err != nil {
			return readFileToolResult{Path: result.Path, Status: result.Status, Reason: result.Reason, Content: result.Content, Bytes: result.Bytes}, nil
		}
		return readFileToolResult{Path: result.Path, Status: result.Status, Reason: result.Reason, Content: result.Content, Bytes: result.Bytes}, nil
	})
	if err != nil {
		return nil, err
	}

	patchTool, err := functiontool.New(functiontool.Config{
		Name:        "patch_file",
		Description: "Apply a targeted text replacement to an existing file inside the current repository. Use this for precise edits instead of rewriting whole files when possible.",
	}, func(_ adktool.Context, args patchFileArgs) (fileToolResult, error) {
		change, err := r.editor.Patch(ctx, workspace.PatchOperation{Path: args.Path, Old: args.Old, New: args.New, ReplaceAll: args.ReplaceAll})
		result.FileChanges = append(result.FileChanges, FileChange{Path: change.Path, Operation: change.Operation, Status: change.Status, Reason: change.Reason, BytesChanged: change.BytesChanged})
		if err != nil {
			return fileToolResult{Path: change.Path, Operation: change.Operation, Status: change.Status, Reason: change.Reason, BytesChanged: change.BytesChanged}, nil
		}
		return fileToolResult{Path: change.Path, Operation: change.Operation, Status: change.Status, Reason: change.Reason, BytesChanged: change.BytesChanged}, nil
	})
	if err != nil {
		return nil, err
	}

	commandTool, err := functiontool.New(functiontool.Config{
		Name:        "run_command",
		Description: "Run a shell command inside the current repository. Use this for repository inspection, tests, builds, or other command-line tasks.",
	}, func(_ adktool.Context, args commandArgs) (commandToolResult, error) {
		commandResult, err := r.executor.Execute(ctx, commandexec.Request{CommandLine: args.Command, WorkingDir: args.WorkingDirectory})
		result.CommandResults = append(result.CommandResults, CommandResult{CommandLine: commandResult.CommandLine, WorkingDir: commandResult.WorkingDir, ExitCode: commandResult.ExitCode, Stdout: commandResult.StdoutSummary, Stderr: commandResult.StderrSummary, Status: commandResult.Status})
		toolResult := commandToolResult{CommandLine: commandResult.CommandLine, WorkingDir: commandResult.WorkingDir, ExitCode: commandResult.ExitCode, Stdout: commandResult.StdoutSummary, Stderr: commandResult.StderrSummary, Status: commandResult.Status}
		if err != nil {
			toolResult.Reason = err.Error()
			return toolResult, nil
		}
		return toolResult, nil
	})
	if err != nil {
		return nil, err
	}

	tools := make([]adktool.Tool, 0, 5)
	if permissions.Read {
		tools = append(tools, readTool)
	}
	if permissions.Write {
		tools = append(tools, writeTool, patchTool, deleteTool)
	}
	if permissions.Execute {
		tools = append(tools, commandTool)
	}
	return tools, nil
}

func defaultInstruction(permissions Permissions) string {
	if !permissions.Read && !permissions.Write && !permissions.Execute {
		return strings.TrimSpace(`You are fastAI, a non-interactive repository coding agent.

You have no repository tools available in this run. Answer using only the prompt context, and do not claim that you can read files, write files, or run commands.

Keep the final response concise and task-focused.`)
	}

	toolText := ""
	if permissions.Read {
		toolText += "reading files"
	}
	if permissions.Write {
		if toolText != "" {
			toolText += ", "
		}
		toolText += "writing files, applying targeted patches, and deleting files"
	}
	if permissions.Execute {
		if toolText != "" {
			toolText += ", "
		}
		toolText += "running commands"
	}

	return strings.TrimSpace(`You are fastAI, a non-interactive repository coding agent.

You have real repository-safe tools available for ` + toolText + ` inside the current repository.

Use available tools whenever the user asks about repository state, file contents, code changes, or command output. Read existing files before modifying them when current contents matter and read access is available, and prefer targeted patches over full rewrites when write access is available. Do not claim that you have unavailable file access, command execution, or tool access.

When the user asks what tools you have, describe the actual available tools and their repository-safe constraints.

Keep the final response concise and task-focused.`)
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
