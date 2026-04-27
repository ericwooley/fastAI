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

	"google.golang.org/genai"

	"google.golang.org/adk/agent"
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
		text, err := l.validator.complete(ctx, l.token, modelName, messagesFromContents(req.Contents))
		if err != nil {
			yield(nil, err)
			return
		}
		yield(&model.LLMResponse{Content: genai.NewContentFromText(text, genai.RoleModel), TurnComplete: true}, nil)
	}
}

func (v *Validator) RunPrompt(ctx context.Context, token string, modelName string, prompt string) (string, error) {
	if strings.TrimSpace(prompt) == "" {
		return "", errors.New("prompt is required")
	}
	resolvedModel, err := v.resolveModel(ctx, token, modelName)
	if err != nil {
		return "", err
	}
	llm := v.NewLLM(token, resolvedModel)
	a, err := llmagent.New(llmagent.Config{
		Name:        "fast_ai_copilot",
		Description: "GitHub Copilot-backed non-interactive coding agent",
		Model:       llm,
		Instruction: "You are fastAI, a non-interactive repository coding agent. Return a concise final result for the requested task.",
	})
	if err != nil {
		return "", err
	}
	service := adksession.InMemoryService()
	if _, err := service.Create(ctx, &adksession.CreateRequest{AppName: "fastAI", UserID: "local", SessionID: "copilot-run"}); err != nil {
		return "", err
	}
	r, err := runner.New(runner.Config{AppName: "fastAI", Agent: a, SessionService: service})
	if err != nil {
		return "", err
	}
	var parts []string
	for event, err := range r.Run(ctx, "local", "copilot-run", genai.NewContentFromText(prompt, genai.RoleUser), agent.RunConfig{}) {
		if err != nil {
			return "", err
		}
		if event == nil || event.LLMResponse.Content == nil {
			continue
		}
		for _, part := range event.LLMResponse.Content.Parts {
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
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (v *Validator) complete(ctx context.Context, token string, modelName string, messages []chatMessage) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", errors.New("GitHub Copilot token is required")
	}
	if strings.TrimSpace(modelName) == "" {
		return "", errors.New("model is required")
	}
	if len(messages) == 0 {
		return "", errors.New("prompt is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	payload := struct {
		Model    string        `json:"model"`
		Messages []chatMessage `json:"messages"`
		Stream   bool          `json:"stream"`
	}{Model: normalizeModelID(modelName), Messages: messages, Stream: false}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", v.userAgent)
	req.Header.Set("Openai-Intent", "conversation-edits")
	req.Header.Set("x-initiator", "agent")
	resp, err := v.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GitHub Copilot completion failed: %s", resp.Status)
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
		return "", err
	}
	if response.Error.Message != "" {
		return "", errors.New(response.Error.Message)
	}
	for _, choice := range response.Choices {
		if strings.TrimSpace(choice.Message.Content) != "" {
			return strings.TrimSpace(choice.Message.Content), nil
		}
	}
	return "", errors.New("GitHub Copilot returned no completion choices")
}

func messagesFromContents(contents []*genai.Content) []chatMessage {
	messages := make([]chatMessage, 0, len(contents))
	for _, content := range contents {
		if content == nil {
			continue
		}
		var parts []string
		for _, part := range content.Parts {
			if strings.TrimSpace(part.Text) != "" {
				parts = append(parts, strings.TrimSpace(part.Text))
			}
		}
		if len(parts) == 0 {
			continue
		}
		role := string(content.Role)
		if role == "" {
			role = "user"
		}
		if role == string(genai.RoleModel) {
			role = "assistant"
		}
		messages = append(messages, chatMessage{Role: role, Content: strings.Join(parts, "\n")})
	}
	return messages
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
