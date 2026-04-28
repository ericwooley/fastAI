package provider

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/ericwooley/fastAI/internal/agent"
)

type Info struct {
	ID      string
	Name    string
	EnvKey  string
	BaseURL string
}

type Factory func(client *http.Client, apiKey string, info Info) (agent.PromptRunner, error)

var known = map[string]Info{
	"github-copilot": {ID: "github-copilot", Name: "GitHub Copilot", EnvKey: "GITHUB_TOKEN", BaseURL: "https://api.githubcopilot.com"},
	"openai":         {ID: "openai", Name: "OpenAI", EnvKey: "OPENAI_API_KEY", BaseURL: "https://api.openai.com/v1"},
	"openrouter":     {ID: "openrouter", Name: "OpenRouter", EnvKey: "OPENROUTER_API_KEY", BaseURL: "https://openrouter.ai/api/v1"},
	"deepseek":       {ID: "deepseek", Name: "DeepSeek", EnvKey: "DEEPSEEK_API_KEY", BaseURL: "https://api.deepseek.com/v1"},
}

func Known() []Info {
	result := make([]Info, 0, len(known))
	for _, info := range known {
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func Lookup(id string) (Info, error) {
	id = strings.TrimSpace(id)
	info, ok := known[id]
	if !ok {
		return Info{}, fmt.Errorf("unknown provider %q", id)
	}
	return info, nil
}
