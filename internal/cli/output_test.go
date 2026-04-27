package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ericwooley/fastAI/internal/agent"
)

func TestFormatRunSuccessWritesOnlyModelOutputToStdout(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	var errOut bytes.Buffer
	result := agent.Result{
		Summary:   "Why don't scientists trust atoms?\nBecause they make up everything.",
		SessionID: "session-1",
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
	for _, want := range []string{"session: session-1", "model: gpt-5-mini", "file: applied update joke.txt (+42 bytes)", "command: go test ./... exit=0 status=succeeded"} {
		if !strings.Contains(errOut.String(), want) {
			t.Fatalf("stderr missing %q in %q", want, errOut.String())
		}
	}
}
