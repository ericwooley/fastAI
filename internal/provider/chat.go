package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

type chatMessage struct {
	Role             string         `json:"role"`
	Content          any            `json:"content,omitempty"`
	ReasoningText    string         `json:"reasoning_text,omitempty"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	ToolCalls        []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
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

func messagesFromContents(config *genai.GenerateContentConfig, contents []*genai.Content, reasoningField string) []chatMessage {
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
				msg := chatMessage{Role: role, Content: text, ReasoningText: reasoning, ToolCalls: toolCalls}
				messages = append(messages, msg)
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
	_ = reasoningField
	return messages
}

func contentFromMessage(message chatMessage, reasoningField string) (*genai.Content, error) {
	parts := make([]*genai.Part, 0, len(message.ToolCalls)+2)
	if reasoning := strings.TrimSpace(reasoningText(message, reasoningField)); reasoning != "" {
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
		return nil, fmt.Errorf("empty response from model")
	}
	return genai.NewContentFromParts(parts, genai.RoleModel), nil
}

func reasoningText(message chatMessage, _ string) string {
	if r := strings.TrimSpace(message.ReasoningContent); r != "" {
		return r
	}
	return strings.TrimSpace(message.ReasoningText)
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

func hasMessageContent(message chatMessage, reasoningField string) bool {
	return extractMessageText(message.Content) != "" || strings.TrimSpace(reasoningText(message, reasoningField)) != "" || len(message.ToolCalls) > 0
}
