package cli

import (
	"errors"
	"fmt"
	"io"
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
