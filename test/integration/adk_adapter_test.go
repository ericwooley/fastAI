package integration_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appagent "github.com/ericwooley/fastAI/internal/agent"
	"github.com/ericwooley/fastAI/internal/agent/githubmodels"
	adktool "google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

func TestADKAdapterModelValidationFailureHandling(t *testing.T) {
	t.Parallel()
	server := modelServer(t, http.StatusOK, `{"data":[{"id":"enabled","name":"enabled","model_picker_enabled":true},{"id":"openai/gpt-5-mini","name":"gpt-5-mini","model_picker_enabled":true},{"id":"disabled","name":"disabled","model_picker_enabled":true,"policy":{"state":"disabled"}},{"id":"hidden","name":"hidden","model_picker_enabled":false}]}`)
	defer server.Close()
	validator := githubmodels.NewValidator(server.Client(), server.URL, "fastAI/test")
	if err := validator.ValidateModel(context.Background(), "token", "enabled"); err != nil {
		t.Fatalf("enabled model: %v", err)
	}
	if err := validator.ValidateModel(context.Background(), "token", "gpt-5-mini"); err != nil {
		t.Fatalf("alias model: %v", err)
	}
	if err := validator.ValidateModel(context.Background(), "token", "github:gpt-5-mini"); err != nil {
		t.Fatalf("github alias model: %v", err)
	}
	if err := validator.ValidateModel(context.Background(), "token", "disabled"); err == nil {
		t.Fatalf("expected disabled model failure")
	}
	if err := validator.ValidateModel(context.Background(), "", "enabled"); err == nil {
		t.Fatalf("expected missing token failure")
	}
}

func TestADKAdapterRunsToolAwareCompletionThroughADK(t *testing.T) {
	t.Parallel()
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			t.Fatalf("RunPrompt should not preflight model validation before completion")
		}
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("missing auth header: %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Openai-Intent") != "conversation-edits" || r.Header.Get("x-initiator") != "agent" {
			t.Fatalf("missing Copilot headers: intent=%q initiator=%q", r.Header.Get("Openai-Intent"), r.Header.Get("x-initiator"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var payload struct {
			Model    string `json:"model"`
			Messages []struct {
				Role          string `json:"role"`
				Content       any    `json:"content"`
				ReasoningText string `json:"reasoning_text"`
				ToolCalls     []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
				ToolCallID string `json:"tool_call_id"`
			} `json:"messages"`
			Tools []struct {
				Type     string `json:"type"`
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			} `json:"tools"`
			ToolChoice any  `json:"tool_choice"`
			Stream     bool `json:"stream"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.Model != "gpt-4.1" || payload.Stream {
			t.Fatalf("unexpected payload model/stream: %+v", payload)
		}
		callCount++
		switch callCount {
		case 1:
			if len(payload.Tools) != 1 || payload.Tools[0].Function.Name != "echo_tool" {
				t.Fatalf("expected tool declaration, got %+v", payload.Tools)
			}
			if choice, ok := payload.ToolChoice.(string); !ok || choice != "auto" {
				t.Fatalf("unexpected tool choice: %#v", payload.ToolChoice)
			}
			if len(payload.Messages) < 2 || payload.Messages[len(payload.Messages)-1].Role != "user" || payload.Messages[len(payload.Messages)-1].Content != "summarize this" {
				t.Fatalf("unexpected first messages: %+v", payload.Messages)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call-1","type":"function","function":{"name":"echo_tool","arguments":"{\"value\":\"from model\"}"}}]}}]}`))
			return
		case 2:
			if len(payload.Messages) < 4 {
				t.Fatalf("expected assistant and tool messages, got %+v", payload.Messages)
			}
			assistant := payload.Messages[len(payload.Messages)-2]
			toolMsg := payload.Messages[len(payload.Messages)-1]
			if assistant.Role != "assistant" || len(assistant.ToolCalls) != 1 || assistant.ToolCalls[0].Function.Name != "echo_tool" {
				t.Fatalf("unexpected assistant tool call message: %+v", assistant)
			}
			if toolMsg.Role != "tool" || toolMsg.ToolCallID != "call-1" {
				t.Fatalf("unexpected tool response message: %+v", toolMsg)
			}
			content, ok := toolMsg.Content.(string)
			if !ok || !strings.Contains(content, "from model") {
				t.Fatalf("unexpected tool content: %#v", toolMsg.Content)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"completed through ADK tool flow"}}]}`))
			return
		default:
			t.Fatalf("unexpected completion call count: %d", callCount)
		}
		if len(payload.Messages) == 0 {
			t.Fatalf("unexpected messages: %+v", payload.Messages)
		}
	}))
	defer server.Close()

	tool, err := functiontool.New(functiontool.Config{
		Name:        "echo_tool",
		Description: "Echoes the provided value.",
	}, func(_ adktool.Context, args struct {
		Value string `json:"value"`
	}) (map[string]any, error) {
		return map[string]any{"echoed": args.Value}, nil
	})
	if err != nil {
		t.Fatalf("new tool: %v", err)
	}

	adapter := githubmodels.NewValidator(server.Client(), server.URL, "fastAI/test")
	text, err := adapter.RunPrompt(context.Background(), appagent.PromptRunRequest{
		AccessToken: "token",
		Model:       "github:gpt-4.1",
		Prompt:      "summarize this",
		SessionID:   "session-1",
		Instruction: "Use tools when needed.",
		Tools:       []adktool.Tool{tool},
	})
	if err != nil {
		t.Fatalf("run prompt: %v", err)
	}
	if text != "completed through ADK tool flow" {
		t.Fatalf("unexpected text: %q", text)
	}
}
