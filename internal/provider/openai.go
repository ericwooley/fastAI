package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"strings"
	"time"

	"github.com/ericwooley/fastAI/internal/agent"
	"google.golang.org/genai"

	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/runner"
	adksession "google.golang.org/adk/session"
)

type OpenAIProvider struct {
	client       *http.Client
	apiKey       string
	baseURL      string
	userAgent    string
	extraHeaders map[string]string
}

type OpenAIOption func(*OpenAIProvider)

func WithExtraHeaders(headers map[string]string) OpenAIOption {
	return func(p *OpenAIProvider) { p.extraHeaders = headers }
}

func NewOpenAI(client *http.Client, apiKey string, baseURL string, userAgent string, opts ...OpenAIOption) *OpenAIProvider {
	if client == nil {
		client = http.DefaultClient
	}
	if userAgent == "" {
		userAgent = "fastAI/0.1"
	}
	p := &OpenAIProvider{
		client:    client,
		apiKey:    apiKey,
		baseURL:   strings.TrimRight(baseURL, "/"),
		userAgent: userAgent,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *OpenAIProvider) BaseURL() string { return p.baseURL }

func (p *OpenAIProvider) RunPrompt(ctx context.Context, req agent.PromptRunRequest) (string, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return "", errors.New("prompt is required")
	}
	llm := newOpenAILLM(p, req.Model)
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = "fastai-run"
	}
	a, err := llmagent.New(llmagent.Config{
		Name:        "fast_ai",
		Description: "Non-interactive coding agent",
		Model:       llm,
		Instruction: strings.TrimSpace(req.Instruction),
		Tools:       req.Tools,
	})
	if err != nil {
		return "", err
	}
	service := adksession.InMemoryService()
	if _, err := service.Create(ctx, &adksession.CreateRequest{AppName: "fastAI", UserID: "local", SessionID: sessionID}); err != nil {
		return "", err
	}
	r, err := runner.New(runner.Config{AppName: "fastAI", Agent: a, SessionService: service})
	if err != nil {
		return "", err
	}
	var parts []string
	for event, err := range r.Run(ctx, "local", sessionID, genai.NewContentFromText(req.Prompt, genai.RoleUser), adkagent.RunConfig{}) {
		if err != nil {
			return "", err
		}
		if event == nil || event.LLMResponse.Content == nil {
			continue
		}
		for _, part := range event.LLMResponse.Content.Parts {
			if part.Thought {
				continue
			}
			if strings.TrimSpace(part.Text) != "" {
				parts = append(parts, strings.TrimSpace(part.Text))
			}
		}
	}
	if len(parts) == 0 {
		return "", errors.New("model returned no text response")
	}
	return strings.Join(parts, "\n"), nil
}

type openAILLM struct {
	provider  *OpenAIProvider
	modelName string
}

func newOpenAILLM(p *OpenAIProvider, modelName string) *openAILLM {
	return &openAILLM{provider: p, modelName: modelName}
}

func (l *openAILLM) Name() string {
	if l.modelName != "" {
		return l.modelName
	}
	return "openai"
}

func (l *openAILLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		if stream {
			yield(nil, errors.New("streaming is not supported yet"))
			return
		}
		modelName := req.Model
		if modelName == "" {
			modelName = l.modelName
		}
		message, err := l.provider.complete(ctx, modelName, chatCompletionRequest{
			Messages:   messagesFromContents(req.Config, req.Contents, ""),
			Tools:      toolsFromConfig(req.Config),
			ToolChoice: toolChoiceFromConfig(req.Config),
		})
		if err != nil {
			yield(nil, err)
			return
		}
		content, err := contentFromMessage(message, "")
		if err != nil {
			yield(nil, err)
			return
		}
		yield(&model.LLMResponse{Content: content, TurnComplete: true}, nil)
	}
}

func (p *OpenAIProvider) complete(ctx context.Context, modelName string, request chatCompletionRequest) (chatMessage, error) {
	if strings.TrimSpace(p.apiKey) == "" {
		return chatMessage{}, errors.New("API key is required")
	}
	if strings.TrimSpace(modelName) == "" {
		return chatMessage{}, errors.New("model is required")
	}
	if len(request.Messages) == 0 {
		return chatMessage{}, errors.New("prompt is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	payload := struct {
		Model      string        `json:"model"`
		Messages   []chatMessage `json:"messages"`
		Tools      []chatTool    `json:"tools,omitempty"`
		ToolChoice any           `json:"tool_choice,omitempty"`
		Stream     bool          `json:"stream"`
	}{Model: modelName, Messages: request.Messages, Tools: request.Tools, ToolChoice: request.ToolChoice, Stream: false}
	body, err := json.Marshal(payload)
	if err != nil {
		return chatMessage{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return chatMessage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", p.userAgent)
	for k, v := range p.extraHeaders {
		req.Header.Set(k, v)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return chatMessage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody struct{ Error struct{ Message string } }
		if json.NewDecoder(resp.Body).Decode(&errBody) == nil && errBody.Error.Message != "" {
			return chatMessage{}, fmt.Errorf("API error (%d): %s", resp.StatusCode, errBody.Error.Message)
		}
		return chatMessage{}, fmt.Errorf("API request failed: %s", resp.Status)
	}
	var response struct {
		Choices []struct {
			Message chatMessage `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return chatMessage{}, err
	}
	if response.Error.Message != "" {
		return chatMessage{}, errors.New(response.Error.Message)
	}
	for _, choice := range response.Choices {
		if hasMessageContent(choice.Message, "") {
			return choice.Message, nil
		}
	}
	return chatMessage{}, errors.New("API returned no completion choices")
}
