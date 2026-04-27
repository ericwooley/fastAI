package cli

import (
	"strings"

	"github.com/ericwooley/fastAI/internal/session"
)

type RunInput struct {
	Prompt    string
	Model     string
	SessionID string
}

func ValidateRunInput(input RunInput) error {
	if strings.TrimSpace(input.Prompt) == "" {
		return NewError(ExitValidation, "prompt is required", "Pass the task as a quoted argument, for example: fastAI --model github:gpt-4.1 'Do the task'.")
	}
	if strings.TrimSpace(input.Model) == "" {
		return NewError(ExitValidation, "--model is required", "Retry with --model <model>.")
	}
	if strings.TrimSpace(input.SessionID) != "" {
		if err := session.ValidateSessionID(input.SessionID); err != nil {
			return NewError(ExitValidation, "--session must contain only letters, numbers, '.', '_', or '-'", "Retry with a filesystem-safe session identifier.")
		}
	}
	return nil
}
