package cli

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/auth"
	"github.com/ericwooley/fastAI/internal/commandexec"
	"github.com/ericwooley/fastAI/internal/session"
	"github.com/ericwooley/fastAI/internal/workspace"
)

type ExitCode int

const (
	ExitSuccess    ExitCode = 0
	ExitAgent      ExitCode = 1
	ExitValidation ExitCode = 2
	ExitAuth       ExitCode = 3
	ExitBlocked    ExitCode = 4
)

type Error struct {
	Code     ExitCode
	Message  string
	Recovery string
	Cause    error
}

func NewError(code ExitCode, message string, recovery string) *Error {
	return &Error{Code: code, Message: message, Recovery: recovery}
}

func WrapError(code ExitCode, message string, recovery string, cause error) *Error {
	return &Error{Code: code, Message: message, Recovery: recovery, Cause: cause}
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Cause }

func CodeForError(err error) ExitCode {
	if err == nil {
		return ExitSuccess
	}
	var cliErr *Error
	if errors.As(err, &cliErr) {
		return cliErr.Code
	}
	if errors.Is(err, auth.ErrNoAccount) || errors.Is(err, auth.ErrExpired) || errors.Is(err, auth.ErrClientID) {
		return ExitAuth
	}
	if errors.Is(err, workspace.ErrUnsafePath) || errors.Is(err, commandexec.ErrBlocked) {
		return ExitBlocked
	}
	if errors.Is(err, session.ErrInvalidSessionID) || errors.Is(err, session.ErrSessionNotFound) || errors.Is(err, session.ErrRepoMismatch) {
		return ExitValidation
	}
	return ExitAgent
}

func WrapRunError(err error) error {
	switch CodeForError(err) {
	case ExitAuth:
		return WrapError(ExitAuth, "authentication required", "Run `fastAI login` and retry.", err)
	case ExitBlocked:
		return WrapError(ExitBlocked, "unsafe repository operation blocked", "Keep file and command operations inside the active repository.", err)
	case ExitValidation:
		return WrapError(ExitValidation, "validation failed", "Fix the command input and retry.", err)
	default:
		return WrapError(ExitAgent, "agent execution failed", "Review the error and retry when the repository is ready.", err)
	}
}

func FormatError(w io.Writer, err error) {
	if err == nil || w == nil {
		return
	}
	var cliErr *Error
	if errors.As(err, &cliErr) {
		fmt.Fprintf(w, "error: %s\n", cliErr.Message)
		if cliErr.Cause != nil {
			fmt.Fprintf(w, "detail: %s\n", cliErr.Cause)
		}
		if cliErr.Recovery != "" {
			fmt.Fprintf(w, "recovery: %s\n", cliErr.Recovery)
		}
		return
	}
	fmt.Fprintf(w, "error: %s\n", err)
}

func FormatLoginSuccess(w io.Writer, account auth.Account) {
	if w == nil {
		return
	}
	login := account.Login
	if login == "" {
		login = "GitHub Copilot account"
	}
	fmt.Fprintf(w, "Login succeeded for %s.\n", login)
}

func FormatRunSuccess(out io.Writer, thinking io.Writer, result agent.Result) {
	writeStderr(thinking, "run completed successfully")
	writeStderr(thinking, "session: %s", result.SessionID)
	writeStderr(thinking, "provider: %s", result.Provider)
	writeStderr(thinking, "model: %s", result.Model)
	if out != nil && strings.TrimSpace(result.Summary) != "" {
		fmt.Fprintln(out, result.Summary)
	}
	for _, change := range result.FileChanges {
		writeStderr(thinking, "file: %s %s %s (%+d bytes)", change.Status, change.Operation, change.Path, change.BytesChanged)
	}
	for _, command := range result.CommandResults {
		writeStderr(thinking, "command: %s exit=%d status=%s", command.CommandLine, command.ExitCode, command.Status)
		if command.Stdout != "" {
			writeStderr(thinking, "stdout: %s", command.Stdout)
		}
		if command.Stderr != "" {
			writeStderr(thinking, "stderr: %s", command.Stderr)
		}
	}
}

func newTelemetryProgress(w io.Writer) func(agent.ProviderRequestTelemetry) {
	var count int
	return func(request agent.ProviderRequestTelemetry) {
		count++
		formatTelemetryRequest(w, count, request)
	}
}

func formatTelemetryRequest(w io.Writer, number int, request agent.ProviderRequestTelemetry) {
	writeStderr(w, "request: #%d provider=%s model=%s endpoint=%s duration=%s", number, request.Provider, request.Model, request.Endpoint, request.Duration)
	if usage := formatUsage(request.Usage); usage != "" {
		writeStderr(w, "tokens: %s", usage)
	}
	for _, call := range request.ToolCalls {
		line := fmt.Sprintf("tool call: %s", call.Name)
		if strings.TrimSpace(call.ID) != "" {
			line += " id=" + strings.TrimSpace(call.ID)
		}
		if strings.TrimSpace(call.Arguments) != "" {
			line += " args=" + strings.TrimSpace(call.Arguments)
		}
		writeStderr(w, "%s", line)
	}
}

func formatUsage(usage map[string]any) string {
	if len(usage) == 0 {
		return ""
	}
	values := flattenUsage("", usage)
	parts := make([]string, 0, len(values))
	for key, value := range values {
		parts = append(parts, key+"="+value)
	}
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

func flattenUsage(prefix string, usage map[string]any) map[string]string {
	values := map[string]string{}
	for key, raw := range usage {
		name := key
		if prefix != "" {
			name = prefix + "." + key
		}
		switch value := raw.(type) {
		case map[string]any:
			for nestedKey, nestedValue := range flattenUsage(name, value) {
				values[nestedKey] = nestedValue
			}
		case float64:
			values[name] = fmt.Sprintf("%.0f", value)
		case float32:
			values[name] = fmt.Sprintf("%.0f", value)
		case int:
			values[name] = fmt.Sprintf("%d", value)
		case int64:
			values[name] = fmt.Sprintf("%d", value)
		case string:
			if strings.TrimSpace(value) != "" {
				values[name] = strings.TrimSpace(value)
			}
		case bool:
			values[name] = fmt.Sprintf("%t", value)
		}
	}
	return values
}

func writeStderr(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, format+"\n", args...)
}

func writeThinking(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "thinking: "+format+"\n", args...)
}
