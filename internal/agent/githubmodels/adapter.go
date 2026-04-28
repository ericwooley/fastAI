package githubmodels

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

	appagent "github.com/ericwooley/fastAI/internal/agent"
	"google.golang.org/genai"

	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/runner"
	adksession "google.golang.org/adk/session"
)

const DefaultBaseURL = "https://api.githubcopilot.com"

type Validator struct {
	client    *http.Client
	baseURL   string
	userAgent string
	offline   bool
}

type Option func(*Validator)

func WithOfflineValidation() Option {
	return func(v *Validator) { v.offline = true }
}

func NewValidator(client *http.Client, baseURL string, userAgent string, opts ...Option) *Validator {
	if client == nil {
		client = http.DefaultClient
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if userAgent == "" {
		userAgent = "fastAI/0.1"
	}
	v := &Validator{client: client, baseURL: strings.TrimRight(baseURL, "/"), userAgent: userAgent}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

func (v *Validator) ValidateModel(ctx context.Context, token string, model string) error {
	_, err := v.resolveModel(ctx, token, model)
	return err
}

func (v *Validator) ListModels(ctx context.Context, token string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", v.userAgent)
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub Copilot model list failed: %s", resp.Status)
	}
	var payload struct {
		Data []struct {
			ID                 string `json:"id"`
			Name               string `json:"name"`
			ModelPickerEnabled *bool  `json:"model_picker_enabled"`
			Policy             struct {
				State string `json:"state"`
			} `json:"policy"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		if item.ID == "" {
			continue
		}
		if item.ModelPickerEnabled != nil && !*item.ModelPickerEnabled {
			continue
		}
		if strings.EqualFold(item.Policy.State, "disabled") {
			continue
		}
		models = append(models, item.ID)
	}
	return models, nil
}

type LLM struct {
	validator *Validator
	token     string
	modelName string
}

func (v *Validator) NewLLM(token string, modelName string) *LLM {
	return &LLM{validator: v, token: token, modelName: normalizeModelID(modelName)}
}

func (l *LLM) Name() string {
	if l.modelName == "" {
		return "github-copilot"
	}
	return l.modelName
}

func (l *LLM) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		if stream {
			yield(nil, errors.New("streaming GitHub Copilot responses are not supported yet"))
			return
		}
		modelName := normalizeModelID(req.Model)
		if modelName == "" {
			modelName = l.Name()
		}
		message, err := l.validator.complete(ctx, l.token, modelName, chatCompletionRequest{
			Messages:   messagesFromContents(req.Config, req.Contents),
			Tools:      toolsFromConfig(req.Config),
			ToolChoice: toolChoiceFromConfig(req.Config),
		})
		if err != nil {
			yield(nil, err)
			return
		}
		content, err := contentFromMessage(message)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(&model.LLMResponse{Content: content, TurnComplete: true}, nil)
	}
}

func (v *Validator) RunPrompt(ctx context.Context, req appagent.PromptRunRequest) (string, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return "", errors.New("prompt is required")
	}
	resolvedModel, err := v.resolveModel(ctx, req.AccessToken, req.Model)
	if err != nil {
		return "", err
	}
	llm := v.NewLLM(req.AccessToken, resolvedModel)
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = "copilot-run"
	}
	a, err := llmagent.New(llmagent.Config{
		Name:        "fast_ai_copilot",
		Description: "GitHub Copilot-backed non-interactive coding agent",
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
		return "", errors.New("GitHub Copilot returned no text response")
	}
	return strings.Join(parts, "\n"), nil
}

type chatMessage struct {
	Role          string         `json:"role"`
	Content       any            `json:"content,omitempty"`
	ReasoningText string         `json:"reasoning_text,omitempty"`
	ToolCalls     []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID    string         `json:"tool_call_id,omitempty"`
}

type chatTool struct {
	Type     string           `json:"type"`
	Function chatToolFunction `json:"function"`
}

type chatToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type chatToolCall struct {
	ID       string               `json:"id,omitempty"`
	Type     string               `json:"type,omitempty"`
	Function chatToolCallFunction `json:"function"`
}

type chatToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
}

type chatCompletionRequest struct {
	Messages   []chatMessage
	Tools      []chatTool
	ToolChoice any
}

func (v *Validator) complete(ctx context.Context, token string, modelName string, request chatCompletionRequest) (chatMessage, error) {
	if strings.TrimSpace(token) == "" {
		return chatMessage{}, errors.New("GitHub Copilot token is required")
	}
	if strings.TrimSpace(modelName) == "" {
		return chatMessage{}, errors.New("model is required")
	}
	if len(request.Messages) == 0 {
		return chatMessage{}, errors.New("prompt is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	payload := struct {
		Model      string        `json:"model"`
		Messages   []chatMessage `json:"messages"`
		Tools      []chatTool    `json:"tools,omitempty"`
		ToolChoice any           `json:"tool_choice,omitempty"`
		Stream     bool          `json:"stream"`
	}{Model: normalizeModelID(modelName), Messages: request.Messages, Tools: request.Tools, ToolChoice: request.ToolChoice, Stream: false}
	body, err := json.Marshal(payload)
	if err != nil {
		return chatMessage{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return chatMessage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", v.userAgent)
	req.Header.Set("Openai-Intent", "conversation-edits")
	req.Header.Set("x-initiator", "agent")
	resp, err := v.client.Do(req)
	if err != nil {
		return chatMessage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return chatMessage{}, fmt.Errorf("GitHub Copilot completion failed: %s", resp.Status)
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
		if hasMessageContent(choice.Message) {
			return choice.Message, nil
		}
	}
	return chatMessage{}, errors.New("GitHub Copilot returned no completion choices")
}

func messagesFromContents(config *genai.GenerateContentConfig, contents []*genai.Content) []chatMessage {
	messages := make([]chatMessage, 0, len(contents)+1)
	if config != nil && config.SystemInstruction != nil {
		if system := contentText(config.SystemInstruction, false); system != "" {
			messages = append(messages, chatMessage{Role: "system", Content: system})
		}
	}
	for _, content := range contents {
		if content == nil {
			continue
		}
		role := roleForContent(content)
		text := contentText(content, false)
		reasoning := contentText(content, true)
		if role == "assistant" {
			toolCalls, err := toolCallsFromContent(content)
			if err != nil {
				continue
			}
			if text != "" || reasoning != "" || len(toolCalls) > 0 {
				messages = append(messages, chatMessage{Role: role, Content: text, ReasoningText: reasoning, ToolCalls: toolCalls})
			}
			continue
		}
		if text != "" {
			messages = append(messages, chatMessage{Role: role, Content: text})
		}
		for _, part := range content.Parts {
			if part == nil || part.FunctionResponse == nil {
				continue
			}
			toolContent, err := json.Marshal(part.FunctionResponse.Response)
			if err != nil {
				continue
			}
			messages = append(messages, chatMessage{
				Role:       "tool",
				ToolCallID: part.FunctionResponse.ID,
				Content:    string(toolContent),
			})
		}
	}
	return messages
}

func contentFromMessage(message chatMessage) (*genai.Content, error) {
	parts := make([]*genai.Part, 0, len(message.ToolCalls)+2)
	if reasoning := strings.TrimSpace(message.ReasoningText); reasoning != "" {
		parts = append(parts, &genai.Part{Text: reasoning, Thought: true})
	}
	if text := extractMessageText(message.Content); text != "" {
		parts = append(parts, genai.NewPartFromText(text))
	}
	for index, toolCall := range message.ToolCalls {
		args := map[string]any{}
		if strings.TrimSpace(toolCall.Function.Arguments) != "" {
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				return nil, err
			}
		}
		id := strings.TrimSpace(toolCall.ID)
		if id == "" {
			id = fmt.Sprintf("fastai-tool-%d", index+1)
		}
		parts = append(parts, &genai.Part{FunctionCall: &genai.FunctionCall{ID: id, Name: toolCall.Function.Name, Args: args}})
	}
	if len(parts) == 0 {
		return nil, errors.New("GitHub Copilot returned an empty message")
	}
	return genai.NewContentFromParts(parts, genai.RoleModel), nil
}

func toolsFromConfig(config *genai.GenerateContentConfig) []chatTool {
	if config == nil {
		return nil
	}
	var tools []chatTool
	for _, tool := range config.Tools {
		if tool == nil {
			continue
		}
		for _, declaration := range tool.FunctionDeclarations {
			if declaration == nil || declaration.Name == "" {
				continue
			}
			tools = append(tools, chatTool{
				Type: "function",
				Function: chatToolFunction{
					Name:        declaration.Name,
					Description: declaration.Description,
					Parameters:  declaration.ParametersJsonSchema,
				},
			})
		}
	}
	return tools
}

func toolChoiceFromConfig(config *genai.GenerateContentConfig) any {
	if config == nil || len(config.Tools) == 0 || config.ToolConfig == nil || config.ToolConfig.FunctionCallingConfig == nil {
		if config != nil && len(config.Tools) > 0 {
			return "auto"
		}
		return nil
	}
	calling := config.ToolConfig.FunctionCallingConfig
	switch calling.Mode {
	case genai.FunctionCallingConfigModeNone:
		return "none"
	case genai.FunctionCallingConfigModeAny, genai.FunctionCallingConfigModeValidated:
		if len(calling.AllowedFunctionNames) == 1 {
			return map[string]any{"type": "function", "function": map[string]any{"name": calling.AllowedFunctionNames[0]}}
		}
		return "required"
	default:
		return "auto"
	}
}

func toolCallsFromContent(content *genai.Content) ([]chatToolCall, error) {
	var toolCalls []chatToolCall
	for index, part := range content.Parts {
		if part == nil || part.FunctionCall == nil {
			continue
		}
		args, err := json.Marshal(part.FunctionCall.Args)
		if err != nil {
			return nil, err
		}
		id := strings.TrimSpace(part.FunctionCall.ID)
		if id == "" {
			id = fmt.Sprintf("fastai-tool-%d", index+1)
		}
		toolCalls = append(toolCalls, chatToolCall{
			ID:   id,
			Type: "function",
			Function: chatToolCallFunction{
				Name:      part.FunctionCall.Name,
				Arguments: string(args),
			},
		})
	}
	return toolCalls, nil
}

func roleForContent(content *genai.Content) string {
	role := strings.TrimSpace(content.Role)
	if role == "" {
		return "user"
	}
	if role == string(genai.RoleModel) {
		return "assistant"
	}
	return role
}

func contentText(content *genai.Content, thought bool) string {
	var parts []string
	for _, part := range content.Parts {
		if part == nil || part.Thought != thought {
			continue
		}
		if text := strings.TrimSpace(part.Text); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func extractMessageText(content any) string {
	switch value := content.(type) {
	case string:
		return strings.TrimSpace(value)
	case []any:
		var parts []string
		for _, item := range value {
			if text, ok := item.(string); ok {
				if trimmed := strings.TrimSpace(text); trimmed != "" {
					parts = append(parts, trimmed)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func hasMessageContent(message chatMessage) bool {
	return extractMessageText(message.Content) != "" || strings.TrimSpace(message.ReasoningText) != "" || len(message.ToolCalls) > 0
}

type modelDescriptor struct {
	ID   string
	Name string
}

func (v *Validator) resolveModel(ctx context.Context, token string, requested string) (string, error) {
	requested = sanitizeRequestedModel(requested)
	if requested == "" {
		return "", errors.New("model is required")
	}
	if strings.TrimSpace(token) == "" {
		return "", errors.New("GitHub Copilot token is required")
	}
	if v.offline {
		return requested, nil
	}
	models, err := v.listModelDescriptors(ctx, token)
	if err != nil {
		return "", err
	}
	wantedKeys := comparableModelKeys(requested)
	for _, available := range models {
		for key := range comparableModelKeys(available.ID) {
			if _, ok := wantedKeys[key]; ok {
				return available.ID, nil
			}
		}
		for key := range comparableModelKeys(available.Name) {
			if _, ok := wantedKeys[key]; ok {
				return available.ID, nil
			}
		}
	}
	return "", fmt.Errorf("model %q is not available for this account", requested)
}

func (v *Validator) listModelDescriptors(ctx context.Context, token string) ([]modelDescriptor, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("GitHub Copilot token is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", v.userAgent)
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub Copilot model list failed: %s", resp.Status)
	}
	var payload struct {
		Data []struct {
			ID                 string `json:"id"`
			Name               string `json:"name"`
			ModelPickerEnabled *bool  `json:"model_picker_enabled"`
			Policy             struct {
				State string `json:"state"`
			} `json:"policy"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	models := make([]modelDescriptor, 0, len(payload.Data))
	for _, item := range payload.Data {
		if item.ID == "" {
			continue
		}
		if item.ModelPickerEnabled != nil && !*item.ModelPickerEnabled {
			continue
		}
		if strings.EqualFold(item.Policy.State, "disabled") {
			continue
		}
		models = append(models, modelDescriptor{ID: item.ID, Name: item.Name})
	}
	return models, nil
}

func sanitizeRequestedModel(modelName string) string {
	return strings.TrimPrefix(strings.TrimSpace(modelName), "github:")
}

func normalizeModelID(modelName string) string {
	return sanitizeRequestedModel(modelName)
}

func comparableModelKeys(modelName string) map[string]struct{} {
	trimmed := strings.ToLower(strings.TrimSpace(modelName))
	keys := map[string]struct{}{}
	if trimmed == "" {
		return keys
	}
	keys[trimmed] = struct{}{}
	if strings.HasPrefix(trimmed, "github:") {
		keys[strings.TrimPrefix(trimmed, "github:")] = struct{}{}
	}
	if strings.HasPrefix(trimmed, "openai/") {
		keys[strings.TrimPrefix(trimmed, "openai/")] = struct{}{}
	}
	return keys
}
