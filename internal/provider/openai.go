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
	reasoningKey string
}

type OpenAIOption func(*OpenAIProvider)

func WithExtraHeaders(headers map[string]string) OpenAIOption {
	return func(p *OpenAIProvider) { p.extraHeaders = headers }
}

func WithReasoningKey(reasoningKey string) OpenAIOption {
	return func(p *OpenAIProvider) { p.reasoningKey = strings.TrimSpace(reasoningKey) }
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

func (p *OpenAIProvider) RunPrompt(ctx context.Context, req agent.PromptRunRequest) (agent.PromptRunResult, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return agent.PromptRunResult{}, errors.New("prompt is required")
	}
	llm := newOpenAILLM(p, req.Model)
	llm.progress = req.Progress
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
		return agent.PromptRunResult{}, err
	}
	service := adksession.InMemoryService()
	if _, err := service.Create(ctx, &adksession.CreateRequest{AppName: "fastAI", UserID: "local", SessionID: sessionID}); err != nil {
		return agent.PromptRunResult{}, err
	}
	r, err := runner.New(runner.Config{AppName: "fastAI", Agent: a, SessionService: service})
	if err != nil {
		return agent.PromptRunResult{}, err
	}
	var parts []string
	for event, err := range r.Run(ctx, "local", sessionID, genai.NewContentFromText(req.Prompt, genai.RoleUser), adkagent.RunConfig{}) {
		if err != nil {
			return agent.PromptRunResult{Telemetry: llm.telemetry}, err
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
		return agent.PromptRunResult{Telemetry: llm.telemetry}, errors.New("model returned no text response")
	}
	return agent.PromptRunResult{Text: strings.Join(parts, "\n"), Telemetry: llm.telemetry}, nil
}

type openAILLM struct {
	provider  *OpenAIProvider
	modelName string
	telemetry agent.ProviderTelemetry
	progress  func(agent.ProviderRequestTelemetry)
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
		message, telemetry, err := l.provider.complete(ctx, modelName, chatCompletionRequest{
			Messages:   messagesFromContents(req.Config, req.Contents, l.provider.reasoningKey),
			Tools:      toolsFromConfig(req.Config),
			ToolChoice: toolChoiceFromConfig(req.Config),
		})
		l.telemetry.Requests = append(l.telemetry.Requests, telemetry)
		if l.progress != nil {
			l.progress(telemetry)
		}
		if err != nil {
			yield(nil, err)
			return
		}
		content, err := contentFromMessage(message, l.provider.reasoningKey)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(&model.LLMResponse{Content: content, TurnComplete: true}, nil)
	}
}

func (p *OpenAIProvider) complete(ctx context.Context, modelName string, request chatCompletionRequest) (chatMessage, agent.ProviderRequestTelemetry, error) {
	telemetry := agent.ProviderRequestTelemetry{Provider: "openai-compatible", Model: strings.TrimSpace(modelName), Endpoint: "/chat/completions"}
	if strings.TrimSpace(p.apiKey) == "" {
		return chatMessage{}, telemetry, errors.New("API key is required")
	}
	if strings.TrimSpace(modelName) == "" {
		return chatMessage{}, telemetry, errors.New("model is required")
	}
	if len(request.Messages) == 0 {
		return chatMessage{}, telemetry, errors.New("prompt is required")
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
		return chatMessage{}, telemetry, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return chatMessage{}, telemetry, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", p.userAgent)
	for k, v := range p.extraHeaders {
		req.Header.Set(k, v)
	}
	started := time.Now()
	resp, err := p.client.Do(req)
	telemetry.Duration = time.Since(started)
	if err != nil {
		return chatMessage{}, telemetry, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody struct{ Error struct{ Message string } }
		if json.NewDecoder(resp.Body).Decode(&errBody) == nil && errBody.Error.Message != "" {
			return chatMessage{}, telemetry, fmt.Errorf("API error (%d): %s", resp.StatusCode, errBody.Error.Message)
		}
		return chatMessage{}, telemetry, fmt.Errorf("API request failed: %s", resp.Status)
	}
	var response struct {
		Choices []struct {
			Message chatMessage `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Usage map[string]any `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return chatMessage{}, telemetry, err
	}
	telemetry.Usage = response.Usage
	if response.Error.Message != "" {
		return chatMessage{}, telemetry, errors.New(response.Error.Message)
	}
	for _, choice := range response.Choices {
		if hasMessageContent(choice.Message, "") {
			telemetry.ToolCalls = toolCallTelemetry(choice.Message.ToolCalls)
			return choice.Message, telemetry, nil
		}
	}
	return chatMessage{}, telemetry, errors.New("API returned no completion choices")
}

func toolCallTelemetry(calls []chatToolCall) []agent.ToolCallTelemetry {
	items := make([]agent.ToolCallTelemetry, 0, len(calls))
	for _, call := range calls {
		items = append(items, agent.ToolCallTelemetry{ID: strings.TrimSpace(call.ID), Name: strings.TrimSpace(call.Function.Name), Arguments: strings.TrimSpace(call.Function.Arguments)})
	}
	return items
}
