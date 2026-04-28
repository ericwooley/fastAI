package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/ericwooley/fastAI/internal/agent"
)

func TestFormatRunSuccessWritesOnlyModelOutputToStdout(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	var errOut bytes.Buffer
	result := agent.Result{
		Summary:   "Why don't scientists trust atoms?\nBecause they make up everything.",
		SessionID: "session-1",
		Provider:  "github-copilot",
		Model:     "gpt-5-mini",
		FileChanges: []agent.FileChange{{
			Path:         "joke.txt",
			Operation:    "update",
			Status:       "applied",
			BytesChanged: 42,
		}},
		CommandResults: []agent.CommandResult{{
			CommandLine: "go test ./...",
			ExitCode:    0,
			Status:      "succeeded",
			Stdout:      "ok",
		}},
	}

	FormatRunSuccess(&out, &errOut, result)

	if got, want := out.String(), result.Summary+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	for _, forbidden := range []string{"Run completed successfully", "Session:", "Model:", "Summary:", "File:", "Command:"} {
		if strings.Contains(out.String(), forbidden) {
			t.Fatalf("stdout contains %q: %q", forbidden, out.String())
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(errOut.String()), "\n") {
		if strings.HasPrefix(line, "thinking: ") {
			t.Fatalf("stderr unexpectedly contains thinking prefix: %q in %q", line, errOut.String())
		}
	}
	for _, want := range []string{
		"session: session-1",
		"provider: github-copilot",
		"model: gpt-5-mini",
		"file: applied update joke.txt (+42 bytes)",
		"command: go test ./... exit=0 status=succeeded",
	} {
		if !strings.Contains(errOut.String(), want) {
			t.Fatalf("stderr missing %q in %q", want, errOut.String())
		}
	}
	for _, unwanted := range []string{"requests:", "request: #", "tokens:", "tool call:"} {
		if strings.Contains(errOut.String(), unwanted) {
			t.Fatalf("stderr contains final telemetry %q in %q", unwanted, errOut.String())
		}
	}
}

func TestTelemetryProgressWritesRequestAsItHappens(t *testing.T) {
	t.Parallel()
	var errOut bytes.Buffer
	progress := newTelemetryProgress(&errOut)

	progress(agent.ProviderRequestTelemetry{
		Provider: "github-copilot",
		Model:    "gpt-5-mini",
		Endpoint: "/chat/completions",
		Duration: 150 * time.Millisecond,
		Usage: map[string]any{
			"prompt_tokens":     float64(20),
			"completion_tokens": float64(7),
			"total_tokens":      float64(27),
			"prompt_tokens_details": map[string]any{
				"cached_tokens": float64(8),
			},
		},
		ToolCalls: []agent.ToolCallTelemetry{{ID: "call-1", Name: "read_file", Arguments: `{"path":"main.go"}`}},
	})

	for _, want := range []string{
		"request: #1 provider=github-copilot model=gpt-5-mini endpoint=/chat/completions duration=150ms",
		"tokens: completion_tokens=7 prompt_tokens=20 prompt_tokens_details.cached_tokens=8 total_tokens=27",
		`tool call: read_file id=call-1 args={"path":"main.go"}`,
	} {
		if !strings.Contains(errOut.String(), want) {
			t.Fatalf("stderr missing %q in %q", want, errOut.String())
		}
	}
}
