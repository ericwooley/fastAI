package provider

import (
	"fmt"
	"net/http"

	"github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/agent/githubmodels"
)

func NewPromptRunner(client *http.Client, providerID string, apiKey string) (agent.PromptRunner, error) {
	info, err := Lookup(providerID)
	if err != nil {
		return nil, err
	}
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required for provider %q (set %s or use --provider)", info.ID, info.EnvKey)
	}
	switch info.ID {
	case "github-copilot":
		return githubmodels.NewValidator(client, info.BaseURL, "fastAI/0.1"), nil
	case "deepseek":
		return NewOpenAI(client, apiKey, info.BaseURL, "fastAI/0.1", WithReasoningKey("reasoning_content")), nil
	case "openrouter":
		return NewOpenAI(client, apiKey, info.BaseURL, "fastAI/0.1", WithExtraHeaders(map[string]string{
			"HTTP-Referer": "https://github.com/ericwooley/fastAI",
			"X-Title":      "fastAI",
		})), nil
	default:
		return NewOpenAI(client, apiKey, info.BaseURL, "fastAI/0.1"), nil
	}
}
