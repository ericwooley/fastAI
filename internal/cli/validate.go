package cli

import (
	"os"
	"strings"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/provider"
	"github.com/ericwooley/fastAI/internal/session"
)

type RunInput struct {
	Prompt      string
	Model       string
	SessionID   string
	Provider    string
	Permissions string
}

type ResolvedRunInput struct {
	Prompt      string
	Model       string
	SessionID   string
	Provider    string
	Permissions agent.Permissions
}

func ResolveRunInput(input RunInput) (ResolvedRunInput, error) {
	input.Model = valueOrEnv(input.Model, "FASTAI_DEFAULT_MODEL")
	input.Provider = valueOrEnv(input.Provider, "FASTAI_DEFAULT_PROVIDER")
	input.Permissions = valueOrEnv(input.Permissions, "FASTAI_DEFAULT_PERMISSIONS")

	permissions, err := parsePermissions(input.Permissions)
	if err != nil {
		return ResolvedRunInput{}, err
	}
	input.Model = strings.TrimSpace(input.Model)
	input.Provider = strings.TrimSpace(input.Provider)
	input.SessionID = session.HashSessionID(input.SessionID)
	return ResolvedRunInput{Prompt: input.Prompt, Model: input.Model, SessionID: input.SessionID, Provider: input.Provider, Permissions: permissions}, nil
}

func ValidateRunInput(input ResolvedRunInput) error {
	if strings.TrimSpace(input.Prompt) == "" {
		return NewError(ExitValidation, "prompt is required", "Pass the task as a quoted argument, for example: fastAI --provider openai --model gpt-4.1 'Do the task'.")
	}
	if strings.TrimSpace(input.Model) == "" {
		return NewError(ExitValidation, "--model is required", "Retry with --model <model>.")
	}
	if strings.TrimSpace(input.Provider) == "" {
		return NewError(ExitValidation, "--provider is required", "Retry with --provider <provider>. Valid providers: "+strings.Join(providerIDs(), ", "))
	}
	if !input.Permissions.Set {
		return NewError(ExitValidation, "--permissions is required", "Retry with --permissions=all or set FASTAI_DEFAULT_PERMISSIONS=all.")
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

func parsePermissions(value string) (agent.Permissions, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return agent.Permissions{}, nil
	}

	permissions := agent.Permissions{Set: true}
	seen := map[string]bool{}
	for _, raw := range strings.Split(value, ",") {
		part := strings.ToLower(strings.TrimSpace(raw))
		if part == "" {
			return agent.Permissions{}, permissionsError()
		}
		seen[part] = true
		if (seen["none"] || seen["all"]) && len(seen) > 1 {
			return agent.Permissions{}, permissionsError()
		}
		switch part {
		case "read":
			permissions.Read = true
		case "write":
			permissions.Write = true
		case "execute":
			permissions.Execute = true
		case "all":
			permissions = agent.AllPermissions()
		case "none":
			permissions.Read = false
			permissions.Write = false
			permissions.Execute = false
		default:
			return agent.Permissions{}, permissionsError()
		}
	}
	return permissions, nil
}

func permissionsError() error {
	return NewError(ExitValidation, "--permissions must be a comma-separated list of read, write, execute, all, or none", "Retry with --permissions=read,write, --permissions=all, or --permissions=none.")
}

func valueOrEnv(value string, envKey string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return os.Getenv(envKey)
}

func providerIDs() []string {
	known := provider.Known()
	ids := make([]string, len(known))
	for i, info := range known {
		ids[i] = info.ID
	}
	return ids
}
