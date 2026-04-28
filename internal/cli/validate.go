package cli

import (
	"strings"

	"github.com/ericwooley/fastAI/internal/provider"
	"github.com/ericwooley/fastAI/internal/session"
)

type RunInput struct {
	Prompt    string
	Model     string
	SessionID string
	Provider  string
}

func ResolveRunInput(input RunInput) (RunInput, error) {
	input.Model = strings.TrimSpace(input.Model)
	input.Provider = strings.TrimSpace(input.Provider)
	input.SessionID = session.HashSessionID(input.SessionID)
	return input, nil
}

func ValidateRunInput(input RunInput) error {
	if strings.TrimSpace(input.Prompt) == "" {
		return NewError(ExitValidation, "prompt is required", "Pass the task as a quoted argument, for example: fastAI --provider openai --model gpt-4.1 'Do the task'.")
	}
	if strings.TrimSpace(input.Model) == "" {
		return NewError(ExitValidation, "--model is required", "Retry with --model <model>.")
	}
	if strings.TrimSpace(input.Provider) == "" {
		return NewError(ExitValidation, "--provider is required", "Retry with --provider <provider>. Valid providers: "+strings.Join(providerIDs(), ", "))
	}
	if _, err := provider.Lookup(input.Provider); err != nil {
		return NewError(ExitValidation, "--provider: "+err.Error(), "Valid providers: "+strings.Join(providerIDs(), ", "))
	}
	if strings.TrimSpace(input.SessionID) != "" {
		if err := session.ValidateSessionID(input.SessionID); err != nil {
			return NewError(ExitValidation, "--session must contain only letters, numbers, '.', '_', or '-'", "Retry with a filesystem-safe session identifier.")
		}
	}
	return nil
}

func providerIDs() []string {
	known := provider.Known()
	ids := make([]string, len(known))
	for i, info := range known {
		ids[i] = info.ID
	}
	return ids
}
