package integration_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ericwooley/fastAI/internal/agent/githubmodels"
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

func TestADKAdapterRunsCopilotCompletionThroughADK(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"openai/gpt-4.1","name":"gpt-4.1","model_picker_enabled":true}]}`))
			return
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
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			Stream bool `json:"stream"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.Model != "openai/gpt-4.1" || payload.Stream {
			t.Fatalf("unexpected payload model/stream: %+v", payload)
		}
		if len(payload.Messages) == 0 || payload.Messages[len(payload.Messages)-1].Content != "summarize this" {
			t.Fatalf("unexpected messages: %+v", payload.Messages)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"completed through ADK"}}]}`))
	}))
	defer server.Close()

	adapter := githubmodels.NewValidator(server.Client(), server.URL, "fastAI/test")
	text, err := adapter.RunPrompt(context.Background(), "token", "github:gpt-4.1", "summarize this")
	if err != nil {
		t.Fatalf("run prompt: %v", err)
	}
	if text != "completed through ADK" {
		t.Fatalf("unexpected text: %q", text)
	}
}
